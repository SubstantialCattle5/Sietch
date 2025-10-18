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
	"slices"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/manifest" //golangci-lint error
)

const (
	// Protocol IDs for different sync operations
	ManifestProtocolID   = "/sietch/manifest/1.0.0"
	ManifestProtocolIDv0 = "/sietch/manifest/0.9.0" // Fallback version
	ChunkProtocolID      = "/sietch/chunk/1.0.0"
	KeyExchangeProtocol  = "/sietch/key-exchange/1.0.0"
	AuthProtocol         = "/sietch/auth/1.0.0"

	// RSA encryption chunk size (must be smaller than key size to account for padding)
	RSAChunkSize = 256 // For 2048-bit keys
)

// SyncService handles vault synchronization
type SyncService struct {
	host                   host.Host
	vaultMgr               *config.Manager
	privateKey             *rsa.PrivateKey
	publicKey              *rsa.PublicKey
	rsaConfig              *config.RSAConfig
	trustedPeers           map[peer.ID]*PeerInfo
	vaultConfig            *config.VaultConfig
	trustAllPeers          bool                  // Legacy flag - use autoTrustAllPeers instead
	autoTrustAllPeers      bool                  // New flag to automatically trust all peers
	pendingOutgoingPeers   map[peer.ID]time.Time // Peers we want to pair with
	pendingIncomingAllowed map[peer.ID]time.Time // Peers allowed to pair with us
	pairingWindow          time.Duration         // TTL for pending pairs
	Verbose                bool                  // Enable verbose debug output
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
		host:          h,
		vaultMgr:      vm,
		trustedPeers:  make(map[peer.ID]*PeerInfo),
		trustAllPeers: true, // Trust all peers by default
	}

	// Register basic protocol handlers
	h.SetStreamHandler(protocol.ID(ManifestProtocolID), s.handleManifestRequest)
	h.SetStreamHandler(protocol.ID(ManifestProtocolIDv0), s.handleManifestRequest) // Support fallback version
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

	// Determine auto-trust behavior from config
	autoTrustAllPeers := true // Default to true for backward compatibility
	if rsaConfig != nil && rsaConfig.AutoTrustAllPeers != nil {
		autoTrustAllPeers = *rsaConfig.AutoTrustAllPeers
	}

	s := &SyncService{
		host:                   h,
		vaultMgr:               vm,
		privateKey:             privateKey,
		publicKey:              publicKey,
		rsaConfig:              rsaConfig,
		trustedPeers:           make(map[peer.ID]*PeerInfo),
		vaultConfig:            vaultConfig,
		trustAllPeers:          autoTrustAllPeers, // Legacy flag - kept for backward compatibility
		autoTrustAllPeers:      autoTrustAllPeers, // Use config value
		pendingOutgoingPeers:   make(map[peer.ID]time.Time),
		pendingIncomingAllowed: make(map[peer.ID]time.Time),
		pairingWindow:          5 * time.Minute, // Default 5 minute pairing window
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
	s.host.SetStreamHandler(protocol.ID(ManifestProtocolIDv0), s.handleManifestRequest) // Support fallback version
	s.host.SetStreamHandler(protocol.ID(ChunkProtocolID), s.handleChunkRequest)

	// Register secure protocol handlers
	if s.privateKey != nil {
		s.host.SetStreamHandler(protocol.ID(KeyExchangeProtocol), s.handleKeyExchange)
		s.host.SetStreamHandler(protocol.ID(AuthProtocol), s.handleAuthentication)
	}
}

// SetTrustAllPeers sets whether to automatically trust all peers
func (s *SyncService) SetTrustAllPeers(trustAll bool) {
	s.trustAllPeers = trustAll
	fmt.Printf("Trust all peers set to: %v\n", trustAll)
}

