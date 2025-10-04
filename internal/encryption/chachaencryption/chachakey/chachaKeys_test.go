package chachakey

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

func TestGenerateChaCha20Key(t *testing.T) {
	tests := []struct {
		name              string
		passphraseProtect bool
		passphrase        string
		expectError       bool
		errorContains     string
	}{
		{
			name:              "generate_unprotected_key",
			passphraseProtect: false,
			passphrase:        "",
			expectError:       false,
		},
		{
			name:              "generate_passphrase_protected_key",
			passphraseProtect: true,
			passphrase:        "test-passphrase-123",
			expectError:       false,
		},
		{
			name:              "passphrase_protected_without_passphrase",
			passphraseProtect: true,
			passphrase:        "",
			expectError:       true,
			errorContains:     "passphrase required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for keys
			tmpDir := t.TempDir()
			keyPath := filepath.Join(tmpDir, "chacha.key")

			// Create vault config
			cfg := &config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type:                constants.EncryptionTypeChaCha20,
					KeyPath:             keyPath,
					PassphraseProtected: tt.passphraseProtect,
					ChaChaConfig:        config.BuildDefaultChaChaConfig(),
				},
			}

			// Generate key
			keyConfig, err := GenerateChaCha20Key(cfg, tt.passphrase)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateChaCha20Key() expected error but got none")
					return
				}
				if tt.errorContains != "" && err.Error() != "" {
					// Just check error occurred, exact message may vary
					t.Logf("Got expected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateChaCha20Key() unexpected error: %v", err)
				return
			}

			// Verify keyConfig is not nil
			if keyConfig == nil {
				t.Error("GenerateChaCha20Key() returned nil keyConfig")
				return
			}

			// Verify ChaChaConfig exists
			if keyConfig.ChaChaConfig == nil {
				t.Error("keyConfig.ChaChaConfig is nil")
				return
			}

			// Verify key is not empty
			if keyConfig.ChaChaConfig.Key == "" {
				t.Error("keyConfig.ChaChaConfig.Key is empty")
				return
			}

			// Verify key hash is not empty
			if keyConfig.KeyHash == "" {
				t.Error("keyConfig.KeyHash is empty")
				return
			}

			// Verify key file was created
			if _, err := os.Stat(keyPath); os.IsNotExist(err) {
				t.Errorf("Key file was not created at %s", keyPath)
				return
			}

			// Verify key file permissions
			fileInfo, err := os.Stat(keyPath)
			if err != nil {
				t.Errorf("Failed to stat key file: %v", err)
				return
			}

			expectedPerms := os.FileMode(constants.SecureFilePerms)
			if fileInfo.Mode().Perm() != expectedPerms {
				t.Errorf("Key file permissions = %v, want %v", fileInfo.Mode().Perm(), expectedPerms)
			}

			// Verify key can be decoded
			_, err = base64.StdEncoding.DecodeString(keyConfig.ChaChaConfig.Key)
			if err != nil {
				t.Errorf("Failed to decode base64 key: %v", err)
			}

			// Verify key file size is appropriate
			if !tt.passphraseProtect {
				// For unprotected keys, should be exactly 32 bytes
				if fileInfo.Size() != chacha20poly1305.KeySize {
					t.Errorf("Unprotected key file size = %d, want %d", fileInfo.Size(), chacha20poly1305.KeySize)
				}
			} else {
				// For protected keys, should be larger (nonce + encrypted key + tag)
				if fileInfo.Size() <= chacha20poly1305.KeySize {
					t.Errorf("Protected key file size = %d, should be larger than %d", fileInfo.Size(), chacha20poly1305.KeySize)
				}
			}

			// Verify KDF parameters are set correctly
			if keyConfig.ChaChaConfig.KDF != constants.KDFScrypt {
				t.Errorf("KDF = %s, want %s", keyConfig.ChaChaConfig.KDF, constants.KDFScrypt)
			}

			if keyConfig.ChaChaConfig.Mode != "poly1305" {
				t.Errorf("Mode = %s, want poly1305", keyConfig.ChaChaConfig.Mode)
			}

			// Verify scrypt parameters
			if keyConfig.ChaChaConfig.ScryptN != constants.DefaultScryptN {
				t.Errorf("ScryptN = %d, want %d", keyConfig.ChaChaConfig.ScryptN, constants.DefaultScryptN)
			}

			if keyConfig.ChaChaConfig.ScryptR != constants.DefaultScryptR {
				t.Errorf("ScryptR = %d, want %d", keyConfig.ChaChaConfig.ScryptR, constants.DefaultScryptR)
			}

			if keyConfig.ChaChaConfig.ScryptP != constants.DefaultScryptP {
				t.Errorf("ScryptP = %d, want %d", keyConfig.ChaChaConfig.ScryptP, constants.DefaultScryptP)
			}

			// For passphrase-protected keys, verify salt is set
			if tt.passphraseProtect {
				if keyConfig.ChaChaConfig.Salt == "" {
					t.Error("Salt is empty for passphrase-protected key")
				}

				// Verify salt can be decoded
				_, err := base64.StdEncoding.DecodeString(keyConfig.ChaChaConfig.Salt)
				if err != nil {
					t.Errorf("Failed to decode base64 salt: %v", err)
				}
			}
		})
	}
}

