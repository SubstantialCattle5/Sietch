package p2p

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
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
	ManifestProtocolID  = "/sietch/manifest/1.0.0"
	ChunkProtocolID     = "/sietch/chunk/1.0.0"
	KeyExchangeProtocol = "/sietch/key-exchange/1.0.0"
	AuthProtocol        = "/sietch/auth/1.0.0"

	// RSA encryption chunk size (must be smaller than key size to account for padding)
	RSAChunkSize = 256 // For 2048-bit keys
)

// SyncService handles vault synchronization
type SyncService struct {
	host         host.Host
	vaultMgr     *config.Manager
	privateKey   *rsa.PrivateKey
	publicKey    *rsa.PublicKey
	rsaConfig    *config.RSAConfig
	trustedPeers map[peer.ID]*PeerInfo
	vaultConfig  *config.VaultConfig
}

// PeerInfo contains information about a trusted peer
type PeerInfo struct {
	ID           peer.ID
	PublicKey    *rsa.PublicKey
	Fingerprint  string
	Name         string
	TrustedSince time.Time
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
	// Basic initialization without RSA security
	s := &SyncService{
		host:         h,
		vaultMgr:     vm,
		trustedPeers: make(map[peer.ID]*PeerInfo),
	}

	// Register basic protocol handlers
	h.SetStreamHandler(protocol.ID(ManifestProtocolID), s.handleManifestRequest)
	h.SetStreamHandler(protocol.ID(ChunkProtocolID), s.handleChunkRequest)

	return s, nil
}

// NewSecureSyncService creates a new secure sync service with RSA key support
func NewSecureSyncService(
	h host.Host,
	vm *config.Manager,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	rsaConfig *config.RSAConfig,
) (*SyncService, error) {
	// Load vault configuration
	vaultConfig, err := vm.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load vault configuration: %w", err)
	}

	s := &SyncService{
		host:         h,
		vaultMgr:     vm,
		privateKey:   privateKey,
		publicKey:    publicKey,
		rsaConfig:    rsaConfig,
		trustedPeers: make(map[peer.ID]*PeerInfo),
		vaultConfig:  vaultConfig,
	}

	// Load trusted peers from config
	if rsaConfig != nil && rsaConfig.TrustedPeers != nil {
		for _, trustedPeer := range rsaConfig.TrustedPeers {
			// Parse the peer ID
			peerID, err := peer.Decode(trustedPeer.ID)
			if err != nil {
				fmt.Printf("Warning: Failed to decode peer ID %s: %v\n", trustedPeer.ID, err)
				continue
			}

			// Parse the public key
			block, _ := pem.Decode([]byte(trustedPeer.PublicKey))
			if block == nil {
				fmt.Printf("Warning: Failed to decode public key for peer %s\n", trustedPeer.ID)
				continue
			}

			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				fmt.Printf("Warning: Failed to parse public key for peer %s: %v\n", trustedPeer.ID, err)
				continue
			}

			rsaPublicKey, ok := pub.(*rsa.PublicKey)
			if !ok {
				fmt.Printf("Warning: Public key for peer %s is not an RSA key\n", trustedPeer.ID)
				continue
			}

			// Add to trusted peers map
			s.trustedPeers[peerID] = &PeerInfo{
				ID:           peerID,
				PublicKey:    rsaPublicKey,
				Fingerprint:  trustedPeer.Fingerprint,
				Name:         trustedPeer.Name,
				TrustedSince: trustedPeer.TrustedSince,
			}
		}
	}

	// Register all protocol handlers including secure ones
	s.RegisterProtocols(context.Background())

	return s, nil
}

// RegisterProtocols sets up all protocol handlers
func (s *SyncService) RegisterProtocols(ctx context.Context) {
	// Register basic protocol handlers
	s.host.SetStreamHandler(protocol.ID(ManifestProtocolID), s.handleManifestRequest)
	s.host.SetStreamHandler(protocol.ID(ChunkProtocolID), s.handleChunkRequest)

	// Register secure protocol handlers
	if s.privateKey != nil {
		s.host.SetStreamHandler(protocol.ID(KeyExchangeProtocol), s.handleKeyExchange)
		s.host.SetStreamHandler(protocol.ID(AuthProtocol), s.handleAuthentication)
	}
}

