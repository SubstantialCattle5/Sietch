package gpgkey

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/passphrase"
)

// GPGKeyInfo represents information about a GPG key
type GPGKeyInfo struct {
	KeyID       string
	Fingerprint string
	UserID      string
	Email       string
	KeyType     string
	Expired     bool
}

func PromptGPGOptions(configuration *config.VaultConfig) error {
	// Initialize GPG config if not exists
	if configuration.Encryption.GPGConfig == nil {
		configuration.Encryption.GPGConfig = &config.GPGConfig{}
	}

	// Check if GPG is available
	if !isGPGAvailable() {
		return fmt.Errorf("GPG is not available on this system. Please install GPG first")
	}

	// Get available GPG keys
	keys, err := listGPGKeys()
	if err != nil {
		return fmt.Errorf("failed to list GPG keys: %w", err)
	}

	// Prompt for key selection or creation
	selectedKey, err := promptForKeySelection(keys)
	if err != nil {
		return err
	}

	if selectedKey == nil {
		// User chose to create a new key
		newKey, err := promptForNewKeyCreation()
		if err != nil {
			return err
		}
		selectedKey = newKey
	}

	// Store the selected key configuration
	configuration.Encryption.GPGConfig.KeyID = selectedKey.KeyID
	configuration.Encryption.GPGConfig.Recipient = selectedKey.Email
	configuration.Encryption.KeyHash = selectedKey.Fingerprint

	// Prompt for passphrase protection (for private key access)
	if err := passphrase.PromptPassphraseProtection(configuration); err != nil {
		return err
	}

	// Configure key server (optional)
	if err := promptForKeyServer(configuration); err != nil {
		return err
	}

	fmt.Printf("✓ GPG key configured: %s (%s)\n", selectedKey.UserID, selectedKey.KeyID)

	return nil
}

// isGPGAvailable checks if GPG is installed and available
func isGPGAvailable() bool {
	cmd := exec.Command("gpg", "--version")
	return cmd.Run() == nil
}

// listGPGKeys retrieves available GPG keys from the keyring
func listGPGKeys() ([]*GPGKeyInfo, error) {
	cmd := exec.Command("gpg", "--list-keys", "--with-colons")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gpg --list-keys: %w", err)
	}

	return parseGPGKeyList(string(output)), nil
}

// parseGPGKeyList parses GPG key list output and extracts key information
func parseGPGKeyList(output string) []*GPGKeyInfo {
	var keys []*GPGKeyInfo
	lines := strings.Split(output, "\n")

	var currentKey *GPGKeyInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "pub":
			// Public key record
			if len(fields) >= 5 {
				currentKey = &GPGKeyInfo{
					KeyID:   fields[4],
					KeyType: fields[3],
					Expired: fields[1] == "e",
				}
			}
		case "fpr":
			// Fingerprint record
			if currentKey != nil && len(fields) >= 10 {
				currentKey.Fingerprint = fields[9]
			}
		case "uid":
			// User ID record
			if currentKey != nil && len(fields) >= 10 {
				userID := fields[9]
				currentKey.UserID = userID

				// Extract email from user ID
				emailRegex := regexp.MustCompile(`<([^>]+)>`)
				matches := emailRegex.FindStringSubmatch(userID)
				if len(matches) > 1 {
					currentKey.Email = matches[1]
				}

				// Only add the key when we have complete information
				if currentKey.KeyID != "" && !currentKey.Expired {
					keys = append(keys, currentKey)
				}
				currentKey = nil
			}
		}
	}

	return keys
}