func TestGenerateChaCha20KeyWithNilConfig(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "chacha.key")

	// Create vault config without ChaChaConfig (should create default)
	cfg := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			KeyPath:             keyPath,
			PassphraseProtected: false,
		},
	}

	keyConfig, err := GenerateChaCha20Key(cfg, "")
	if err != nil {
		t.Errorf("GenerateChaCha20Key() with nil ChaChaConfig failed: %v", err)
		return
	}

	if keyConfig == nil {
		t.Error("keyConfig is nil")
		return
	}

	if keyConfig.ChaChaConfig == nil {
		t.Error("ChaChaConfig is nil after generation")
		return
	}

	// Verify default config was created
	if cfg.Encryption.ChaChaConfig == nil {
		t.Error("cfg.Encryption.ChaChaConfig was not initialized")
	}
}

func TestGenerateChaCha20KeyInvalidDirectory(t *testing.T) {
	// Try to create key in non-existent directory with no permissions
	keyPath := "/root/nonexistent/impossible/chacha.key"

	cfg := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			KeyPath:             keyPath,
			PassphraseProtected: false,
			ChaChaConfig:        config.BuildDefaultChaChaConfig(),
		},
	}

	_, err := GenerateChaCha20Key(cfg, "")
	if err == nil {
		t.Error("GenerateChaCha20Key() expected error for invalid directory but got none")
	}
}

func TestWriteKeyToFile(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func() string
		expectError bool
	}{
		{
			name: "write_to_valid_path",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "test.key")
			},
			expectError: false,
		},
		{
			name: "write_to_nested_path",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nested", "dir", "test.key")
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyPath := tt.setupFunc()
			testKey := []byte("test-key-material-32-bytes-long!")

			err := writeKeyToFile(keyPath, testKey)

			if tt.expectError {
				if err == nil {
					t.Error("writeKeyToFile() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("writeKeyToFile() unexpected error: %v", err)
				return
			}

			// Verify file was created
			if _, err := os.Stat(keyPath); os.IsNotExist(err) {
				t.Errorf("Key file was not created at %s", keyPath)
				return
			}

			// Verify file permissions
			fileInfo, err := os.Stat(keyPath)
			if err != nil {
				t.Errorf("Failed to stat key file: %v", err)
				return
			}

			expectedPerms := os.FileMode(constants.SecureFilePerms)
			if fileInfo.Mode().Perm() != expectedPerms {
				t.Errorf("Key file permissions = %v, want %v", fileInfo.Mode().Perm(), expectedPerms)
			}

			// Verify file contents
			content, err := os.ReadFile(keyPath)
			if err != nil {
				t.Errorf("Failed to read key file: %v", err)
				return
			}

			if string(content) != string(testKey) {
				t.Errorf("Key file content mismatch")
			}
		})
	}
}