// handleKeyExchange handles key exchange requests from peers
// handleKeyExchange handles key exchange requests from peers
func (s *SyncService) handleKeyExchange(stream network.Stream) {
	defer stream.Close()

	if s.publicKey == nil {
		fmt.Println("Cannot perform key exchange: no public key available")
		return
	}

	// Check if incoming pairing is allowed for this peer
	peerID := stream.Conn().RemotePeer()
	if !s.IsPairingAllowed(peerID) {
		fmt.Printf("Rejecting key exchange from peer %s: pairing not allowed\n", peerID.String())
		// Send error response
		errorResponse := "Pairing not allowed. Peer must be explicitly allowed to pair."
		_, _ = stream.Write([]byte(errorResponse))
		return
	}

	// Use connection deadline instead of separate read/write deadlines
	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
	// Read peer's public key in chunks
	var pemData []byte
	buffer := make([]byte, 1024)
	for {
		n, err := stream.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading peer's public key: %v\n", err)
			return
		}
		pemData = append(pemData, buffer[:n]...)

		// Check if we have a complete PEM block
		if block, _ := pem.Decode(pemData); block != nil {
			// If we got a complete block, we can stop reading
			break
		}
	}

	// Parse peer's public key
	block, _ := pem.Decode(pemData)
	if block == nil {
		fmt.Println("Failed to decode peer's public key: empty block")
		return
	}

	// Support different key formats
	var peerPubKey *rsa.PublicKey
	var err error

	switch block.Type {
	case "RSA PUBLIC KEY":
		// Try PKCS1 format
		directKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			fmt.Printf("Failed to parse as PKCS1: %v\n", err)
		} else {
			peerPubKey = directKey
		}
	case "PUBLIC KEY":
		// Try PKIX format
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			fmt.Printf("Failed to parse peer's public key: %v\n", err)
			return
		}
		var ok bool
		peerPubKey, ok = pub.(*rsa.PublicKey)
		if !ok {
			fmt.Println("Peer's key is not an RSA public key")
			return
		}
	default:
		fmt.Printf("Unknown key format: %s\n", block.Type)
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

	// Store peer info automatically
	s.trustedPeers[peerID] = &PeerInfo{
		ID:           peerID,
		PublicKey:    peerPubKey,
		Fingerprint:  fingerprint,
		TrustedSince: time.Now(),
	}

	// Clear from pending maps since pairing is now complete
	delete(s.pendingIncomingAllowed, peerID)
	delete(s.pendingOutgoingPeers, peerID)

	fmt.Printf("Key exchange completed with peer %s (fingerprint: %s)\n", peerID.String(), fingerprint)
}

