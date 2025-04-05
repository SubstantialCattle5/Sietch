package manifest

import (
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/config"
	"gopkg.in/yaml.v3"
)

func WriteManifest(basePath string, config config.VaultConfig) error {
	manifestPath := filepath.Join(basePath, "vault.yaml")

	manifestFile, err := os.Create(manifestPath)
	if err != nil {
		return err
	}
	defer manifestFile.Close()
	encoder := yaml.NewEncoder(manifestFile)
	encoder.SetIndent(2)
	return encoder.Encode(config)
}
