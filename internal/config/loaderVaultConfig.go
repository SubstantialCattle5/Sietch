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
	configPath := filepath.Join(vaultPath, "vault.yml")

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

	// Validate configuration
	// if err := validateConfig(&config); err != nil {
	// 	return nil, fmt.Errorf("invalid vault configuration: %w", err)
	// }

	return &config, nil
}