// handleKeyExchange handles key exchange requests from peers
func (s *SyncService) handleKeyExchange(stream network.Stream) {
	defer stream.Close()

	if s.publicKey == nil {
		fmt.Println("Cannot perform key exchange: no public key available")
		return
	}

	// Read peer's public key
	pemData, err := io.ReadAll(stream)
	if err != nil {
		fmt.Printf("Error reading peer's public key: %v\n", err)
		return
	}

	// Parse peer's public key
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		fmt.Println("Failed to decode peer's public key")
		return
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fmt.Printf("Failed to parse peer's public key: %v\n", err)
		return
	}

	peerPubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		fmt.Println("Peer's key is not an RSA public key")
		return
	}

	// Calculate fingerprint
	publicKeyDER, err := x509.MarshalPKIXPublicKey(peerPubKey)
	if err != nil {
		fmt.Printf("Failed to marshal peer's public key: %v\n", err)
		return
	}

	hash := sha256.Sum256(publicKeyDER)
	fingerprint := base64.StdEncoding.EncodeToString(hash[:])

	// Send our public key in response
	ourPublicKeyDER, err := x509.MarshalPKIXPublicKey(s.publicKey)
	if err != nil {
		fmt.Printf("Failed to marshal our public key: %v\n", err)
		return
	}

	ourPublicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: ourPublicKeyDER,
	}

	ourPubKeyPEM := pem.EncodeToMemory(ourPublicKeyBlock)
	_, err = stream.Write(ourPubKeyPEM)
	if err != nil {
		fmt.Printf("Failed to send our public key: %v\n", err)
		return
	}

	// Store peer info temporarily
	peerID := stream.Conn().RemotePeer()
	s.trustedPeers[peerID] = &PeerInfo{
		ID:           peerID,
		PublicKey:    peerPubKey,
		Fingerprint:  fingerprint,
		TrustedSince: time.Now(),
	}

	fmt.Printf("Key exchange completed with peer %s (fingerprint: %s)\n", peerID.String(), fingerprint)
}

// handleAuthentication handles authentication requests from peers
func (s *SyncService) handleAuthentication(stream network.Stream) {
	defer stream.Close()

	// Read challenge
	var challenge struct {
		Challenge []byte `json:"challenge"`
		Sender    string `json:"sender"`
	}

	if err := json.NewDecoder(stream).Decode(&challenge); err != nil {
		fmt.Printf("Error reading authentication challenge: %v\n", err)
		return
	}

	// Sign the challenge with our private key
	challengeHash := sha256.Sum256(challenge.Challenge)
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, challengeHash[:])
	if err != nil {
		fmt.Printf("Error signing challenge: %v\n", err)
		return
	}

	// Send response
	response := struct {
		Signature []byte `json:"signature"`
		VaultID   string `json:"vault_id"`
		Name      string `json:"name"`
	}{
		Signature: signature,
		VaultID:   s.vaultConfig.VaultID,
		Name:      s.vaultConfig.Name,
	}

	if err := json.NewEncoder(stream).Encode(response); err != nil {
		fmt.Printf("Error sending authentication response: %v\n", err)
	}
}

// handleManifestRequest processes requests for vault manifests - now with authentication check
func (s *SyncService) handleManifestRequest(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()

	// If we have RSA keys, verify the peer is trusted
	if s.privateKey != nil {
		if _, ok := s.trustedPeers[peerID]; !ok {
			fmt.Printf("Rejecting manifest request from untrusted peer: %s\n", peerID.String())

			// Send error response
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Unauthorized: Peer not trusted",
			}
			json.NewEncoder(stream).Encode(errorResponse)
			return
		}
	}

	// Get our vault manifest
	manifest, err := s.vaultMgr.GetManifest()
	if err != nil {
		fmt.Printf("Error getting manifest: %v\n", err)

		// Send error response
		errorResponse := struct {
			Error string `json:"error"`
		}{
			Error: "Internal error getting manifest",
		}
		json.NewEncoder(stream).Encode(errorResponse)
		return
	}

	// Encode and send the manifest
	if err := json.NewEncoder(stream).Encode(manifest); err != nil {
		fmt.Printf("Error sending manifest: %v\n", err)
	}
}

