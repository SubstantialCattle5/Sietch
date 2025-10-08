package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDirectory ensures a directory exists, creating it if necessary
func EnsureDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0o755)
	} else if err != nil {
		return err
	}
	return nil
}

// IsVaultInitialized checks if the given path contains an initialized vault
func IsVaultInitialized(basePath string) bool {
	sietchDir := filepath.Join(basePath, ".sietch")
	vaultYaml := filepath.Join(basePath, "vault.yaml")

	// Check if both the .sietch directory and vault.yaml exist
	sietchInfo, sietchErr := os.Stat(sietchDir)
	vaultInfo, vaultErr := os.Stat(vaultYaml)

	return sietchErr == nil && sietchInfo.IsDir() &&
		vaultErr == nil && !vaultInfo.IsDir()
}

// GetChunkDirectory returns the path to the chunks directory
func GetChunkDirectory(basePath string) string {
	return filepath.Join(basePath, ".sietch", "chunks")
}

// GetManifestDirectory returns the path to the manifests directory
func GetManifestDirectory(basePath string) string {
	return filepath.Join(basePath, ".sietch", "manifests")
}

// findVaultRoot traverses up the directory tree to find a vault root
func FindVaultRoot() (string, error) {
	// Start from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Traverse up until we find vault.yaml
	for {
		if IsVaultInitialized(currentDir) {
			return currentDir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// We've reached the root directory
			return "", fmt.Errorf("no vault found in the current path hierarchy")
		}
		currentDir = parentDir
	}
}

func VerifyFileAndReturnFileInfo(filePath string) (os.FileInfo, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file does not exist: %s", filePath)
		}
		return nil, fmt.Errorf("error accessing file: %v", err)
	}

	// Verify it's a regular file, not a directory or symlink
	if !fileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filePath)
	}
	return fileInfo, nil
}

func VerifyFileAndReturnFile(filePath string) (*os.File, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found at %s", filePath)
		}
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	return file, nil
}
