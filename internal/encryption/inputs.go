package encryption

import (
	"fmt"

	"github.com/manifoldco/promptui"
	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/encryption/keys"
)

// PromptSecurityConfig asks for security-related configuration
func PromptSecurityConfig(configuration *config.VaultConfig) error {
	// Initialize encryption config if not exists
	if configuration.Encryption.Type == "" {
		configuration.Encryption = config.EncryptionConfig{}
	}

	if err := promptEncryptionType(configuration); err != nil {
		return err
	}

	if configuration.Encryption.Type == "aes" {
		if err := promptAESOptions(configuration); err != nil {
			return err
		}
	} else if configuration.Encryption.Type == "gpg" {
		if err := promptGPGOptions(configuration); err != nil {
			return err
		}
	} else {
		fmt.Println("⚠️  Warning: No encryption will be used. Only suitable for testing.")
	}
	return nil
}

// promptEncryptionType asks the user to select an encryption type
func promptEncryptionType(configuration *config.VaultConfig) error {
	keyTypePrompt := promptui.Select{
		Label: "Encryption type",
		Items: []string{"aes", "gpg", "none"},
		Templates: &promptui.SelectTemplates{
			Selected: "Encryption type: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "aes" }}AES-256 encryption (recommended for most users)
{{ else if eq . "gpg" }}GPG encryption (use your existing GPG keys)
{{ else if eq . "none" }}No encryption (not recommended for sensitive data){{ end }}
`,
		},
	}

	_, keyTypeResult, err := keyTypePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.Type = keyTypeResult
	return nil
}

func promptGPGOptions(configuration *config.VaultConfig) error {
	panic("unimplemented")
}

func promptAESOptions(configuration *config.VaultConfig) error {
	// Initialize AES config if not exists
	if configuration.Encryption.AESConfig == nil {
		configuration.Encryption.AESConfig = &config.AESConfig{}
	}

	if err := promptAESMode(configuration); err != nil {
		return err
	}

	if err := promptPassphraseProtection(configuration); err != nil {
		return err
	}

	if configuration.Encryption.PassphraseProtected {
		if err := promptKDFOptions(configuration); err != nil {
			return err
		}

		// Prompt for passphrase
		passphrase, err := promptForPassphrase(true)
		if err != nil {
			return fmt.Errorf("failed to get passphrase: %w", err)
		}

		// Generate key configuration
		keyConfig, err := keys.GenerateAESKey(configuration, passphrase)
		if err != nil {
			return fmt.Errorf("failed to generate AES key: %w", err)
		}

		// Store key configuration
		configuration.Encryption.AESConfig = keyConfig.AESConfig
		configuration.Encryption.KeyHash = keyConfig.KeyHash
		configuration.Encryption.AESConfig.Salt = keyConfig.Salt
	} else {
		if err := promptKeyFileOptions(configuration); err != nil {
			return err
		}

		// Generate key configuration for key file or random key
		keyConfig, err := keys.GenerateAESKey(configuration, "")
		if err != nil {
			return fmt.Errorf("failed to generate AES key: %w", err)
		}

		// Store key configuration
		configuration.Encryption.AESConfig = keyConfig.AESConfig
		configuration.Encryption.KeyHash = keyConfig.KeyHash
	}

	return nil
}

// promptForPassphrase prompts the user for a passphrase
func promptForPassphrase(confirm bool) (string, error) {
	promptLabel := "Enter passphrase"
	if confirm {
		promptLabel = "Create new passphrase"
	}

	passphrasePrompt := promptui.Prompt{
		Label: promptLabel,
		Mask:  '*',
		Validate: func(input string) error {
			if len(input) < 8 {
				return fmt.Errorf("passphrase must be at least 8 characters")
			}
			return nil
		},
	}

	passphrase, err := passphrasePrompt.Run()
	if err != nil {
		return "", fmt.Errorf("passphrase prompt failed: %w", err)
	}

	if confirm {
		confirmPrompt := promptui.Prompt{
			Label: "Confirm passphrase",
			Mask:  '*',
			Validate: func(input string) error {
				if input != passphrase {
					return fmt.Errorf("passphrases do not match")
				}
				return nil
			},
		}

		_, err = confirmPrompt.Run()
		if err != nil {
			return "", fmt.Errorf("passphrase confirmation failed: %w", err)
		}
	}

	return passphrase, nil
}

// promptAESMode asks for AES encryption mode
func promptAESMode(configuration *config.VaultConfig) error {
	aesModePrompt := promptui.Select{
		Label: "AES encryption mode",
		Items: []string{"gcm", "cbc"},
		Templates: &promptui.SelectTemplates{
			Selected: "AES mode: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "gcm" }}GCM mode (authenticated encryption, recommended)
{{ else if eq . "cbc" }}CBC mode (compatibility with older systems){{ end }}
`,
		},
	}

	_, aesMode, err := aesModePrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.AESConfig.Mode = aesMode
	return nil
}

