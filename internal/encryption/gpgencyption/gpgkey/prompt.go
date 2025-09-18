package gpgkey

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
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

// promptForKeySelection allows user to select an existing key or create a new one
func PromptForKeySelection(keys []*GPGKeyInfo) (*GPGKeyInfo, error) {
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
func PromptForNewKeyCreation() (*GPGKeyInfo, error) {
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

	keyInfo, err := GenerateGPGKey(name, email, keyType, expiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate GPG key: %w", err)
	}

	fmt.Printf("✓ GPG key generated successfully!\n")
	fmt.Printf("  Key ID: %s\n", keyInfo.KeyID)
	fmt.Printf("  Fingerprint: %s\n", keyInfo.Fingerprint)

	return keyInfo, nil
}

// promptForKeyServer prompts for optional key server configuration
func PromptForKeyServer(configuration *config.VaultConfig) error {
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
