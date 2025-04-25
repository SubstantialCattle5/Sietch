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
	if err = os.MkdirAll(syncDir, 0700); err != nil {
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

	privateKeyFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
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

	publicKeyFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err = pem.Encode(publicKeyFile, publicKeyBlock); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Calculate fingerprint for the public key
	hash := sha256.Sum256(publicKeyDER)
	fingerprint := base64.StdEncoding.EncodeToString(hash[:])

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