// handleAuthentication handles authentication requests from peers
func (s *SyncService) handleAuthentication(stream network.Stream) {
	defer stream.Close()

	// Read challenge with timeout
	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
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

	// Send response with timeout
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
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

// handleManifestRequest processes requests for vault manifests
func (s *SyncService) handleManifestRequest(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()

	// If we have RSA keys and not trusting all peers, verify the peer is trusted
	if s.privateKey != nil && !s.trustAllPeers {
		if _, ok := s.trustedPeers[peerID]; !ok {
			fmt.Printf("Rejecting manifest request from untrusted peer: %s\n", peerID.String())
			// Send error response
			errorResponse := struct {
				Error string `json:"error"`
			}{
				Error: "Unauthorized: Peer not trusted",
			}
			_ = json.NewEncoder(stream).Encode(errorResponse)
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
		_ = json.NewEncoder(stream).Encode(errorResponse)
		return
	}

	// Prepare response with correct structure
	response := struct {
		Files []*config.FileManifest `json:"files"`
		Error string                 `json:"error,omitempty"`
	}{
		Files: make([]*config.FileManifest, len(manifest.Files)),
	}

	// Convert from value to pointer slices
	for i := range manifest.Files {
		fileCopy := manifest.Files[i] // Create a copy to avoid aliasing issues
		response.Files[i] = &fileCopy
	}

	// Encode and send the manifest with timeout
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if err := json.NewEncoder(stream).Encode(response); err != nil {
		fmt.Printf("Error sending manifest: %v\n", err)
	}
}

// handleChunkRequest processes requests for chunks
func (s *SyncService) handleChunkRequest(stream network.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()

	// If we have RSA keys and not trusting all peers, verify the peer is trusted
	var peerInfo *PeerInfo
	if s.privateKey != nil && !s.trustAllPeers {
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
			_ = json.NewEncoder(stream).Encode(errorResponse)
			return
		}
	}

	// Read the chunk hash with timeout
	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
	var chunkRequest struct {
		Hash          string `json:"hash"`
		EncryptedHash string `json:"encrypted_hash,omitempty"`
		IsEncrypted   bool   `json:"is_encrypted"`
	}

	if err := json.NewDecoder(stream).Decode(&chunkRequest); err != nil {
		fmt.Printf("Error reading chunk request: %v\n", err)
		return
	}

	// First try using the primary hash
	chunkHash := chunkRequest.Hash
	if s.Verbose {
		fmt.Printf("Looking for chunk with hash: %s\n", chunkHash)
	}
	chunkData, err := s.vaultMgr.GetChunk(chunkHash)

	// If that fails and we have an encrypted hash, try that
	if err != nil && chunkRequest.EncryptedHash != "" {
		if s.Verbose {
			fmt.Printf("Chunk not found, trying encrypted hash: %s\n", chunkRequest.EncryptedHash)
		}
		chunkData, err = s.vaultMgr.GetChunk(chunkRequest.EncryptedHash)
		if err == nil {
			if s.Verbose {
				fmt.Printf("Found chunk using encrypted hash\n")
			}
		}
	}

	// If still not found, return error
	if err != nil {
		if s.Verbose {
			fmt.Printf("Chunk not found with either hash\n")
		}
		response := struct {
			Error string `json:"error"`
		}{
			Error: "Chunk not found",
		}
		_ = json.NewEncoder(stream).Encode(response)
		return
	}

	// If using RSA encryption, encrypt the chunk for the recipient
	var encryptedData []byte
	if s.privateKey != nil && peerInfo != nil && peerInfo.PublicKey != nil {
		encryptedData = s.encryptLargeData(chunkData, peerInfo.PublicKey)
	} else {
		encryptedData = chunkData
	}

	// Send the chunk data with timeout
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
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
	// Check if outgoing pairing is requested for this peer
	if !s.IsOutgoingPairRequested(peerID) {
		return false, fmt.Errorf("pairing not requested for peer %s. Use 'sietch pair' to establish trust", peerID.String())
	}

	// Don't return early with trustAllPeers, just mark for later
	needsKeyExchange := true
	autoTrust := s.autoTrustAllPeers

	// Check if already trusted
	if _, ok := s.trustedPeers[peerID]; ok {
		autoTrust = true
		// We might still need to exchange keys if fingerprint is missing
		if s.trustedPeers[peerID].Fingerprint != "" && s.trustedPeers[peerID].PublicKey != nil {
			needsKeyExchange = false
		}
	}

	// If no RSA keys, return true (no verification needed)
	if s.privateKey == nil {
		return true, nil
	}

	// Do key exchange if needed
	if needsKeyExchange {
		// Create stream and exchange keys as in original code
		timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		stream, err := s.host.NewStream(timeoutCtx, peerID, protocol.ID(KeyExchangeProtocol))
		if err != nil {
			// Check if we already have peer info from reverse connection
			if peerInfo, ok := s.trustedPeers[peerID]; ok && peerInfo.Fingerprint != "" {
				fmt.Printf("Failed to open stream, but have fingerprint from reverse connection: %s\n", peerInfo.Fingerprint)
				return true, nil
			}
			return false, fmt.Errorf("failed to open key exchange stream: %w", err)
		}
		defer stream.Close()

		// Use connection deadline instead of separate read/write deadlines
		_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
		_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
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

		// Read peer's public key in chunks
		var pemData []byte
		buffer := make([]byte, 1024)
		for {
			n, err := stream.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				// Check if we already have peer info from reverse connection
				if peerInfo, ok := s.trustedPeers[peerID]; ok && peerInfo.Fingerprint != "" {
					fmt.Printf("Read error, but have fingerprint from reverse connection: %s\n", peerInfo.Fingerprint)
					return true, nil
				}
				return false, fmt.Errorf("failed reading key data: %w", err)
			}
			pemData = append(pemData, buffer[:n]...)

			// Check if we have a complete PEM block
			if block, _ := pem.Decode(pemData); block != nil {
				// If we got a complete block, we can stop reading
				break
			}
		}

		// Parse peer's public key
		block, _ := pem.Decode(pemData)
		if block == nil {
			// Check if we already have peer info from reverse connection
			if peerInfo, ok := s.trustedPeers[peerID]; ok && peerInfo.Fingerprint != "" {
				fmt.Printf("Failed to decode PEM block, but have fingerprint from reverse connection: %s\n", peerInfo.Fingerprint)
				return true, nil
			}
			return false, fmt.Errorf("failed to decode peer's public key: empty block")
		}

		// Support different key formats
		var peerPubKey *rsa.PublicKey
		var pub interface{}

		switch block.Type {
		case "RSA PUBLIC KEY":
			// Try PKCS1 format
			directKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				fmt.Printf("Failed to parse as PKCS1: %v\n", err)
			} else {
				peerPubKey = directKey
			}
		case "PUBLIC KEY":
			// Try PKIX format
			pub, err = x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return false, fmt.Errorf("failed to parse PKIX public key: %w", err)
			}
			var ok bool
			peerPubKey, ok = pub.(*rsa.PublicKey)
			if !ok {
				return false, fmt.Errorf("peer's key is not an RSA public key")
			}
		default:
			return false, fmt.Errorf("unknown key format: %s", block.Type)
		}

		// Calculate fingerprint
		peerKeyDER, err := x509.MarshalPKIXPublicKey(peerPubKey)
		if err != nil {
			return false, fmt.Errorf("failed to marshal peer's public key: %w", err)
		}

		hash := sha256.Sum256(peerKeyDER)
		fingerprint := base64.StdEncoding.EncodeToString(hash[:])

		// Store peer info
		s.trustedPeers[peerID] = &PeerInfo{
			ID:           peerID,
			PublicKey:    peerPubKey,
			Fingerprint:  fingerprint,
			TrustedSince: time.Now(),
		}
	}

	// Auto-trust if configured to do so
	if autoTrust {
		return true, nil
	}

	// Perform authentication challenge
	if err := s.authenticatePeer(ctx, peerID); err != nil {
		delete(s.trustedPeers, peerID)
		return false, fmt.Errorf("authentication failed: %w", err)
	}

	return true, nil
}

