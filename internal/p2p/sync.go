package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"slices"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/substantialcattle5/sietch/internal/config"
)

const (
	// Protocol IDs for different sync operations
	ManifestProtocolID = "/sietch/manifest/1.0.0"
	ChunkProtocolID    = "/sietch/chunk/1.0.0"
)

// SyncService handles vault synchronization
type SyncService struct {
	host     host.Host
	vaultMgr *config.Manager
}

// SyncResult contains statistics about a sync operation
type SyncResult struct {
	FileCount          int
	ChunksTransferred  int
	ChunksDeduplicated int
	BytesTransferred   int64
	Duration           time.Duration
}

// NewSyncService creates a new sync service
func NewSyncService(h host.Host, vm *config.Manager) (*SyncService, error) {
	s := &SyncService{
		host:     h,
		vaultMgr: vm,
	}

	// Register protocol handlers
	h.SetStreamHandler(protocol.ID(ManifestProtocolID), s.handleManifestRequest)
	h.SetStreamHandler(protocol.ID(ChunkProtocolID), s.handleChunkRequest)

	return s, nil
}

// handleManifestRequest processes requests for vault manifests
func (s *SyncService) handleManifestRequest(stream network.Stream) {
	defer stream.Close()

	// Get our vault manifest
	manifest, err := s.vaultMgr.GetManifest()
	if err != nil {
		fmt.Printf("Error getting manifest: %v\n", err)
		return
	}

	// Encode and send the manifest
	if err := json.NewEncoder(stream).Encode(manifest); err != nil {
		fmt.Printf("Error sending manifest: %v\n", err)
	}
}

// handleChunkRequest processes requests for chunks
func (s *SyncService) handleChunkRequest(stream network.Stream) {
	defer stream.Close()

	// Read the chunk hash
	var chunkRequest struct {
		Hash string `json:"hash"`
	}

	if err := json.NewDecoder(stream).Decode(&chunkRequest); err != nil {
		fmt.Printf("Error reading chunk request: %v\n", err)
		return
	}

	// Get the chunk data
	chunkData, err := s.vaultMgr.GetChunk(chunkRequest.Hash)
	if err != nil {
		// Send error response
		response := struct {
			Error string `json:"error"`
		}{
			Error: "Chunk not found",
		}
		json.NewEncoder(stream).Encode(response)
		return
	}

	// Send the chunk data
	response := struct {
		Size int    `json:"size"`
		Data []byte `json:"data"`
	}{
		Size: len(chunkData),
		Data: chunkData,
	}

	if err := json.NewEncoder(stream).Encode(response); err != nil {
		fmt.Printf("Error sending chunk: %v\n", err)
	}
}

// SyncWithPeer performs a sync operation with a specific peer
func (s *SyncService) SyncWithPeer(ctx context.Context, peerID peer.ID) (*SyncResult, error) {
	startTime := time.Now()
	result := &SyncResult{}

	// Step 1: Get remote manifest
	remoteManifest, err := s.getRemoteManifest(ctx, peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote manifest: %v", err)
	}

	// Step 2: Get local manifest
	localManifest, err := s.vaultMgr.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get local manifest: %v", err)
	}

	// Step 3: Find missing chunks
	missingChunks := s.findMissingChunks(localManifest, remoteManifest)

	// Step 4: Fetch missing chunks
	for _, chunkHash := range missingChunks {
		exists, _ := s.vaultMgr.ChunkExists(chunkHash)
		if exists {
			result.ChunksDeduplicated++
			continue
		}

		chunkData, size, err := s.fetchChunk(ctx, peerID, chunkHash)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chunk %s: %v", chunkHash, err)
		}

		// Store the chunk
		if err := s.vaultMgr.StoreChunk(chunkHash, chunkData); err != nil {
			return nil, fmt.Errorf("failed to store chunk %s: %v", chunkHash, err)
		}

		result.ChunksTransferred++
		result.BytesTransferred += int64(size)
	}

	// Step 5: Update file manifests
	result.FileCount = len(remoteManifest.Files) - len(localManifest.Files)

	// Step 6: Rebuild references
	if err := s.vaultMgr.RebuildReferences(); err != nil {
		return nil, fmt.Errorf("failed to rebuild references: %v", err)
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// getRemoteManifest fetches the manifest from a remote peer
func (s *SyncService) getRemoteManifest(ctx context.Context, peerID peer.ID) (*config.Manifest, error) {
	// Open a stream to the peer
	stream, err := s.host.NewStream(ctx, peerID, protocol.ID(ManifestProtocolID))
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	// Read the manifest
	var manifest config.Manifest
	if err := json.NewDecoder(stream).Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// findMissingChunks identifies chunks that exist in remote but not local manifest
func (s *SyncService) findMissingChunks(local, remote *config.Manifest) []string {
	missingChunks := []string{}

	// Build a map of local chunks for quick lookup
	localChunks := make(map[string]bool)
	for _, file := range local.Files {
		for _, chunk := range file.Chunks {
			localChunks[chunk.Hash] = true
		}
	}

	// Find chunks in remote that don't exist locally
	for _, file := range remote.Files {
		for _, chunk := range file.Chunks {
			if !localChunks[chunk.Hash] {
				// Check if we already added this chunk to the missing list
				alreadyAdded := slices.Contains(missingChunks, chunk.Hash)
				if !alreadyAdded {
					missingChunks = append(missingChunks, chunk.Hash)
				}
			}
		}
	}

	return missingChunks
}

// fetchChunk downloads a chunk from a remote peer
func (s *SyncService) fetchChunk(ctx context.Context, peerID peer.ID, hash string) ([]byte, int, error) {
	// Open a stream to the peer
	stream, err := s.host.NewStream(ctx, peerID, protocol.ID(ChunkProtocolID))
	if err != nil {
		return nil, 0, err
	}
	defer stream.Close()

	// Send chunk request
	request := struct {
		Hash string `json:"hash"`
	}{
		Hash: hash,
	}

	if err := json.NewEncoder(stream).Encode(request); err != nil {
		return nil, 0, err
	}

	// Read response
	var response struct {
		Error string `json:"error,omitempty"`
		Size  int    `json:"size,omitempty"`
		Data  []byte `json:"data,omitempty"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, 0, err
	}

	if response.Error != "" {
		return nil, 0, fmt.Errorf("remote error: %s", response.Error)
	}

	return response.Data, response.Size, nil
}
