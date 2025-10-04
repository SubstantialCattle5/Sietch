package encryption

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/scrypt"
	"gopkg.in/yaml.v2"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/testutil"
)

func TestChaCha20Encryption(t *testing.T) {
	tests := []struct {
		name        string
		plaintext   string
		expectError bool
	}{
		{
			name:        "encrypt empty string",
			plaintext:   "",
			expectError: false,
		},
		{
			name:        "encrypt simple text",
			plaintext:   "Hello, World!",
			expectError: false,
		},
		{
			name:        "encrypt longer text",
			plaintext:   "This is a longer piece of text to test ChaCha20 encryption with more data.",
			expectError: false,
		},
		{
			name:        "encrypt binary data",
			plaintext:   string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary vault directory
			vaultRoot := testutil.TempDir(t, "test-vault-chacha")

			// Create a valid 32-byte ChaCha20 key
			key := make([]byte, chacha20poly1305.KeySize)
			if _, err := rand.Read(key); err != nil {
				t.Fatalf("Failed to generate test key: %v", err)
			}

			// Write key to file
			keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
			if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
				t.Fatalf("Failed to create key directory: %v", err)
			}
			if err := os.WriteFile(keyPath, key, 0600); err != nil {
				t.Fatalf("Failed to write key file: %v", err)
			}

			// Create vault config
			vaultConfig := config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type:    constants.EncryptionTypeChaCha20,
					KeyPath: keyPath,
				},
			}

			// Write vault config to file
			configData, err := yaml.Marshal(vaultConfig)
			if err != nil {
				t.Fatalf("Failed to marshal vault config: %v", err)
			}
			configPath := filepath.Join(vaultRoot, "vault.yaml")
			if err := os.WriteFile(configPath, configData, 0644); err != nil {
				t.Fatalf("Failed to write vault config: %v", err)
			}

			// Test encryption
			ciphertext, err := ChaCha20Encryption(tt.plaintext, vaultConfig)
			if tt.expectError {
				if err == nil {
					t.Errorf("ChaCha20Encryption() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ChaCha20Encryption() unexpected error: %v", err)
				return
			}

			if ciphertext == "" {
				t.Error("ChaCha20Encryption() returned empty ciphertext")
				return
			}

			// Verify ciphertext is different from plaintext
			if ciphertext == tt.plaintext {
				t.Error("ChaCha20Encryption() returned plaintext as ciphertext")
			}

			// Test decryption
			decrypted, err := ChaCha20Decryption(ciphertext, vaultRoot)
			if err != nil {
				t.Errorf("ChaCha20Decryption() error: %v", err)
				return
			}

			if decrypted != tt.plaintext {
				t.Errorf("ChaCha20Decryption() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestChaCha20EncryptionWithPassphrase(t *testing.T) {
	tests := []struct {
		name        string
		plaintext   string
		passphrase  string
		expectError bool
	}{
		{
			name:        "encrypt with passphrase",
			plaintext:   "Secret message",
			passphrase:  "test-passphrase",
			expectError: false,
		},
		{
			name:        "encrypt empty passphrase",
			plaintext:   "Secret message",
			passphrase:  "",
			expectError: true, // Should fail because passphrase is required for protected keys
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary vault directory
			vaultRoot := testutil.TempDir(t, "test-vault-chacha-pass")

			// Create vault config with passphrase protection
			vaultConfig := config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type:                constants.EncryptionTypeChaCha20,
					KeyPath:             filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key"),
					PassphraseProtected: true,
					ChaChaConfig:        config.BuildDefaultChaChaConfig(),
				},
			}

			// Create key file
			keyPath := vaultConfig.Encryption.KeyPath
			if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
				t.Fatalf("Failed to create key directory: %v", err)
			}
			if tt.passphrase != "" {
				// Create encrypted key for passphrase case
				key := make([]byte, chacha20poly1305.KeySize)
				if _, err := rand.Read(key); err != nil {
					t.Fatalf("Failed to generate test key: %v", err)
				}
				salt := make([]byte, 32)
				if _, err := rand.Read(salt); err != nil {
					t.Fatalf("Failed to generate salt: %v", err)
				}
				derivedKey, err := scrypt.Key([]byte(tt.passphrase), salt, vaultConfig.Encryption.ChaChaConfig.ScryptN, vaultConfig.Encryption.ChaChaConfig.ScryptR, vaultConfig.Encryption.ChaChaConfig.ScryptP, chacha20poly1305.KeySize)
				if err != nil {
					t.Fatalf("Failed to derive key: %v", err)
				}
				aead, err := chacha20poly1305.New(derivedKey)
				if err != nil {
					t.Fatalf("Failed to create AEAD: %v", err)
				}
				nonce := make([]byte, aead.NonceSize())
				if _, err := rand.Read(nonce); err != nil {
					t.Fatalf("Failed to generate nonce: %v", err)
				}
				encryptedKey := aead.Seal(nonce, nonce, key, nil)
				if err := os.WriteFile(keyPath, encryptedKey, 0600); err != nil {
					t.Fatalf("Failed to write encrypted key file: %v", err)
				}
				vaultConfig.Encryption.ChaChaConfig.Salt = base64.StdEncoding.EncodeToString(salt)
			} else {
				// Create plain key for empty passphrase case
				key := make([]byte, chacha20poly1305.KeySize)
				if _, err := rand.Read(key); err != nil {
					t.Fatalf("Failed to generate test key: %v", err)
				}
				if err := os.WriteFile(keyPath, key, 0600); err != nil {
					t.Fatalf("Failed to write key file: %v", err)
				}
			}

			// Write vault config to file
			configData, err := yaml.Marshal(vaultConfig)
			if err != nil {
				t.Fatalf("Failed to marshal vault config: %v", err)
			}
			configPath := filepath.Join(vaultRoot, "vault.yaml")
			if err := os.WriteFile(configPath, configData, 0644); err != nil {
				t.Fatalf("Failed to write vault config: %v", err)
			}

			// Test encryption with passphrase
			ciphertext, err := ChaCha20EncryptWithPassphrase(tt.plaintext, vaultConfig, tt.passphrase)
			if tt.expectError {
				if err == nil {
					t.Errorf("ChaCha20EncryptWithPassphrase() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ChaCha20EncryptWithPassphrase() unexpected error: %v", err)
				return
			}

			if ciphertext == "" {
				t.Error("ChaCha20EncryptWithPassphrase() returned empty ciphertext")
				return
			}

			// Test decryption with passphrase
			decrypted, err := ChaCha20DecryptionWithPassphrase(ciphertext, vaultRoot, tt.passphrase)
			if err != nil {
				t.Errorf("ChaCha20DecryptionWithPassphrase() error: %v", err)
				return
			}

			if decrypted != tt.plaintext {
				t.Errorf("ChaCha20DecryptionWithPassphrase() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestChaCha20InvalidKey(t *testing.T) {
	tests := []struct {
		name        string
		keySize     int
		expectError bool
	}{
		{
			name:        "invalid key too short",
			keySize:     16,
			expectError: true,
		},
		{
			name:        "invalid key too long",
			keySize:     64,
			expectError: true,
		},
		{
			name:        "valid key size",
			keySize:     chacha20poly1305.KeySize,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary vault directory
			vaultRoot := testutil.TempDir(t, "test-vault-chacha-invalid")

			// Create key of specified size
			key := make([]byte, tt.keySize)
			if _, err := rand.Read(key); err != nil {
				t.Fatalf("Failed to generate test key: %v", err)
			}

			// Write key to file
			keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
			if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
				t.Fatalf("Failed to create key directory: %v", err)
			}
			if err := os.WriteFile(keyPath, key, 0600); err != nil {
				t.Fatalf("Failed to write key file: %v", err)
			}

			// Create vault config
			vaultConfig := config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type:    constants.EncryptionTypeChaCha20,
					KeyPath: keyPath,
				},
			}

			// Test encryption
			_, err := ChaCha20Encryption("test data", vaultConfig)
			if tt.expectError {
				if err == nil {
					t.Errorf("ChaCha20Encryption() expected error for key size %d but got none", tt.keySize)
				}
			} else {
				if err != nil {
					t.Errorf("ChaCha20Encryption() unexpected error for valid key size: %v", err)
				}
			}
		})
	}
}