// authenticatePeer sends an authentication challenge to verify peer identity
func (s *SyncService) authenticatePeer(ctx context.Context, peerID peer.ID) error {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	stream, err := s.host.NewStream(timeoutCtx, peerID, protocol.ID(AuthProtocol))
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

	// Send challenge with timeout
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
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

	// Read response with timeout
	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
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
	if s.rsaConfig != nil && peerInfo.PublicKey != nil {
		// First, check if this peer already exists in the trusted peers list
		if s.rsaConfig.TrustedPeers == nil {
			s.rsaConfig.TrustedPeers = []config.TrustedPeer{}
		}

		// Check for existing peer by ID or fingerprint
		existingPeer := false
		for _, peer := range s.rsaConfig.TrustedPeers {
			if peer.ID == peerID.String() || peer.Fingerprint == peerInfo.Fingerprint {
				existingPeer = true
				fmt.Printf("Peer already in trusted list (ID: %s, Fingerprint: %s)\n",
					peer.ID, peer.Fingerprint)
				break
			}
		}

		if existingPeer {
			// Peer already exists, no need to add again
			return nil
		}

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

		// Make sure vaultConfig is updated with the latest rsaConfig
		if s.vaultConfig != nil {
			s.vaultConfig.Sync.RSA = s.rsaConfig
		}

		// Save updated config
		if err := s.vaultMgr.SaveConfig(s.vaultConfig); err != nil {
			return fmt.Errorf("failed to save updated config: %w", err)
		}

		// // Pretty print newly trusted peer
		// data, err := yaml.Marshal(trustedPeer)
		// if err != nil {
		// 	fmt.Printf("ERROR: Failed to marshal trusted peer: %v\n", err)
		// } else {
		// 	fmt.Println("=========== NEW TRUSTED PEER ===========")
		// 	fmt.Println(string(data))
		// 	fmt.Println("=========== END TRUSTED PEER ===========")
		// }
	}

	return nil
}

