package config

import "time"

// VaultConfig represents the structure for vault.yaml
type VaultConfig struct {
	VaultID   string    `yaml:"vault_id"`
	Name      string    `yaml:"name"`
	CreatedAt time.Time `yaml:"created_at"`
}

func BuildVaultConfig(vaultID, vaultName, keyPath string) VaultConfig {
	config := VaultConfig{
		VaultID:   vaultID,
		Name:      vaultName,
		CreatedAt: time.Now().UTC(),
	}

	return config

}