func TestChaCha20WrongConfig(t *testing.T) {
	tests := []struct {
		name        string
		configType  string
		expectError bool
	}{
		{
			name:        "wrong encryption type AES",
			configType:  constants.EncryptionTypeAES,
			expectError: true,
		},
		{
			name:        "wrong encryption type GPG",
			configType:  constants.EncryptionTypeGPG,
			expectError: true,
		},
		{
			name:        "correct encryption type",
			configType:  constants.EncryptionTypeChaCha20,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary vault directory
			vaultRoot := testutil.TempDir(t, "test-vault-chacha-config")

			// Create a valid 32-byte ChaCha20 key
			key := make([]byte, chacha20poly1305.KeySize)
			if _, err := rand.Read(key); err != nil {
				t.Fatalf("Failed to generate test key: %v", err)
			}

			// Write key to file
			keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
			if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
				t.Fatalf("Failed to create key directory: %v", err)
			}
			if err := os.WriteFile(keyPath, key, 0600); err != nil {
				t.Fatalf("Failed to write key file: %v", err)
			}

			// Create vault config with wrong type
			vaultConfig := config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type:    tt.configType,
					KeyPath: keyPath,
				},
			}

			// Test encryption
			_, err := ChaCha20Encryption("test data", vaultConfig)
			if tt.expectError {
				if err == nil {
					t.Errorf("ChaCha20Encryption() expected error for config type %s but got none", tt.configType)
				}
			} else {
				if err != nil {
					t.Errorf("ChaCha20Encryption() unexpected error for correct config type: %v", err)
				}
			}
		})
	}
}

