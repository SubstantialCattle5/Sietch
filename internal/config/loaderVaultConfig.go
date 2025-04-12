/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func LoadVaultConfig(vaultPath string) (*VaultConfig, error) {
	// Change from vault.yml to vault.yaml to match the actual file name
	configPath := filepath.Join(vaultPath, "vault.yaml")

	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault configuration not found at %s", configPath)
		}
		return nil, fmt.Errorf("error accessing vault configuration: %w", err)
	}

	// Read the file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading vault configuration: %w", err)
	}

	var config VaultConfig
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing vault configuration: %w", err)
	}

	return &config, nil
}