// promptPassphraseProtection asks if the vault should be protected with a passphrase
func promptPassphraseProtection(configuration *config.VaultConfig) error {
	passphrasePrompt := promptui.Prompt{
		Label:     "Protect with passphrase",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := passphrasePrompt.Run()
	configuration.Encryption.PassphraseProtected = (err == nil)
	return nil
}

// promptKDFOptions handles configuration of the key derivation function
func promptKDFOptions(configuration *config.VaultConfig) error {
	kdfPrompt := promptui.Select{
		Label: "Key derivation function",
		Items: []string{"scrypt", "pbkdf2"},
		Templates: &promptui.SelectTemplates{
			Selected: "KDF: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
{{ if eq . "scrypt" }}Scrypt (memory-hard, recommended)
{{ else if eq . "pbkdf2" }}PBKDF2 (more compatible, less secure){{ end }}
`,
		},
	}

	_, kdf, err := kdfPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.AESConfig.KDF = kdf

	if kdf == "scrypt" {
		return promptScryptParameters(configuration)
	}
	return promptPBKDF2Parameters(configuration)
}

// promptScryptParameters handles configuration of scrypt parameters
func promptScryptParameters(configuration *config.VaultConfig) error {
	advancedPrompt := promptui.Prompt{
		Label:     "Configure advanced scrypt parameters",
		IsConfirm: true,
		Default:   "n",
	}

	_, err := advancedPrompt.Run()
	if err == nil { // User selected yes
		return promptAdvancedScryptParameters(configuration)
	}
	// Default scrypt parameters
	configuration.Encryption.AESConfig.ScryptN = 32768
	configuration.Encryption.AESConfig.ScryptR = 8
	configuration.Encryption.AESConfig.ScryptP = 1
	return nil
}

// promptAdvancedScryptParameters handles configuration of advanced scrypt parameters
func promptAdvancedScryptParameters(configuration *config.VaultConfig) error {
	// Scrypt N parameter
	nPrompt := promptui.Select{
		Label: "Scrypt N parameter (CPU/memory cost)",
		Items: []string{"16384", "32768", "65536", "131072"},
		Templates: &promptui.SelectTemplates{
			Selected: "N: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
Higher values are more secure but slower. Values:
- 16384: Fast, lower security
- 32768: Balanced (recommended)
- 65536: More secure, slower
- 131072: Most secure, much slower
`,
		},
	}

	nIdx, _, err := nPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Convert selection to numeric values
	nValues := []int{16384, 32768, 65536, 131072}
	configuration.Encryption.AESConfig.ScryptN = nValues[nIdx]

	// Scrypt r and p parameters
	configuration.Encryption.AESConfig.ScryptR = 8
	configuration.Encryption.AESConfig.ScryptP = 1

	return nil
}

// promptPBKDF2Parameters handles configuration of PBKDF2 parameters
func promptPBKDF2Parameters(configuration *config.VaultConfig) error {
	// PBKDF2 iterations
	iterPrompt := promptui.Select{
		Label: "PBKDF2 iterations",
		Items: []string{"100000", "200000", "500000", "1000000"},
		Templates: &promptui.SelectTemplates{
			Selected: "Iterations: {{ . }}",
			Active:   "▸ {{ . }}",
			Inactive: "  {{ . }}",
			Details: `
{{ "Details:" | faint }}
Higher values are more secure but slower. Values:
- 100000: Fast, lower security
- 200000: Balanced (recommended)
- 500000: More secure, slower
- 1000000: Most secure, much slower
`,
		},
	}

	iterIdx, _, err := iterPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}

	// Convert selection to numeric values
	iterValues := []int{100000, 200000, 500000, 1000000}
	configuration.Encryption.AESConfig.PBKDF2I = iterValues[iterIdx]

	return nil
}

// promptKeyFileOptions handles configuration when not using passphrase protection
func promptKeyFileOptions(configuration *config.VaultConfig) error {
	// Ask if user wants to use an existing key file
	keyFilePrompt := promptui.Prompt{
		Label:     "Use existing key file",
		IsConfirm: true,
		Default:   "n",
	}

	_, err := keyFilePrompt.Run()
	if err == nil { // User selected yes
		return promptExistingKeyFile(configuration)
	}
	return promptRandomKeyGeneration(configuration)
}

// promptExistingKeyFile handles configuration for using an existing key file
func promptExistingKeyFile(configuration *config.VaultConfig) error {
	configuration.Encryption.KeyFile = true

	// Get path to key file
	pathPrompt := promptui.Prompt{
		Label:     "Path to key file",
		Default:   "~/.keys/sietch.key",
		AllowEdit: true,
	}

	keyPath, err := pathPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.KeyFilePath = keyPath

	return nil
}

// promptRandomKeyGeneration handles configuration for random key generation
func promptRandomKeyGeneration(configuration *config.VaultConfig) error {
	// Random key generation
	configuration.Encryption.RandomKey = true

	// Ask about backing up the key
	backupPrompt := promptui.Prompt{
		Label:     "Backup random key to file",
		IsConfirm: true,
		Default:   "y",
	}

	_, err := backupPrompt.Run()
	if err == nil { // User selected yes
		return promptKeyBackupPath(configuration)
	}

	// Make sure KeyBackupPath is empty if not backing up
	configuration.Encryption.KeyBackupPath = ""
	return nil
}

// promptKeyBackupPath handles configuration for key backup path
func promptKeyBackupPath(configuration *config.VaultConfig) error {
	pathPrompt := promptui.Prompt{
		Label:     "Backup file path",
		Default:   "~/sietch-key-backup.bin",
		AllowEdit: true,
	}

	backupPath, err := pathPrompt.Run()
	if err != nil {
		return fmt.Errorf("prompt failed: %w", err)
	}
	configuration.Encryption.KeyBackupPath = backupPath
	return nil
}
