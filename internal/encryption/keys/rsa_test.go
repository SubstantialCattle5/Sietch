package keys

import (
	"crypto/rsa"
	"path/filepath"
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/testutil"
)

func TestGenerateRSAKeyPair(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
		wantErr bool
	}{
		{
			name:    "valid 2048 bit key",
			keySize: 2048,
			wantErr: false,
		},
		{
			name:    "valid 4096 bit key",
			keySize: 4096,
			wantErr: false,
		},
		{
			name:    "invalid small key",
			keySize: 1024,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary vault directory
			vaultRoot := testutil.TempDir(t, "test-vault")

			// Create test config
			testConfig := &config.VaultConfig{
				Sync: config.SyncConfig{
					RSA: &config.RSAConfig{
						KeySize:      tt.keySize,
						TrustedPeers: []config.TrustedPeer{},
					},
				},
			}

			err := GenerateRSAKeyPair(vaultRoot, testConfig)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateRSAKeyPair() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateRSAKeyPair() unexpected error: %v", err)
				return
			}

			// Verify key files were created
			privateKeyPath := filepath.Join(vaultRoot, ".sietch", "sync", "sync_private.pem")
			publicKeyPath := filepath.Join(vaultRoot, ".sietch", "sync", "sync_public.pem")

			testutil.AssertFileExists(t, privateKeyPath)
			testutil.AssertFileExists(t, publicKeyPath)

			// Verify config was updated
			if testConfig.Sync.RSA.PrivateKeyPath == "" {
				t.Error("PrivateKeyPath was not set in config")
			}
			if testConfig.Sync.RSA.PublicKeyPath == "" {
				t.Error("PublicKeyPath was not set in config")
			}
			if testConfig.Sync.RSA.Fingerprint == "" {
				t.Error("Fingerprint was not set in config")
			}

			// Verify we can load the generated keys
			privateKey, publicKey, loadedConfig, err := LoadRSAKeys(vaultRoot, testConfig.Sync.RSA)
			if err != nil {
				t.Errorf("Failed to load generated keys: %v", err)
				return
			}

			// Verify key pair is valid
			if err := ValidateRSAKeyPair(privateKey, publicKey); err != nil {
				t.Errorf("Generated key pair is invalid: %v", err)
			}

			// Verify loaded config matches
			if loadedConfig.KeySize != tt.keySize {
				t.Errorf("Loaded config key size = %d, want %d", loadedConfig.KeySize, tt.keySize)
			}
		})
	}
}

func TestLoadRSAKeys(t *testing.T) {
	// Create temporary vault directory
	vaultRoot := testutil.TempDir(t, "test-vault")

	// Generate a key pair first
	testConfig := &config.VaultConfig{
		Sync: config.SyncConfig{
			RSA: &config.RSAConfig{
				KeySize:      2048,
				TrustedPeers: []config.TrustedPeer{},
			},
		},
	}

	err := GenerateRSAKeyPair(vaultRoot, testConfig)
	if err != nil {
		t.Fatalf("Failed to generate test key pair: %v", err)
	}

	t.Run("load valid keys", func(t *testing.T) {
		privateKey, publicKey, loadedConfig, err := LoadRSAKeys(vaultRoot, testConfig.Sync.RSA)
		if err != nil {
			t.Errorf("LoadRSAKeys() error = %v", err)
			return
		}

		if privateKey == nil {
			t.Error("LoadRSAKeys() returned nil private key")
		}
		if publicKey == nil {
			t.Error("LoadRSAKeys() returned nil public key")
		}
		if loadedConfig == nil {
			t.Error("LoadRSAKeys() returned nil config")
		}

		// Verify key pair is valid
		if err := ValidateRSAKeyPair(privateKey, publicKey); err != nil {
			t.Errorf("Loaded key pair is invalid: %v", err)
		}
	})

	t.Run("load nonexistent private key", func(t *testing.T) {
		// Create config with invalid path
		invalidConfig := &config.RSAConfig{
			PrivateKeyPath: "nonexistent/private.pem",
			PublicKeyPath:  "test/public.pem",
			KeySize:        2048,
		}

		_, _, _, err := LoadRSAKeys(vaultRoot, invalidConfig)
		if err == nil {
			t.Error("LoadRSAKeys() expected error for nonexistent private key")
		}
	})

	t.Run("load nonexistent public key", func(t *testing.T) {
		// Create config with invalid path
		invalidConfig := &config.RSAConfig{
			PrivateKeyPath: "test/private.pem",
			PublicKeyPath:  "nonexistent/public.pem",
			KeySize:        2048,
		}

		_, _, _, err := LoadRSAKeys(vaultRoot, invalidConfig)
		if err == nil {
			t.Error("LoadRSAKeys() expected error for nonexistent public key")
		}
	})
}