// handleChunkRequest processes requests for chunks - now with authentication and encryption
func (s *SyncService) handleChunkRequest(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()

	// If we have RSA keys, verify the peer is trusted
	var peerInfo *PeerInfo
	if s.privateKey != nil {
		var ok bool
		peerInfo, ok = s.trustedPeers[peerID]
		if !ok {
			fmt.Printf("Rejecting chunk request from untrusted peer: %s\n", peerID.String())

			// Send error response
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Unauthorized: Peer not trusted",
			}
			json.NewEncoder(stream).Encode(errorResponse)
			return
		}
	}

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

	// If using RSA encryption, encrypt the chunk for the recipient
	var encryptedData []byte
	if s.privateKey != nil && peerInfo != nil && peerInfo.PublicKey != nil {
		encryptedData = s.encryptLargeData(chunkData, peerInfo.PublicKey)
	} else {
		encryptedData = chunkData
	}

	// Send the chunk data
	response := struct {
		Size      int    `json:"size"`
		Data      []byte `json:"data"`
		Encrypted bool   `json:"encrypted"`
	}{
		Size:      len(chunkData),
		Data:      encryptedData,
		Encrypted: (s.privateKey != nil && peerInfo != nil),
	}

	if err := json.NewEncoder(stream).Encode(response); err != nil {
		fmt.Printf("Error sending chunk: %v\n", err)
	}
}

// encryptLargeData encrypts data that may be larger than RSA can handle in one block
func (s *SyncService) encryptLargeData(data []byte, publicKey *rsa.PublicKey) []byte {
	result := []byte{}

	// Calculate max chunk size based on key size (with overhead for PKCS#1v15 padding)
	maxChunkSize := (publicKey.Size() - 11)

	// Process data in chunks
	for i := 0; i < len(data); i += maxChunkSize {
		end := i + maxChunkSize
		if end > len(data) {
			end = len(data)
		}

		chunk := data[i:end]
		encryptedChunk, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, chunk)
		if err != nil {
			fmt.Printf("Error encrypting chunk: %v\n", err)
			continue
		}

		// Add encrypted chunk to result
		result = append(result, encryptedChunk...)
	}

	return result
}

// decryptLargeData decrypts data that was encrypted in chunks
func (s *SyncService) decryptLargeData(data []byte) []byte {
	result := []byte{}

	// Process data in chunks based on key size
	chunkSize := s.privateKey.Size()

	for i := 0; i < len(data); i += chunkSize {
		end := min(i+chunkSize, len(data))

		chunk := data[i:end]
		if len(chunk) < chunkSize {
			fmt.Printf("Warning: Incomplete chunk size %d vs %d\n", len(chunk), chunkSize)
			continue
		}

		decryptedChunk, err := rsa.DecryptPKCS1v15(rand.Reader, s.privateKey, chunk)
		if err != nil {
			fmt.Printf("Error decrypting chunk: %v\n", err)
			continue
		}

		// Add decrypted chunk to result
		result = append(result, decryptedChunk...)
	}

	return result
}

// VerifyAndExchangeKeys performs key exchange with a peer
func (s *SyncService) VerifyAndExchangeKeys(ctx context.Context, peerID peer.ID) (bool, error) {
	// Check if already trusted
	if _, ok := s.trustedPeers[peerID]; ok {
		return true, nil
	}

	// If no RSA keys, return true (no verification needed)
	if s.privateKey == nil {
		return true, nil
	}

	// Open a stream for key exchange
	stream, err := s.host.NewStream(ctx, peerID, protocol.ID(KeyExchangeProtocol))
	if err != nil {
		return false, fmt.Errorf("failed to open key exchange stream: %w", err)
	}
	defer stream.Close()

	// Send our public key
	publicKeyDER, err := x509.MarshalPKIXPublicKey(s.publicKey)
	if err != nil {
		return false, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}

	publicKeyPEM := pem.EncodeToMemory(publicKeyBlock)
	_, err = stream.Write(publicKeyPEM)
	if err != nil {
		return false, fmt.Errorf("failed to send public key: %w", err)
	}

	// Receive peer's public key
	pemData, err := io.ReadAll(stream)
	if err != nil {
		return false, fmt.Errorf("failed to read peer's public key: %w", err)
	}

	// Parse peer's public key
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return false, fmt.Errorf("failed to decode peer's public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse peer's public key: %w", err)
	}

	peerPubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("peer's key is not an RSA public key")
	}

	// Calculate fingerprint
	peerKeyDER, err := x509.MarshalPKIXPublicKey(peerPubKey)
	if err != nil {
		return false, fmt.Errorf("failed to marshal peer's public key: %w", err)
	}

	hash := sha256.Sum256(peerKeyDER)
	fingerprint := base64.StdEncoding.EncodeToString(hash[:])

	// Store peer info temporarily
	s.trustedPeers[peerID] = &PeerInfo{
		ID:           peerID,
		PublicKey:    peerPubKey,
		Fingerprint:  fingerprint,
		TrustedSince: time.Now(),
	}

	// Perform authentication challenge
	if err := s.authenticatePeer(ctx, peerID); err != nil {
		delete(s.trustedPeers, peerID)
		return false, fmt.Errorf("authentication failed: %w", err)
	}

	return false, nil // Not fully trusted until user confirms
}

