package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

func PrepareVaultPath(vaultPath string, vaultName string, forceInit bool) (string, error) {
	absVaultPath, err := filepath.Abs(filepath.Join(vaultPath, vaultName))
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if vault already exists by checking for .sietch directory
	sietchDir := filepath.Join(absVaultPath, ".sietch")
	if _, err := os.Stat(sietchDir); err == nil {
		if !forceInit {
			return "", fmt.Errorf("vault already exists at %s. Use --force to re-initialize (warning: this will destroy existing data)", absVaultPath)
		}
		// If force is true, we'll continue and overwrite
	}

	return absVaultPath, nil
}
