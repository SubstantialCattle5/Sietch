package config

import "time"

// VaultConfig represents the structure for vault.yaml
type VaultConfig struct {
	Name      string    `yaml:"name"`
	VaultID   string    `yaml:"vault_id"`
	CreatedAt time.Time `yaml:"created_at"`

	Metadata struct {
		Author string   `yaml:"author"`
		Tags   []string `yaml:"tags"`
	} `yaml:"metadata"`
}

func BuildVaultConfig(vaultID, vaultName, author, keyPath string, tags []string) VaultConfig {
	config := VaultConfig{
		VaultID:   vaultID,
		Name:      vaultName,
		CreatedAt: time.Now().UTC(),
	}

	config.Metadata.Author = author
	config.Metadata.Tags = tags

	return config
}