func TestParseRSAPrivateKeyFromPEM(t *testing.T) {
	// Generate a test key
	privateKey, _ := testutil.CreateTestRSAKeyPair(t, 2048)

	// Encode to PEM
	pemData := EncodeRSAPrivateKeyToPEM(privateKey)

	t.Run("parse valid PEM", func(t *testing.T) {
		parsedKey, err := ParseRSAPrivateKeyFromPEM(pemData)
		if err != nil {
			t.Errorf("ParseRSAPrivateKeyFromPEM() error = %v", err)
			return
		}

		if parsedKey == nil {
			t.Error("ParseRSAPrivateKeyFromPEM() returned nil key")
			return
		}

		// Verify it's the same key
		if privateKey.N.Cmp(parsedKey.N) != 0 {
			t.Error("Parsed key N does not match original")
		}
		if privateKey.E != parsedKey.E {
			t.Error("Parsed key E does not match original")
		}
	})

	t.Run("parse invalid PEM", func(t *testing.T) {
		invalidPEM := []byte("invalid pem data")
		_, err := ParseRSAPrivateKeyFromPEM(invalidPEM)
		if err == nil {
			t.Error("ParseRSAPrivateKeyFromPEM() expected error for invalid PEM")
		}
	})

	t.Run("parse empty PEM", func(t *testing.T) {
		_, err := ParseRSAPrivateKeyFromPEM([]byte{})
		if err == nil {
			t.Error("ParseRSAPrivateKeyFromPEM() expected error for empty PEM")
		}
	})
}

func TestParseRSAPublicKeyFromPEM(t *testing.T) {
	// Generate a test key
	_, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

	// Encode to PEM
	pemData, err := EncodeRSAPublicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("Failed to encode public key to PEM: %v", err)
	}

	t.Run("parse valid PEM", func(t *testing.T) {
		parsedKey, err := ParseRSAPublicKeyFromPEM(pemData)
		if err != nil {
			t.Errorf("ParseRSAPublicKeyFromPEM() error = %v", err)
			return
		}

		if parsedKey == nil {
			t.Error("ParseRSAPublicKeyFromPEM() returned nil key")
			return
		}

		// Verify it's the same key
		if publicKey.N.Cmp(parsedKey.N) != 0 {
			t.Error("Parsed key N does not match original")
		}
		if publicKey.E != parsedKey.E {
			t.Error("Parsed key E does not match original")
		}
	})

	t.Run("parse invalid PEM", func(t *testing.T) {
		invalidPEM := []byte("invalid pem data")
		_, err := ParseRSAPublicKeyFromPEM(invalidPEM)
		if err == nil {
			t.Error("ParseRSAPublicKeyFromPEM() expected error for invalid PEM")
		}
	})
}

func TestGetRSAPublicKeyFingerprint(t *testing.T) {
	// Generate test key
	_, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

	fingerprint, err := GetRSAPublicKeyFingerprint(publicKey)
	if err != nil {
		t.Errorf("GetRSAPublicKeyFingerprint() error = %v", err)
		return
	}

	if fingerprint == "" {
		t.Error("GetRSAPublicKeyFingerprint() returned empty fingerprint")
	}

	// Test consistency - same key should produce same fingerprint
	fingerprint2, err := GetRSAPublicKeyFingerprint(publicKey)
	if err != nil {
		t.Errorf("GetRSAPublicKeyFingerprint() second call error = %v", err)
		return
	}

	if fingerprint != fingerprint2 {
		t.Error("GetRSAPublicKeyFingerprint() returned different fingerprints for same key")
	}

	// Test that different keys produce different fingerprints
	_, publicKey2 := testutil.CreateTestRSAKeyPair(t, 2048)
	fingerprint3, err := GetRSAPublicKeyFingerprint(publicKey2)
	if err != nil {
		t.Errorf("GetRSAPublicKeyFingerprint() third call error = %v", err)
		return
	}

	if fingerprint == fingerprint3 {
		t.Error("GetRSAPublicKeyFingerprint() returned same fingerprint for different keys")
	}
}

func TestValidateRSAKeyPair(t *testing.T) {
	t.Run("valid key pair", func(t *testing.T) {
		privateKey, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

		err := ValidateRSAKeyPair(privateKey, publicKey)
		if err != nil {
			t.Errorf("ValidateRSAKeyPair() error = %v", err)
		}
	})

	t.Run("mismatched key pair", func(t *testing.T) {
		privateKey1, _ := testutil.CreateTestRSAKeyPair(t, 2048)
		_, publicKey2 := testutil.CreateTestRSAKeyPair(t, 2048)

		err := ValidateRSAKeyPair(privateKey1, publicKey2)
		if err == nil {
			t.Error("ValidateRSAKeyPair() expected error for mismatched keys")
		}
	})

	t.Run("invalid private key", func(t *testing.T) {
		// Create an invalid private key
		invalidPrivateKey := &rsa.PrivateKey{}
		_, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

		err := ValidateRSAKeyPair(invalidPrivateKey, publicKey)
		if err == nil {
			t.Error("ValidateRSAKeyPair() expected error for invalid private key")
		}
	})
}

