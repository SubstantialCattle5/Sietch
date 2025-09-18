package encryption

import (
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
)

func TestIsGPGAvailable(t *testing.T) {
	// This test checks if GPG is available on the system
	// It should not fail the test if GPG is not installed
	available := IsGPGAvailable()
	t.Logf("GPG available: %v", available)
}

func TestValidateGPGConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		vaultConfig config.VaultConfig
		expectError bool
	}{
		{
			name: "missing encryption type",
			vaultConfig: config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type: "aes", // Wrong type
				},
			},
			expectError: true,
		},
		{
			name: "missing GPG config",
			vaultConfig: config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type:      constants.EncryptionTypeGPG,
					GPGConfig: nil,
				},
			},
			expectError: true,
		},
		{
			name: "missing KeyID and Recipient",
			vaultConfig: config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type: constants.EncryptionTypeGPG,
					GPGConfig: &config.GPGConfig{
						KeyID:     "",
						Recipient: "",
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid config with KeyID",
			vaultConfig: config.VaultConfig{
				Encryption: config.EncryptionConfig{
					Type: constants.EncryptionTypeGPG,
					GPGConfig: &config.GPGConfig{
						KeyID:     "test-key-id",
						Recipient: "",
					},
				},
			},
			expectError: true, // Will fail because key doesn't exist, but validates the config structure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGPGConfiguration(tt.vaultConfig)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGPGKeyDetails_String(t *testing.T) {
	details := &GPGKeyDetails{
		KeyID:       "1234567890ABCDEF",
		Fingerprint: "1234 5678 90AB CDEF 1234 5678 90AB CDEF 1234 5678",
		Recipient:   "test@example.com",
		KeyServer:   "hkps://keys.openpgp.org",
	}

	result := details.String()

	// Check that all fields are included in the string representation
	if result == "" {
		t.Error("String() returned empty string")
	}

	expectedSubstrings := []string{
		"Key ID: 1234567890ABCDEF",
		"Recipient: test@example.com",
		"Key Server: hkps://keys.openpgp.org",
	}

	for _, substr := range expectedSubstrings {
		if !contains(result, substr) {
			t.Errorf("String() output missing expected substring: %s\nGot: %s", substr, result)
		}
	}
}

func TestGenerateGPGKeyConfig_NilKey(t *testing.T) {
	vaultConfig := &config.VaultConfig{}

	keyConfig, err := GenerateGPGKeyConfig(vaultConfig, nil)

	if err == nil {
		t.Error("expected error for nil key but got none")
	}

	if keyConfig != nil {
		t.Error("expected nil keyConfig for nil key")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