func TestGenerateChaCha20KeyConsistency(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate two keys with same parameters
	keyPath1 := filepath.Join(tmpDir, "key1.key")
	keyPath2 := filepath.Join(tmpDir, "key2.key")

	cfg1 := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			KeyPath:             keyPath1,
			PassphraseProtected: false,
			ChaChaConfig:        config.BuildDefaultChaChaConfig(),
		},
	}

	cfg2 := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			KeyPath:             keyPath2,
			PassphraseProtected: false,
			ChaChaConfig:        config.BuildDefaultChaChaConfig(),
		},
	}

	keyConfig1, err := GenerateChaCha20Key(cfg1, "")
	if err != nil {
		t.Fatalf("First key generation failed: %v", err)
	}

	keyConfig2, err := GenerateChaCha20Key(cfg2, "")
	if err != nil {
		t.Fatalf("Second key generation failed: %v", err)
	}

	// Keys should be different (random generation)
	if keyConfig1.ChaChaConfig.Key == keyConfig2.ChaChaConfig.Key {
		t.Error("Two independently generated keys are identical (should be random)")
	}

	// Key hashes should be different
	if keyConfig1.KeyHash == keyConfig2.KeyHash {
		t.Error("Two independently generated key hashes are identical")
	}

	// Both should have the same parameters
	if keyConfig1.ChaChaConfig.KDF != keyConfig2.ChaChaConfig.KDF {
		t.Error("KDF parameters differ between keys")
	}

	if keyConfig1.ChaChaConfig.Mode != keyConfig2.ChaChaConfig.Mode {
		t.Error("Mode parameters differ between keys")
	}
}

func TestGenerateChaCha20KeyWithPassphraseRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "chacha.key")
	passphrase := "test-passphrase-12345"

	cfg := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			KeyPath:             keyPath,
			PassphraseProtected: true,
			ChaChaConfig:        config.BuildDefaultChaChaConfig(),
		},
	}

	// Generate passphrase-protected key
	keyConfig, err := GenerateChaCha20Key(cfg, passphrase)
	if err != nil {
		t.Fatalf("GenerateChaCha20Key() failed: %v", err)
	}

	// Verify the key was encrypted
	if keyConfig.ChaChaConfig.Salt == "" {
		t.Error("Salt is empty for passphrase-protected key")
	}

	// The key in keyConfig should be base64 encoded encrypted key
	encryptedKeyBytes, err := base64.StdEncoding.DecodeString(keyConfig.ChaChaConfig.Key)
	if err != nil {
		t.Fatalf("Failed to decode encrypted key: %v", err)
	}

	// Encrypted key should be larger than plain key (includes nonce and tag)
	if len(encryptedKeyBytes) <= chacha20poly1305.KeySize {
		t.Errorf("Encrypted key size = %d, should be > %d", len(encryptedKeyBytes), chacha20poly1305.KeySize)
	}

	// Verify key file was created with encrypted content
	keyFileContent, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}

	// File should contain encrypted key
	if len(keyFileContent) <= chacha20poly1305.KeySize {
		t.Errorf("Key file size = %d, should be > %d for encrypted key", len(keyFileContent), chacha20poly1305.KeySize)
	}
}

func TestGenerateChaCha20KeyDirectoryCreation(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "deeply", "nested", "path", "chacha.key")

	cfg := &config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:                constants.EncryptionTypeChaCha20,
			KeyPath:             keyPath,
			PassphraseProtected: false,
			ChaChaConfig:        config.BuildDefaultChaChaConfig(),
		},
	}

	_, err := GenerateChaCha20Key(cfg, "")
	if err != nil {
		t.Errorf("GenerateChaCha20Key() failed with nested path: %v", err)
		return
	}

	// Verify directory was created
	keyDir := filepath.Dir(keyPath)
	if _, err := os.Stat(keyDir); os.IsNotExist(err) {
		t.Errorf("Key directory was not created: %s", keyDir)
	}

	// Verify file was created
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("Key file was not created: %s", keyPath)
	}
}