func TestGenerateTestRSAKeyPair(t *testing.T) {
	tests := []struct {
		name     string
		bits     int
		expected int
	}{
		{
			name:     "2048 bits",
			bits:     2048,
			expected: 2048,
		},
		{
			name:     "4096 bits",
			bits:     4096,
			expected: 4096,
		},
		{
			name:     "small key enforced to minimum",
			bits:     1024,
			expected: 2048, // Should be enforced to minimum
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privateKey, publicKey, err := GenerateTestRSAKeyPair(tt.bits)
			if err != nil {
				t.Errorf("GenerateTestRSAKeyPair() error = %v", err)
				return
			}

			if privateKey == nil {
				t.Error("GenerateTestRSAKeyPair() returned nil private key")
			}
			if publicKey == nil {
				t.Error("GenerateTestRSAKeyPair() returned nil public key")
			}

			// Verify key size
			actualBits := privateKey.N.BitLen()
			if actualBits < tt.expected-10 || actualBits > tt.expected+10 {
				t.Errorf("GenerateTestRSAKeyPair() key size = %d bits, expected around %d bits", actualBits, tt.expected)
			}

			// Verify key pair is valid
			if err := ValidateRSAKeyPair(privateKey, publicKey); err != nil {
				t.Errorf("Generated test key pair is invalid: %v", err)
			}
		})
	}
}

func TestEncodeRSAPrivateKeyToPEM(t *testing.T) {
	privateKey, _ := testutil.CreateTestRSAKeyPair(t, 2048)

	pemData := EncodeRSAPrivateKeyToPEM(privateKey)
	if len(pemData) == 0 {
		t.Error("EncodeRSAPrivateKeyToPEM() returned empty data")
	}

	// Verify we can parse it back
	parsedKey, err := ParseRSAPrivateKeyFromPEM(pemData)
	if err != nil {
		t.Errorf("Failed to parse encoded PEM: %v", err)
		return
	}

	// Verify it's the same key
	if privateKey.N.Cmp(parsedKey.N) != 0 {
		t.Error("Encoded/decoded key N does not match original")
	}
}

func TestEncodeRSAPublicKeyToPEM(t *testing.T) {
	_, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

	pemData, err := EncodeRSAPublicKeyToPEM(publicKey)
	if err != nil {
		t.Errorf("EncodeRSAPublicKeyToPEM() error = %v", err)
		return
	}

	if len(pemData) == 0 {
		t.Error("EncodeRSAPublicKeyToPEM() returned empty data")
	}

	// Verify we can parse it back
	parsedKey, err := ParseRSAPublicKeyFromPEM(pemData)
	if err != nil {
		t.Errorf("Failed to parse encoded PEM: %v", err)
		return
	}

	// Verify it's the same key
	if publicKey.N.Cmp(parsedKey.N) != 0 {
		t.Error("Encoded/decoded key N does not match original")
	}
}

func TestGetPublicKeyFingerprint(t *testing.T) {
	_, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

	fingerprint, err := GetPublicKeyFingerprint(publicKey)
	if err != nil {
		t.Errorf("GetPublicKeyFingerprint() error = %v", err)
		return
	}

	if fingerprint == "" {
		t.Error("GetPublicKeyFingerprint() returned empty fingerprint")
	}

	// Should be consistent with GetRSAPublicKeyFingerprint
	fingerprint2, err := GetRSAPublicKeyFingerprint(publicKey)
	if err != nil {
		t.Errorf("GetRSAPublicKeyFingerprint() error = %v", err)
		return
	}

	if fingerprint != fingerprint2 {
		t.Error("GetPublicKeyFingerprint() and GetRSAPublicKeyFingerprint() returned different results")
	}
}

func TestExportRSAPublicKeyToPEM(t *testing.T) {
	_, publicKey := testutil.CreateTestRSAKeyPair(t, 2048)

	pemData, err := ExportRSAPublicKeyToPEM(publicKey)
	if err != nil {
		t.Errorf("ExportRSAPublicKeyToPEM() error = %v", err)
		return
	}

	if len(pemData) == 0 {
		t.Error("ExportRSAPublicKeyToPEM() returned empty data")
	}

	// Should be consistent with EncodeRSAPublicKeyToPEM
	pemData2, err := EncodeRSAPublicKeyToPEM(publicKey)
	if err != nil {
		t.Errorf("EncodeRSAPublicKeyToPEM() error = %v", err)
		return
	}

	testutil.CompareBytes(t, pemData2, pemData, "PEM export consistency")
}

// Benchmark tests
func BenchmarkGenerateRSAKeyPair2048(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateTestRSAKeyPair(2048)
		if err != nil {
			b.Fatalf("Failed to generate RSA key pair: %v", err)
		}
	}
}

func BenchmarkGenerateRSAKeyPair4096(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateTestRSAKeyPair(4096)
		if err != nil {
			b.Fatalf("Failed to generate RSA key pair: %v", err)
		}
	}
}

func BenchmarkGetRSAPublicKeyFingerprint(b *testing.B) {
	_, publicKey := testutil.CreateTestRSAKeyPair(nil, 2048)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetRSAPublicKeyFingerprint(publicKey)
		if err != nil {
			b.Fatalf("Failed to get fingerprint: %v", err)
		}
	}
}