func TestChaCha20RoundTrip(t *testing.T) {
	// Create temporary vault directory
	vaultRoot := testutil.TempDir(t, "test-vault-chacha-roundtrip")

	// Create a valid 32-byte ChaCha20 key
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Write key to file
	keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		t.Fatalf("Failed to create key directory: %v", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Create vault config
	vaultConfig := config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:    constants.EncryptionTypeChaCha20,
			KeyPath: keyPath,
		},
	}

	// Write vault config to file
	configData, err := yaml.Marshal(vaultConfig)
	if err != nil {
		t.Fatalf("Failed to marshal vault config: %v", err)
	}
	configPath := filepath.Join(vaultRoot, "vault.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write vault config: %v", err)
	}

	testData := []string{
		"",
		"a",
		"Hello, World!",
		"This is a test message with special characters: !@#$%^&*()",
		string(make([]byte, 1000)), // 1KB of data
	}

	for i, data := range testData {
		t.Run(fmt.Sprintf("roundtrip-%d", i), func(t *testing.T) {
			// Encrypt
			ciphertext, err := ChaCha20Encryption(data, vaultConfig)
			if err != nil {
				t.Errorf("ChaCha20Encryption() error: %v", err)
				return
			}

			// Decrypt
			decrypted, err := ChaCha20Decryption(ciphertext, vaultRoot)
			if err != nil {
				t.Errorf("ChaCha20Decryption() error: %v", err)
				return
			}

			// Verify
			if decrypted != data {
				t.Errorf("Round-trip failed: got %q, want %q", decrypted, data)
			}
		})
	}
}

func TestChaCha20TamperedCiphertext(t *testing.T) {
	// Create temporary vault directory
	vaultRoot := testutil.TempDir(t, "test-vault-chacha-tamper")

	// Create a valid 32-byte ChaCha20 key
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	// Write key to file
	keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		t.Fatalf("Failed to create key directory: %v", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}

	// Create vault config
	vaultConfig := config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:    constants.EncryptionTypeChaCha20,
			KeyPath: keyPath,
		},
	}

	// Write vault config to file
	configData, err := yaml.Marshal(vaultConfig)
	if err != nil {
		t.Fatalf("Failed to marshal vault config: %v", err)
	}
	configPath := filepath.Join(vaultRoot, "vault.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write vault config: %v", err)
	}

	// Encrypt valid data
	ciphertext, err := ChaCha20Encryption("test data", vaultConfig)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	// Tamper with ciphertext
	tamperedCiphertext := ciphertext[:len(ciphertext)-1] + "x"

	// Try to decrypt tampered data - should fail
	_, err = ChaCha20Decryption(tamperedCiphertext, vaultRoot)
	if err == nil {
		t.Error("ChaCha20Decryption() expected error for tampered ciphertext but got none")
	}
}