// authenticatePeer sends an authentication challenge to verify peer identity
func (s *SyncService) authenticatePeer(ctx context.Context, peerID peer.ID) error {
	stream, err := s.host.NewStream(ctx, peerID, protocol.ID(AuthProtocol))
	if err != nil {
		return fmt.Errorf("failed to open authentication stream: %w", err)
	}
	defer stream.Close()

	// Generate random challenge
	challenge := make([]byte, 32)
	_, err = rand.Read(challenge)
	if err != nil {
		return fmt.Errorf("failed to generate challenge: %w", err)
	}

	// Send challenge
	request := struct {
		Challenge []byte `json:"challenge"`
		Sender    string `json:"sender"`
	}{
		Challenge: challenge,
		Sender:    s.vaultConfig.VaultID,
	}

	if err := json.NewEncoder(stream).Encode(request); err != nil {
		return fmt.Errorf("failed to send challenge: %w", err)
	}

	// Read response
	var response struct {
		Signature []byte `json:"signature"`
		VaultID   string `json:"vault_id"`
		Name      string `json:"name"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return fmt.Errorf("failed to read auth response: %w", err)
	}

	// Get peer's public key
	peerInfo, ok := s.trustedPeers[peerID]
	if !ok {
		return fmt.Errorf("peer not found in trusted list")
	}

	// Verify signature
	challengeHash := sha256.Sum256(challenge)
	err = rsa.VerifyPKCS1v15(peerInfo.PublicKey, crypto.SHA256, challengeHash[:], response.Signature)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// Update peer info with vault details
	peerInfo.Name = response.Name

	return nil
}

// GetPeerFingerprint returns the fingerprint of a peer's public key
func (s *SyncService) GetPeerFingerprint(peerID peer.ID) (string, error) {
	peerInfo, ok := s.trustedPeers[peerID]
	if !ok {
		return "", fmt.Errorf("peer not found in trusted list")
	}

	return peerInfo.Fingerprint, nil
}

// AddTrustedPeer adds a peer to the trusted peers list and saves to config
func (s *SyncService) AddTrustedPeer(ctx context.Context, peerID peer.ID) error {
	peerInfo, ok := s.trustedPeers[peerID]
	if !ok {
		return fmt.Errorf("peer not found in temporary trusted list")
	}

	// Add to permanent trusted peers in config
	if s.rsaConfig != nil {
		// Convert public key to PEM
		publicKeyDER, err := x509.MarshalPKIXPublicKey(peerInfo.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to marshal public key: %w", err)
		}

		publicKeyBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyDER,
		}

		publicKeyPEM := string(pem.EncodeToMemory(publicKeyBlock))

		// Create trusted peer entry
		trustedPeer := config.TrustedPeer{
			ID:           peerID.String(),
			Name:         peerInfo.Name,
			PublicKey:    publicKeyPEM,
			Fingerprint:  peerInfo.Fingerprint,
			TrustedSince: time.Now(),
		}

		// Add to config
		s.rsaConfig.TrustedPeers = append(s.rsaConfig.TrustedPeers, trustedPeer)

		// Save updated config
		err = s.vaultMgr.SaveConfig(s.vaultConfig)
		if err != nil {
			return fmt.Errorf("failed to save updated config: %w", err)
		}
	}

	return nil
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
	var response struct {
		Error string                 `json:"error,omitempty"`
		Files []*config.FileManifest `json:"files,omitempty"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, err
	}

	if response.Error != "" {
		return nil, fmt.Errorf("remote error: %s", response.Error)
	}

	valueFiles := make([]config.FileManifest, len(response.Files))
	for i, filePtr := range response.Files {
		if filePtr != nil {
			valueFiles[i] = *filePtr
		}
	}
	manifest := &config.Manifest{
		Files: valueFiles,
	}

	return manifest, nil
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
		Error     string `json:"error,omitempty"`
		Size      int    `json:"size,omitempty"`
		Data      []byte `json:"data,omitempty"`
		Encrypted bool   `json:"encrypted"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, 0, err
	}

	if response.Error != "" {
		return nil, 0, fmt.Errorf("remote error: %s", response.Error)
	}

	// Decrypt data if necessary
	var chunkData []byte
	if response.Encrypted && s.privateKey != nil {
		chunkData = s.decryptLargeData(response.Data)
	} else {
		chunkData = response.Data
	}

	return chunkData, response.Size, nil
}