// SyncWithPeer performs a sync operation with a specific peer
func (s *SyncService) SyncWithPeer(ctx context.Context, peerID peer.ID) (*SyncResult, error) {
	// Create a context with timeout for the entire operation
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	result := &SyncResult{}

	// First verify and exchange keys with peer (will auto-trust if trustAllPeers is true)
	if s.Verbose {
		fmt.Printf("Starting key verification with peer %s...\n", peerID.String())
	}
	trusted, err := s.VerifyAndExchangeKeys(timeoutCtx, peerID)
	if err != nil {
		return nil, fmt.Errorf("key exchange failed: %w", err)
	}

	if !trusted {
		return nil, fmt.Errorf("peer %s is not trusted", peerID.String())
	}
	if s.Verbose {
		fmt.Printf("Peer %s is trusted, proceeding with sync\n", peerID.String())
	}

	// Step 1: Get remote manifest
	if s.Verbose {
		fmt.Printf("Retrieving manifest from peer %s...\n", peerID.String())
	}
	remoteManifest, err := s.getRemoteManifest(timeoutCtx, peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote manifest: %v", err)
	}
	if s.Verbose {
		fmt.Printf("Retrieved manifest from peer with %d files\n", len(remoteManifest.Files))
	}

	// Step 2: Get local manifest
	localManifest, err := s.vaultMgr.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to get local manifest: %v", err)
	}

	// Step 3: Find missing chunks
	missingChunks := s.findMissingChunks(localManifest, remoteManifest)
	if s.Verbose {
		fmt.Printf("Found %d missing chunks to fetch\n", len(missingChunks))
	}

	// Step 4: Fetch missing chunks
	for i, chunkHash := range missingChunks {
		if s.Verbose && i%10 == 0 {
			fmt.Printf("Fetching chunk %d of %d...\n", i+1, len(missingChunks))
		}

		exists, _ := s.vaultMgr.ChunkExists(chunkHash)
		if exists {
			result.ChunksDeduplicated++
			continue
		}

		// Find associated encrypted hash if any
		var encryptedHash string
		for _, file := range remoteManifest.Files {
			for _, chunk := range file.Chunks {
				if chunk.Hash == chunkHash && chunk.EncryptedHash != "" {
					encryptedHash = chunk.EncryptedHash
					break
				}
			}
			if encryptedHash != "" {
				break
			}
		}

		// Pass the encrypted hash directly to fetchChunk
		chunkData, size, err := s.fetchChunk(timeoutCtx, peerID, chunkHash, encryptedHash)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch chunk %s: %v", chunkHash, err)
		}

		// Store the chunk with both hashes if needed
		if err := s.StoreChunk(chunkHash, chunkData, encryptedHash); err != nil {
			return nil, fmt.Errorf("failed to store chunk %s: %v", chunkHash, err)
		}

		result.ChunksTransferred++
		result.BytesTransferred += int64(size)
	}

	// Step 5: Save file manifests for synced files
	if s.Verbose {
		fmt.Println("Saving file manifests...")
	}
	savedCount := 0
	for _, remoteFile := range remoteManifest.Files {
		// Check if this file already exists locally
		exists := false
		for _, localFile := range localManifest.Files {
			if localFile.FilePath == remoteFile.FilePath {
				exists = true
				break
			}
		}

		if !exists {
			// Create a copy of the file manifest to avoid pointer issues
			fileManifest := remoteFile

			err := manifest.StoreFileManifest(
				s.vaultMgr.VaultRoot(),
				fileManifest.FilePath,
				&fileManifest,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to save manifest for %s: %v",
					fileManifest.FilePath, err)
			}
			if s.Verbose {
				fmt.Printf("Saved manifest for: %s\n", fileManifest.FilePath)
			}
			savedCount++
		}
	}
	if s.Verbose {
		fmt.Printf("Saved %d file manifests\n", savedCount)
	}
	result.FileCount = savedCount

	// Step 6: Rebuild references
	if err := s.vaultMgr.RebuildReferences(); err != nil {
		return nil, fmt.Errorf("failed to rebuild references: %v", err)
	}

	result.Duration = time.Since(startTime)
	if s.Verbose {
		fmt.Printf("Sync completed in %v: %d files, %d chunks transferred, %d chunks reused\n",
			result.Duration, result.FileCount, result.ChunksTransferred, result.ChunksDeduplicated)
	}

	return result, nil
}

