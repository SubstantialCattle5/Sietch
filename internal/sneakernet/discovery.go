package sneakernet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
)

// DiscoverVaults scans for vaults in common locations
func DiscoverVaults(searchPaths []string) ([]VaultInfo, error) {
	var vaults []VaultInfo

	// Default search paths if none provided
	if len(searchPaths) == 0 {
		defaultPaths, err := getDefaultSearchPaths()
		if err != nil {
			return nil, fmt.Errorf("failed to get default search paths: %v", err)
		}
		searchPaths = defaultPaths
	}

	// Search each path
	for _, searchPath := range searchPaths {
		foundVaults, err := scanForVaults(searchPath)
		if err != nil {
			// Log warning but continue
			fmt.Printf("Warning: Failed to scan %s: %v\n", searchPath, err)
			continue
		}
		vaults = append(vaults, foundVaults...)
	}

	return vaults, nil
}

// getDefaultSearchPaths returns common locations to search for vaults
func getDefaultSearchPaths() ([]string, error) {
	var paths []string

	// Current directory
	currentDir, err := os.Getwd()
	if err == nil {
		paths = append(paths, currentDir)
	}

	// USB mount points (Linux/macOS)
	usbPaths := []string{
		"/media",
		"/mnt",
		"/Volumes", // macOS
	}

	for _, usbPath := range usbPaths {
		if _, err := os.Stat(usbPath); err == nil {
			entries, err := os.ReadDir(usbPath)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				if entry.IsDir() {
					paths = append(paths, filepath.Join(usbPath, entry.Name()))
				}
			}
		}
	}

	// Home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, homeDir)
	}

	return paths, nil
}

// scanForVaults recursively scans a directory for vault structures
func scanForVaults(rootPath string) ([]VaultInfo, error) {
	var vaults []VaultInfo

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors, continue walking
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Check if this directory contains a vault
		if IsValidVault(path) {
			vaultInfo, err := getVaultInfo(path)
			if err != nil {
				fmt.Printf("Warning: Found vault at %s but failed to read info: %v\n", path, err)
				return nil
			}
			vaults = append(vaults, *vaultInfo)

			// Skip scanning subdirectories of vaults
			return filepath.SkipDir
		}

		// Skip deep scanning in certain directories
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, ".") && baseName != ".sietch" {
			return filepath.SkipDir
		}

		return nil
	})

	return vaults, err
}

// IsValidVault checks if a directory contains a valid Sietch vault
func IsValidVault(vaultPath string) bool {
	// Check for .sietch directory
	sietchDir := filepath.Join(vaultPath, ".sietch")
	sietchInfo, err := os.Stat(sietchDir)
	if err != nil || !sietchInfo.IsDir() {
		return false
	}

	// Check for vault.yaml
	vaultConfig := filepath.Join(vaultPath, "vault.yaml")
	if _, err := os.Stat(vaultConfig); err != nil {
		return false
	}

	// Check for chunks directory
	chunksDir := filepath.Join(sietchDir, "chunks")
	if chunksInfo, err := os.Stat(chunksDir); err != nil || !chunksInfo.IsDir() {
		return false
	}

	// Check for manifests directory
	manifestsDir := filepath.Join(sietchDir, "manifests")
	if manifestsInfo, err := os.Stat(manifestsDir); err != nil || !manifestsInfo.IsDir() {
		return false
	}

	return true
}

// getVaultInfo extracts vault information from a vault directory
func getVaultInfo(vaultPath string) (*VaultInfo, error) {
	// Load vault configuration
	vaultConfig, err := config.LoadVaultConfig(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault config: %v", err)
	}

	// Create vault manager to get file count and size
	manager, err := config.NewManager(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault manager: %v", err)
	}

	// Get manifest to calculate statistics
	manifest, err := manager.GetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest: %v", err)
	}

	// Calculate total size
	var totalSize int64
	for _, file := range manifest.Files {
		totalSize += file.Size
	}

	// Get last access time (use vault directory modification time)
	vaultInfo, err := os.Stat(vaultPath)
	var lastAccess time.Time
	if err == nil {
		lastAccess = vaultInfo.ModTime()
	}

	return &VaultInfo{
		Path:       vaultPath,
		Name:       vaultConfig.Name,
		VaultID:    vaultConfig.VaultID,
		FileCount:  len(manifest.Files),
		TotalSize:  totalSize,
		LastAccess: lastAccess,
		CreatedAt:  vaultConfig.CreatedAt,
	}, nil
}

// FindUSBMountPoints returns mounted USB devices (Linux/macOS specific)
func FindUSBMountPoints() ([]string, error) {
	var mountPoints []string

	// Common USB mount points
	usbDirs := []string{"/media", "/mnt", "/Volumes"}

	for _, usbDir := range usbDirs {
		if _, err := os.Stat(usbDir); err != nil {
			continue
		}

		entries, err := os.ReadDir(usbDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				mountPoint := filepath.Join(usbDir, entry.Name())
				// Simple check if it's likely a USB device
				if isLikelyUSBDevice(mountPoint) {
					mountPoints = append(mountPoints, mountPoint)
				}
			}
		}
	}

	return mountPoints, nil
}

// isLikelyUSBDevice performs basic heuristics to detect USB devices
func isLikelyUSBDevice(path string) bool {
	// Check if path is writable (mounted USB devices usually are)
	testFile := filepath.Join(path, ".sietch_test_write")
	file, err := os.Create(testFile)
	if err != nil {
		return false // Not writable, probably not a USB device we can use
	}
	file.Close()
	os.Remove(testFile)

	return true
}
