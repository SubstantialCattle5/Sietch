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

// VerifyPathAndReturnInfo verifies that a path exists and returns its file info
// Accepts regular files, directories, and symlinks
func VerifyPathAndReturnInfo(path string) (os.FileInfo, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not exist: %s", path)
		}
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied: %s", path)
		}
		return nil, fmt.Errorf("error accessing path: %v", err)
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

// CollectFilesRecursively recursively collects all regular files from a directory
// Follows symlinks and adds their target files
func CollectFilesRecursively(path string) ([]string, error) {
	var files []string

	// Get file info (follows symlinks by default)
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("error accessing %s: %v", path, err)
	}

	// If it's a regular file, return it directly
	if fileInfo.Mode().IsRegular() {
		return []string{path}, nil
	}

	// If it's a directory, walk through it
	if fileInfo.IsDir() {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				// Check if it's a permission error
				if os.IsPermission(err) {
					fmt.Printf("Warning: Permission denied for %s, skipping\n", filePath)
					return nil // Skip this file but continue walking
				}
				return err
			}

			// Check if it's a symlink
			if info.Mode()&os.ModeSymlink != 0 {
				// Resolve symlink
				target, err := os.Readlink(filePath)
				if err != nil {
					fmt.Printf("Warning: Cannot read symlink %s: %v, skipping\n", filePath, err)
					return nil
				}

				// Make target path absolute if it's relative
				if !filepath.IsAbs(target) {
					target = filepath.Join(filepath.Dir(filePath), target)
				}

				// Get info about the symlink target
				targetInfo, err := os.Stat(target)
				if err != nil {
					fmt.Printf("Warning: Cannot access symlink target %s -> %s: %v, skipping\n", filePath, target, err)
					return nil
				}

				// If target is a regular file, add it
				if targetInfo.Mode().IsRegular() {
					files = append(files, target)
				} else if targetInfo.IsDir() {
					// If target is a directory, recursively collect files from it
					dirFiles, err := CollectFilesRecursively(target)
					if err != nil {
						fmt.Printf("Warning: Error collecting files from symlinked directory %s: %v, skipping\n", target, err)
						return nil
					}
					files = append(files, dirFiles...)
				}
				return nil
			}

			// Add regular files
			if info.Mode().IsRegular() {
				files = append(files, filePath)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}

		return files, nil
	}

	// If it's a symlink (shouldn't happen as os.Stat follows symlinks, but just in case)
	return nil, fmt.Errorf("unexpected file type for %s", path)
}