func TestChaCha20WrongKey(t *testing.T) {
	// Create temporary vault directory
	vaultRoot := testutil.TempDir(t, "test-vault-chacha-wrongkey")

	// Create first key
	key1 := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key1); err != nil {
		t.Fatalf("Failed to generate first test key: %v", err)
	}

	// Create second key
	key2 := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key2); err != nil {
		t.Fatalf("Failed to generate second test key: %v", err)
	}

	// Write first key to file
	keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		t.Fatalf("Failed to create key directory: %v", err)
	}
	if err := os.WriteFile(keyPath, key1, 0600); err != nil {
		t.Fatalf("Failed to write first key file: %v", err)
	}

	// Create vault config
	vaultConfig := config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:    constants.EncryptionTypeChaCha20,
			KeyPath: keyPath,
		},
	}

	// Write vault config to file
	configData, err := yaml.Marshal(vaultConfig)
	if err != nil {
		t.Fatalf("Failed to marshal vault config: %v", err)
	}
	configPath := filepath.Join(vaultRoot, "vault.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed to write vault config: %v", err)
	}

	// Encrypt with first key
	ciphertext, err := ChaCha20Encryption("test data", vaultConfig)
	if err != nil {
		t.Fatalf("Failed to encrypt with first key: %v", err)
	}

	// Replace with second key
	if err := os.WriteFile(keyPath, key2, 0600); err != nil {
		t.Fatalf("Failed to write second key file: %v", err)
	}

	// Try to decrypt with wrong key - should fail
	_, err = ChaCha20Decryption(ciphertext, vaultRoot)
	if err == nil {
		t.Error("ChaCha20Decryption() expected error for wrong key but got none")
	}
}

// Benchmark tests
func BenchmarkChaCha20Encryption(b *testing.B) {
	// Create temporary vault directory
	vaultRoot := testutil.TempDir(b, "bench-vault-chacha")

	// Create a valid 32-byte ChaCha20 key
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		b.Fatalf("Failed to generate test key: %v", err)
	}

	// Write key to file
	keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		b.Fatalf("Failed to create key directory: %v", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		b.Fatalf("Failed to write key file: %v", err)
	}

	// Create vault config
	vaultConfig := config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:    constants.EncryptionTypeChaCha20,
			KeyPath: keyPath,
		},
	}

	testData := "This is benchmark test data for ChaCha20 encryption performance measurement."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ChaCha20Encryption(testData, vaultConfig)
		if err != nil {
			b.Fatalf("ChaCha20Encryption failed: %v", err)
		}
	}
}

func BenchmarkChaCha20Decryption(b *testing.B) {
	// Create temporary vault directory
	vaultRoot := testutil.TempDir(b, "bench-vault-chacha-decrypt")

	// Create a valid 32-byte ChaCha20 key
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := rand.Read(key); err != nil {
		b.Fatalf("Failed to generate test key: %v", err)
	}

	// Write key to file
	keyPath := filepath.Join(vaultRoot, ".sietch", "keys", "chacha.key")
	if err := os.MkdirAll(filepath.Dir(keyPath), 0755); err != nil {
		b.Fatalf("Failed to create key directory: %v", err)
	}
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		b.Fatalf("Failed to write key file: %v", err)
	}

	// Create vault config
	vaultConfig := config.VaultConfig{
		Encryption: config.EncryptionConfig{
			Type:    constants.EncryptionTypeChaCha20,
			KeyPath: keyPath,
		},
	}

	// Write vault config to file
	configData, err := yaml.Marshal(vaultConfig)
	if err != nil {
		b.Fatalf("Failed to marshal vault config: %v", err)
	}
	configPath := filepath.Join(vaultRoot, "vault.yaml")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		b.Fatalf("Failed to write vault config: %v", err)
	}

	testData := "This is benchmark test data for ChaCha20 decryption performance measurement."

	// Pre-encrypt the data
	ciphertext, err := ChaCha20Encryption(testData, vaultConfig)
	if err != nil {
		b.Fatalf("Failed to pre-encrypt test data: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ChaCha20Decryption(ciphertext, vaultRoot)
		if err != nil {
			b.Fatalf("ChaCha20Decryption failed: %v", err)
		}
	}
}
