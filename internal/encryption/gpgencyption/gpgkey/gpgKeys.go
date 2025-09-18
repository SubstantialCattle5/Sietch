package gpgkey

import (
	"fmt"
	"os/exec"
	"strings"
)

// generateGPGKey creates a new GPG key with the specified parameters
func GenerateGPGKey(name, email, keyType, expiration string) (*GPGKeyInfo, error) {
	// Map key type to GPG parameters
	var algorithm, keyLength string
	switch keyType {
	case "RSA 4096":
		algorithm = "rsa"
		keyLength = "4096"
	case "RSA 2048":
		algorithm = "rsa"
		keyLength = "2048"
	case "Ed25519":
		algorithm = "ed25519"
		keyLength = ""
	default:
		algorithm = "rsa"
		keyLength = "4096"
	}

	// Map expiration to GPG format
	var expire string
	switch expiration {
	case "1 year":
		expire = "1y"
	case "2 years":
		expire = "2y"
	case "5 years":
		expire = "5y"
	case "Never expires":
		expire = "0"
	default:
		expire = "1y"
	}

	// Create GPG key generation batch file content
	batchContent := fmt.Sprintf(`Key-Type: %s
Key-Length: %s
Subkey-Type: %s
Subkey-Length: %s
Name-Real: %s
Name-Email: %s
Expire-Date: %s
%%commit
%%echo done
`, algorithm, keyLength, algorithm, keyLength, name, email, expire)

	// For Ed25519, adjust the batch content
	if algorithm == "ed25519" {
		batchContent = fmt.Sprintf(`Key-Type: EDDSA
Key-Curve: Ed25519
Subkey-Type: ECDH
Subkey-Curve: Curve25519
Name-Real: %s
Name-Email: %s
Expire-Date: %s
%%commit
%%echo done
`, name, email, expire)
	}

	// Execute GPG key generation
	cmd := exec.Command("gpg", "--batch", "--generate-key")
	cmd.Stdin = strings.NewReader(batchContent)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("GPG key generation failed: %w\nOutput: %s", err, string(output))
	}

	// Retrieve the newly created key
	keys, err := ListGPGKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys after generation: %w", err)
	}

	// Find the key that matches our email
	for _, key := range keys {
		if key.Email == email {
			return key, nil
		}
	}

	return nil, fmt.Errorf("could not find the newly generated key")
}