// promptForKeySelection allows user to select an existing key or create a new one
func promptForKeySelection(keys []*GPGKeyInfo) (*GPGKeyInfo, error) {
	if len(keys) == 0 {
		fmt.Println("No GPG keys found in your keyring.")
		return nil, nil
	}

	// Prepare selection options
	items := make([]string, len(keys)+1)
	for i, key := range keys {
		items[i] = fmt.Sprintf("%s (%s) - %s", key.UserID, key.KeyID, key.KeyType)
	}
	items[len(keys)] = "Create a new GPG key"

	prompt := promptui.Select{
		Label: "Select a GPG key for encryption",
		Items: items,
		Templates: &promptui.SelectTemplates{
			Selected: "GPG Key: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "Create a new GPG key" }}Generate a new GPG key pair for vault encryption
{{ else }}Use existing GPG key from your keyring{{ end }}
`,
		},
	}

	index, _, err := prompt.Run()
	if err != nil {
		return nil, fmt.Errorf("key selection failed: %w", err)
	}

	// If user selected "Create a new GPG key"
	if index == len(keys) {
		return nil, nil
	}

	return keys[index], nil
}

// promptForNewKeyCreation guides user through creating a new GPG key
func promptForNewKeyCreation() (*GPGKeyInfo, error) {
	fmt.Println("\n🔑 Creating a new GPG key...")

	// Prompt for name
	namePrompt := promptui.Prompt{
		Label: "Full name",
		Validate: func(input string) error {
			if len(strings.TrimSpace(input)) < 2 {
				return fmt.Errorf("name must be at least 2 characters")
			}
			return nil
		},
	}
	name, err := namePrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("name prompt failed: %w", err)
	}

	// Prompt for email
	emailPrompt := promptui.Prompt{
		Label: "Email address",
		Validate: func(input string) error {
			emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
			if !emailRegex.MatchString(input) {
				return fmt.Errorf("invalid email address")
			}
			return nil
		},
	}
	email, err := emailPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("email prompt failed: %w", err)
	}

	// Prompt for key type
	keyTypePrompt := promptui.Select{
		Label: "Key type",
		Items: []string{"RSA 4096", "RSA 2048", "Ed25519"},
		Templates: &promptui.SelectTemplates{
			Selected: "Key type: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "RSA 4096" }}RSA 4096-bit (recommended for compatibility)
{{ else if eq . "RSA 2048" }}RSA 2048-bit (faster, still secure)
{{ else if eq . "Ed25519" }}Ed25519 (modern, efficient){{ end }}
`,
		},
	}
	_, keyType, err := keyTypePrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("key type selection failed: %w", err)
	}

	// Prompt for key expiration
	expirationPrompt := promptui.Select{
		Label: "Key expiration",
		Items: []string{"1 year", "2 years", "5 years", "Never expires"},
		Templates: &promptui.SelectTemplates{
			Selected: "Expiration: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "Never expires" }}⚠️  Not recommended for production use{{ end }}
`,
		},
	}
	_, expiration, err := expirationPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("expiration selection failed: %w", err)
	}

	// Create the GPG key
	fmt.Println("\n🔄 Generating GPG key... This may take a moment.")

	keyInfo, err := generateGPGKey(name, email, keyType, expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GPG key: %w", err)
	}

	fmt.Printf("✓ GPG key generated successfully!\n")
	fmt.Printf("  Key ID: %s\n", keyInfo.KeyID)
	fmt.Printf("  Fingerprint: %s\n", keyInfo.Fingerprint)

	return keyInfo, nil
}

// generateGPGKey creates a new GPG key with the specified parameters
func generateGPGKey(name, email, keyType, expiration string) (*GPGKeyInfo, error) {
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
	keys, err := listGPGKeys()
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

// promptForKeyServer prompts for optional key server configuration
func promptForKeyServer(configuration *config.VaultConfig) error {
	prompt := promptui.Prompt{
		Label:     "Configure custom key server (optional)",
		IsConfirm: true,
		Default:   "n",
	}

	_, err := prompt.Run()
	if err != nil {
		// User chose not to configure custom key server, use default
		configuration.Encryption.GPGConfig.KeyServer = "hkps://keys.openpgp.org"
		return nil
	}

	// User wants to configure custom key server
	serverPrompt := promptui.Prompt{
		Label:   "Key server URL",
		Default: "hkps://keys.openpgp.org",
		Validate: func(input string) error {
			if !strings.HasPrefix(input, "hkp://") && !strings.HasPrefix(input, "hkps://") {
				return fmt.Errorf("key server URL must start with hkp:// or hkps://")
			}
			return nil
		},
	}

	keyServer, err := serverPrompt.Run()
	if err != nil {
		return fmt.Errorf("key server prompt failed: %w", err)
	}

	configuration.Encryption.GPGConfig.KeyServer = keyServer
	return nil
}
