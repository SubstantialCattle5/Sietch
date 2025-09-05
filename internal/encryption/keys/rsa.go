package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"
)

// GenerateRSAKeyPair generates an RSA key pair and saves it to the specified directory
func GenerateRSAKeyPair(vaultRoot string, config *config.VaultConfig) error {
	bits := config.Sync.RSA.KeySize
	if bits < 2048 {
		return fmt.Errorf("RSA key size too small, minimum recommended is 2048 bits")
	}

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	// Validate private key
	if err = privateKey.Validate(); err != nil {
		return fmt.Errorf("invalid RSA private key: %w", err)
	}

	// Create sync key directory
	syncDir := filepath.Join(vaultRoot, ".sietch", "sync")
	if err = os.MkdirAll(syncDir, 0o700); err != nil {
		return fmt.Errorf("failed to create sync key directory: %w", err)
	}

	// Define relative paths for keys
	relPrivateKeyPath := filepath.Join(".sietch", "sync", "sync_private.pem")
	relPublicKeyPath := filepath.Join(".sietch", "sync", "sync_public.pem")

	// Define absolute paths for file operations
	privateKeyPath := filepath.Join(vaultRoot, relPrivateKeyPath)
	publicKeyPath := filepath.Join(vaultRoot, relPublicKeyPath)

	// Save private key (PKCS#1 format)
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	}

	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err = pem.Encode(privateKeyFile, privateKeyBlock); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Save public key (PKIX format)
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}

	publicKeyFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err = pem.Encode(publicKeyFile, publicKeyBlock); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Calculate fingerprint for the public key
	fingerprint, err := GetRSAPublicKeyFingerprint(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to calculate key fingerprint: %w", err)
	}

	// Update config with key information
	config.Sync.RSA.PublicKeyPath = relPublicKeyPath
	config.Sync.RSA.PrivateKeyPath = relPrivateKeyPath
	config.Sync.RSA.Fingerprint = fingerprint

	fmt.Printf("RSA key pair generated for sync operations:\n")
	fmt.Printf("  - Private key: %s\n", privateKeyPath)
	fmt.Printf("  - Public key: %s\n", publicKeyPath)
	fmt.Printf("  - Fingerprint: %s\n", fingerprint)

	return nil
}

// LoadRSAKeys loads RSA keys from the specified paths
func LoadRSAKeys(vaultPath string, rsaConfig *config.RSAConfig) (*rsa.PrivateKey, *rsa.PublicKey, *config.RSAConfig, error) {
	// Load private key
	privateKeyPath := filepath.Join(vaultPath, rsaConfig.PrivateKeyPath)
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	privateKey, err := ParseRSAPrivateKeyFromPEM(privateKeyData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Load public key
	publicKeyPath := filepath.Join(vaultPath, rsaConfig.PublicKeyPath)
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse public key
	publicKey, err := ParseRSAPublicKeyFromPEM(publicKeyData)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Create RSA config
	newRsaConfig := &config.RSAConfig{
		KeySize:        rsaConfig.KeySize,
		TrustedPeers:   rsaConfig.TrustedPeers,
		PublicKeyPath:  rsaConfig.PublicKeyPath,
		PrivateKeyPath: rsaConfig.PrivateKeyPath,
		Fingerprint:    rsaConfig.Fingerprint,
	}

	return privateKey, publicKey, newRsaConfig, nil
}

// ParseRSAPrivateKeyFromPEM parses an RSA private key from PEM format
func ParseRSAPrivateKeyFromPEM(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return privateKey, nil
}

// ParseRSAPublicKeyFromPEM parses an RSA public key from PEM format
func ParseRSAPublicKeyFromPEM(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return publicKey, nil
}

// EncodeRSAPrivateKeyToPEM encodes an RSA private key to PEM format
func EncodeRSAPrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	}
	return pem.EncodeToMemory(privateKeyBlock)
}

// EncodeRSAPublicKeyToPEM encodes an RSA public key to PEM format
func EncodeRSAPublicKeyToPEM(publicKey *rsa.PublicKey) ([]byte, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}
	return pem.EncodeToMemory(publicKeyBlock), nil
}

// GetRSAPublicKeyFingerprint calculates the fingerprint for an RSA public key
func GetRSAPublicKeyFingerprint(publicKey *rsa.PublicKey) (string, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	hash := sha256.Sum256(publicKeyDER)
	fingerprint := base64.StdEncoding.EncodeToString(hash[:])
	return fingerprint, nil
}

// ValidateRSAKeyPair validates that the private and public keys form a valid pair
func ValidateRSAKeyPair(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) error {
	// Check if private key is valid
	if err := privateKey.Validate(); err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	// Check if public key matches private key
	privateN := privateKey.N.String()
	publicN := publicKey.N.String()
	privateE := privateKey.E
	publicE := publicKey.E

	if privateN != publicN || privateE != publicE {
		return fmt.Errorf("private and public keys do not form a valid pair")
	}

	return nil
}

// GenerateTestRSAKeyPair generates a key pair for testing purposes
func GenerateTestRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if bits < 2048 {
		bits = 2048 // Enforce minimum key size
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate test RSA key: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

// Get fingerprint for an RSA public key
func GetPublicKeyFingerprint(publicKey *rsa.PublicKey) (string, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %v", err)
	}

	// Calculate SHA-256 hash of the DER encoded public key
	hash := sha256.Sum256(publicKeyDER)
	fingerprint := base64.StdEncoding.EncodeToString(hash[:])

	return fingerprint, nil
}

// Export RSA public key to PEM format
func ExportRSAPublicKeyToPEM(publicKey *rsa.PublicKey) ([]byte, error) {
	publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %v", err)
	}

	publicKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDER,
	}

	return pem.EncodeToMemory(publicKeyBlock), nil
}
