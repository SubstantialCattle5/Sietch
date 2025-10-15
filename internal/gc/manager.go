package gc

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/deduplication"
	"github.com/substantialcattle5/sietch/internal/fs"
)

// Manager manages automatic garbage collection for Sietch vaults
type Manager struct {
	vaultRoot string
	monitor   *Monitor
	config    config.VaultConfig
	started   bool
}

// NewManager creates a new GC manager for a vault
func NewManager(vaultRoot string) (*Manager, error) {
	// Check if vault is initialized
	if !fs.IsVaultInitialized(vaultRoot) {
		return nil, fmt.Errorf("vault not initialized")
	}

	// Load vault configuration
	vaultConfig, err := config.LoadVaultConfig(vaultRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to load vault configuration: %w", err)
	}

	return &Manager{
		vaultRoot: vaultRoot,
		config:    *vaultConfig,
		started:   false,
	}, nil
}

// Start begins automatic GC monitoring for the vault
func (m *Manager) Start(ctx context.Context) error {
	if m.started {
		return fmt.Errorf("GC manager already started")
	}

	// Check if automatic GC is enabled
	if !m.config.Deduplication.AutoGC.Enabled {
		return fmt.Errorf("automatic GC is disabled in vault configuration")
	}

	// Parse check interval
	checkInterval, err := time.ParseDuration(m.config.Deduplication.AutoGC.CheckInterval)
	if err != nil {
		return fmt.Errorf("invalid check interval '%s': %w", m.config.Deduplication.AutoGC.CheckInterval, err)
	}

	// Create monitor configuration
	monitorConfig := MonitorConfig{
		Enabled:         m.config.Deduplication.AutoGC.Enabled,
		CheckInterval:   checkInterval,
		AutoGCThreshold: m.getEffectiveGCThreshold(),
		EnableLogging:   m.config.Deduplication.AutoGC.EnableLogging,
		LogFile:         m.getEffectiveLogFile(),
		AlertThreshold:  m.config.Deduplication.AutoGC.AlertThreshold,
		AlertWebhook:    m.config.Deduplication.AutoGC.AlertWebhook,
	}

	// Create and start monitor
	monitor, err := NewMonitor(m.vaultRoot, monitorConfig, m.config.Deduplication)
	if err != nil {
		return fmt.Errorf("failed to create GC monitor: %w", err)
	}

	if err := monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start GC monitor: %w", err)
	}

	m.monitor = monitor
	m.started = true

	return nil
}

// Stop halts automatic GC monitoring
func (m *Manager) Stop() error {
	if !m.started {
		return fmt.Errorf("GC manager not started")
	}

	err := m.monitor.Stop()
	m.started = false
	return err
}

// IsRunning returns whether the GC manager is currently running
func (m *Manager) IsRunning() bool {
	return m.started && m.monitor.IsRunning()
}

// GetStats returns current deduplication statistics
func (m *Manager) GetStats() (deduplication.DeduplicationStats, error) {
	if !m.started {
		return deduplication.DeduplicationStats{}, fmt.Errorf("GC manager not started")
	}
	return m.monitor.GetStats(), nil
}

// getEffectiveGCThreshold returns the effective GC threshold to use
func (m *Manager) getEffectiveGCThreshold() int {
	if m.config.Deduplication.AutoGC.AutoGCThreshold > 0 {
		return m.config.Deduplication.AutoGC.AutoGCThreshold
	}
	return m.config.Deduplication.GCThreshold
}

// getEffectiveLogFile returns the effective log file path
func (m *Manager) getEffectiveLogFile() string {
	if m.config.Deduplication.AutoGC.LogFile != "" {
		// Convert relative path to absolute path within vault
		if !filepath.IsAbs(m.config.Deduplication.AutoGC.LogFile) {
			return filepath.Join(m.vaultRoot, m.config.Deduplication.AutoGC.LogFile)
		}
		return m.config.Deduplication.AutoGC.LogFile
	}
	return filepath.Join(m.vaultRoot, ".sietch", "logs", "gc.log")
}

// Global variable to hold the active GC manager
var activeManager *Manager

// StartGlobalGC starts automatic GC for the vault in the current directory
func StartGlobalGC(ctx context.Context) error {
	vaultRoot, err := fs.FindVaultRoot()
	if err != nil {
		// Not in a vault, silently ignore
		return nil
	}

	manager, err := NewManager(vaultRoot)
	if err != nil {
		return fmt.Errorf("failed to create GC manager: %w", err)
	}

	if err := manager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start GC manager: %w", err)
	}

	activeManager = manager
	return nil
}

// StopGlobalGC stops the global GC manager
func StopGlobalGC() error {
	if activeManager == nil {
		return nil
	}

	err := activeManager.Stop()
	activeManager = nil
	return err
}

// GetGlobalGCManager returns the active GC manager
func GetGlobalGCManager() *Manager {
	return activeManager
}