// getRemoteManifest fetches the manifest from a remote peer
func (s *SyncService) getRemoteManifest(ctx context.Context, peerID peer.ID) (*config.Manifest, error) {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Try current protocol version first
	stream, err := s.host.NewStream(timeoutCtx, peerID, protocol.ID(ManifestProtocolID))
	// If current version fails, try fallback
	if err != nil {
		stream, err = s.host.NewStream(timeoutCtx, peerID, protocol.ID(ManifestProtocolIDv0))
		if err != nil {
			return nil, fmt.Errorf("failed to connect with any protocol version: %w", err)
		}
	}
	defer stream.Close()

	// Set read deadline
	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read the manifest
	var response struct {
		Error string                 `json:"error,omitempty"`
		Files []*config.FileManifest `json:"files,omitempty"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode manifest: %w", err)
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
	encryptedToRegularHash := make(map[string]string) // Track relationship between hashes

	// Log all chunk hashes for debugging
	if s.Verbose {
		fmt.Println("Local chunks:")
	}
	for _, file := range local.Files {
		for _, chunk := range file.Chunks {
			localChunks[chunk.Hash] = true
			if s.Verbose {
				fmt.Printf("  - Regular hash: %s\n", chunk.Hash)
			}

			// Also add encrypted hash if available
			if chunk.EncryptedHash != "" {
				localChunks[chunk.EncryptedHash] = true
				encryptedToRegularHash[chunk.EncryptedHash] = chunk.Hash
				if s.Verbose {
					fmt.Printf("    Encrypted hash: %s\n", chunk.EncryptedHash)
				}
			}
		}
	}

	// Find chunks in remote that don't exist locally
	if s.Verbose {
		fmt.Println("Remote chunks:")
	}
	for _, file := range remote.Files {
		for _, chunk := range file.Chunks {
			if s.Verbose {
				fmt.Printf("  - Checking remote chunk: %s\n", chunk.Hash)
				if chunk.EncryptedHash != "" {
					fmt.Printf("    With encrypted hash: %s\n", chunk.EncryptedHash)
				}
			}

			// Check if either regular or encrypted hash exists locally
			regularExists := localChunks[chunk.Hash]
			encryptedExists := chunk.EncryptedHash != "" && localChunks[chunk.EncryptedHash]

			if !regularExists && !encryptedExists {
				// This chunk is missing completely
				chunkToFetch := chunk.Hash
				alreadyAdded := slices.Contains(missingChunks, chunkToFetch)
				if !alreadyAdded {
					if s.Verbose {
						fmt.Printf("  - Adding missing chunk: %s\n", chunkToFetch)
					}
					missingChunks = append(missingChunks, chunkToFetch)
				}
			}
		}
	}

	return missingChunks
}

// fetchChunk downloads a chunk from a remote peer
func (s *SyncService) fetchChunk(ctx context.Context, peerID peer.ID, hash string, encryptedHash string) ([]byte, int, error) {
	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Use the provided encrypted hash instead of looking it up
	isEncrypted := encryptedHash != "" && s.privateKey != nil

	// Open a stream to the peer
	stream, err := s.host.NewStream(timeoutCtx, peerID, protocol.ID(ChunkProtocolID))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open chunk stream: %w", err)
	}
	defer stream.Close()

	// Set write deadline
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))

	// Send chunk request with both hash types
	request := struct {
		Hash          string `json:"hash"`
		EncryptedHash string `json:"encrypted_hash,omitempty"`
		IsEncrypted   bool   `json:"is_encrypted"`
	}{
		Hash:          hash,
		EncryptedHash: encryptedHash,
		IsEncrypted:   isEncrypted,
	}

	if s.Verbose {
		fmt.Printf("Requesting chunk with hash: %s, encrypted hash: %s\n", hash, encryptedHash)
	}
	if err := json.NewEncoder(stream).Encode(request); err != nil {
		return nil, 0, fmt.Errorf("failed to send chunk request: %w", err)
	}

	// Set read deadline
	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))

	// Read response
	var response struct {
		Error     string `json:"error,omitempty"`
		Size      int    `json:"size,omitempty"`
		Data      []byte `json:"data,omitempty"`
		Encrypted bool   `json:"encrypted"`
	}

	if err := json.NewDecoder(stream).Decode(&response); err != nil {
		return nil, 0, fmt.Errorf("failed to decode chunk response: %w", err)
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

// StoreChunk stores a chunk and handles the relationship between regular and encrypted hashes
func (s *SyncService) StoreChunk(hash string, data []byte, encryptedHash string) error {
	// Store the chunk with the primary hash
	if err := s.vaultMgr.StoreChunk(hash, data); err != nil {
		return fmt.Errorf("failed to store chunk with regular hash: %w", err)
	}

	// If we have an encrypted hash, store with that too
	if encryptedHash != "" {
		if err := s.vaultMgr.StoreChunk(encryptedHash, data); err != nil {
			fmt.Printf("Warning: Failed to store chunk with encrypted hash: %v\n", err)
			// Continue anyway since we stored it with the regular hash
		}
	}

	return nil
}

// HasPeer returns true if peer is already in trustedPeers map
func (s *SyncService) HasPeer(id peer.ID) bool {
	if s == nil {
		return false
	}
	_, ok := s.trustedPeers[id]
	return ok
}

// Pairing Management Methods

// AllowIncomingPair grants permission for a peer to pair with us
func (s *SyncService) AllowIncomingPair(peerID peer.ID, until time.Time) {
	if s == nil {
		return
	}
	s.pendingIncomingAllowed[peerID] = until
	if s.Verbose {
		fmt.Printf("Allowed incoming pairing from peer %s until %v\n", peerID.String(), until)
	}
}

// RequestPair requests to pair with a specific peer
func (s *SyncService) RequestPair(peerID peer.ID, until time.Time) {
	if s == nil {
		return
	}
	s.pendingOutgoingPeers[peerID] = until
	if s.Verbose {
		fmt.Printf("Requested pairing with peer %s until %v\n", peerID.String(), until)
	}
}

// ClearExpiredPairs removes expired entries from pending maps
func (s *SyncService) ClearExpiredPairs() {
	if s == nil {
		return
	}
	now := time.Now()

	// Clear expired outgoing pairs
	for peerID, until := range s.pendingOutgoingPeers {
		if now.After(until) {
			delete(s.pendingOutgoingPeers, peerID)
			if s.Verbose {
				fmt.Printf("Cleared expired outgoing pair request for peer %s\n", peerID.String())
			}
		}
	}

	// Clear expired incoming pairs
	for peerID, until := range s.pendingIncomingAllowed {
		if now.After(until) {
			delete(s.pendingIncomingAllowed, peerID)
			if s.Verbose {
				fmt.Printf("Cleared expired incoming pair permission for peer %s\n", peerID.String())
			}
		}
	}
}

// IsPairingAllowed checks if a peer is allowed to pair with us
func (s *SyncService) IsPairingAllowed(peerID peer.ID) bool {
	if s == nil {
		return false
	}

	// If auto-trust is enabled, allow all
	if s.autoTrustAllPeers {
		return true
	}

	// Check if peer is already trusted
	if _, ok := s.trustedPeers[peerID]; ok {
		return true
	}

	// Check if peer is in incoming allowed list and not expired
	if until, ok := s.pendingIncomingAllowed[peerID]; ok {
		if time.Now().Before(until) {
			return true
		}
		// Clean up expired entry
		delete(s.pendingIncomingAllowed, peerID)
	}

	return false
}

// IsOutgoingPairRequested checks if we have requested to pair with a peer
func (s *SyncService) IsOutgoingPairRequested(peerID peer.ID) bool {
	if s == nil {
		return false
	}

	// If auto-trust is enabled, allow all
	if s.autoTrustAllPeers {
		return true
	}

	// Check if peer is already trusted
	if _, ok := s.trustedPeers[peerID]; ok {
		return true
	}

	// Check if peer is in outgoing request list and not expired
	if until, ok := s.pendingOutgoingPeers[peerID]; ok {
		if time.Now().Before(until) {
			return true
		}
		// Clean up expired entry
		delete(s.pendingOutgoingPeers, peerID)
	}

	return false
}

// SetAutoTrustAllPeers sets the auto-trust behavior
func (s *SyncService) SetAutoTrustAllPeers(enabled bool) {
	if s == nil {
		return
	}
	s.autoTrustAllPeers = enabled
	s.trustAllPeers = enabled // Keep legacy flag in sync
	if s.Verbose {
		fmt.Printf("Auto-trust all peers: %v\n", enabled)
	}
}

// SetPairingWindow sets the pairing window duration
func (s *SyncService) SetPairingWindow(duration time.Duration) {
	if s == nil {
		return
	}
	s.pairingWindow = duration
	if s.Verbose {
		fmt.Printf("Pairing window set to: %v\n", duration)
	}
}

// TrustedPeers returns a copy of the trusted peers map
func (s *SyncService) TrustedPeers() map[peer.ID]*PeerInfo {
	if s == nil {
		return make(map[peer.ID]*PeerInfo)
	}

	// Return a copy to prevent external modification
	result := make(map[peer.ID]*PeerInfo)
	for k, v := range s.trustedPeers {
		result[k] = v
	}
	return result
}

// IsAutoTrustEnabled returns whether auto-trust is enabled
func (s *SyncService) IsAutoTrustEnabled() bool {
	if s == nil {
		return false
	}
	return s.autoTrustAllPeers
}
