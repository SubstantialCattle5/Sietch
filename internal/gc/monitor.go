package gc

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/substantialcattle5/sietch/internal/config"
	"github.com/substantialcattle5/sietch/internal/constants"
	"github.com/substantialcattle5/sietch/internal/deduplication"
)

// MonitorConfig contains configuration for the GC monitor
type MonitorConfig struct {
	Enabled         bool          `yaml:"enabled"`
	CheckInterval   time.Duration `yaml:"check_interval"`    // How often to check threshold
	AutoGCThreshold int           `yaml:"auto_gc_threshold"` // Threshold for automatic GC
	EnableLogging   bool          `yaml:"enable_logging"`    // Enable GC logging
	LogFile         string        `yaml:"log_file"`          // Log file path
	MaxConcurrentGC int           `yaml:"max_concurrent_gc"` // Max concurrent GC operations
	AlertThreshold  int           `yaml:"alert_threshold"`   // Alert when unreferenced chunks exceed this
	AlertWebhook    string        `yaml:"alert_webhook"`     // Webhook URL for alerts
}

// DefaultMonitorConfig returns default configuration for GC monitoring
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		Enabled:         true,
		CheckInterval:   1 * time.Hour, // Check every hour
		AutoGCThreshold: 1000,          // Default threshold from vault config
		EnableLogging:   true,
		LogFile:         ".sietch/logs/gc.log",
		MaxConcurrentGC: 1,
		AlertThreshold:  5000,
		AlertWebhook:    "",
	}
}

// Monitor handles automatic garbage collection monitoring
type Monitor struct {
	vaultRoot   string
	config      MonitorConfig
	dedupConfig config.DeduplicationConfig
	manager     *deduplication.Manager
	isRunning   bool
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mutex       sync.Mutex
	logger      *log.Logger
}

// NewMonitor creates a new GC monitor
func NewMonitor(vaultRoot string, monitorConfig MonitorConfig, dedupConfig config.DeduplicationConfig) (*Monitor, error) {
	// Initialize deduplication manager
	manager, err := deduplication.NewManager(vaultRoot, dedupConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create deduplication manager: %w", err)
	}

	monitor := &Monitor{
		vaultRoot:   vaultRoot,
		config:      monitorConfig,
		dedupConfig: dedupConfig,
		manager:     manager,
		isRunning:   false,
		stopChan:    make(chan struct{}),
	}

	// Set default threshold if not configured
	if monitorConfig.AutoGCThreshold == 0 {
		monitorConfig.AutoGCThreshold = dedupConfig.GCThreshold
	}

	// Setup logging if enabled
	if monitorConfig.EnableLogging {
		monitor.setupLogging(monitorConfig.LogFile)
	}

	return monitor, nil
}

// Start begins the GC monitoring process
func (m *Monitor) Start(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.isRunning {
		return fmt.Errorf("GC monitor is already running")
	}

	if !m.config.Enabled {
		return fmt.Errorf("GC monitor is disabled")
	}

	m.isRunning = true
	m.stopChan = make(chan struct{})

	m.wg.Add(1)
	go m.monitorLoop(ctx)

	m.log("GC monitor started with check interval: %v", m.config.CheckInterval)
	return nil
}

// Stop halts the GC monitoring process
func (m *Monitor) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isRunning {
		return fmt.Errorf("GC monitor is not running")
	}

	close(m.stopChan)
	m.wg.Wait()
	m.isRunning = false

	m.log("GC monitor stopped")
	return nil
}

// IsRunning returns whether the monitor is currently running
func (m *Monitor) IsRunning() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.isRunning
}

// monitorLoop is the main monitoring loop
func (m *Monitor) monitorLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.log("GC monitor context cancelled")
			return
		case <-m.stopChan:
			m.log("GC monitor stop signal received")
			return
		case <-ticker.C:
			if err := m.checkAndTriggerGC(); err != nil {
				m.log("Error during GC check: %v", err)
			}
		}
	}
}

// checkAndTriggerGC checks if GC threshold is exceeded and triggers GC if needed
func (m *Monitor) checkAndTriggerGC() error {
	// Get current statistics
	stats := m.manager.GetStats()

	m.log("GC check: %d unreferenced chunks (threshold: %d)",
		stats.UnreferencedChunks, m.config.AutoGCThreshold)

	// Check if threshold is exceeded
	if stats.UnreferencedChunks >= m.config.AutoGCThreshold {
		m.log("GC threshold exceeded (%d >= %d), triggering automatic GC",
			stats.UnreferencedChunks, m.config.AutoGCThreshold)

		// Trigger garbage collection
		if err := m.triggerGC(); err != nil {
			return fmt.Errorf("automatic GC failed: %w", err)
		}

		// Check if we should send alert
		if m.shouldAlert(stats.UnreferencedChunks) {
			m.sendAlert(stats)
		}
	}

	return nil
}

// triggerGC performs garbage collection
func (m *Monitor) triggerGC() error {
	m.log("Starting automatic garbage collection...")

	// Run garbage collection
	removedChunks, err := m.manager.GarbageCollect()
	if err != nil {
		return fmt.Errorf("garbage collection failed: %w", err)
	}

	// Save the updated index
	if err := m.manager.Save(); err != nil {
		return fmt.Errorf("failed to save index after GC: %w", err)
	}

	m.log("Automatic GC completed: removed %d chunks", removedChunks)
	return nil
}

// shouldAlert determines if an alert should be sent
func (m *Monitor) shouldAlert(unreferencedCount int) bool {
	return m.config.AlertWebhook != "" && unreferencedCount >= m.config.AlertThreshold
}

// sendAlert sends an alert via webhook
func (m *Monitor) sendAlert(stats deduplication.DeduplicationStats) {
	m.log("Alert: High number of unreferenced chunks detected: %d", stats.UnreferencedChunks)

	if m.config.AlertWebhook != "" {
		// TODO: Implement HTTP POST to webhook URL
		// For now, just log the webhook URL
		m.log("Would send webhook alert to: %s", m.config.AlertWebhook)
	}
}

// setupLogging initializes the logger with file output
func (m *Monitor) setupLogging(logFile string) {
	// Ensure log directory exists
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, constants.StandardDirPerms); err != nil {
		fmt.Printf("Warning: Failed to create log directory %s: %v\n", logDir, err)
		m.logger = log.Default()
		return
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, constants.StandardFilePerms)
	if err != nil {
		fmt.Printf("Warning: Failed to open log file %s: %v\n", logFile, err)
		m.logger = log.Default()
		return
	}

	m.logger = log.New(file, "[GC Monitor] ", log.LstdFlags)
}

// log logs a message if logging is enabled
func (m *Monitor) log(format string, args ...interface{}) {
	if m.config.EnableLogging && m.logger != nil {
		m.logger.Printf(format, args...)
	}
}

// GetStats returns current deduplication statistics
func (m *Monitor) GetStats() deduplication.DeduplicationStats {
	return m.manager.GetStats()
}

// UpdateConfig updates the monitor configuration
func (m *Monitor) UpdateConfig(config MonitorConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config = config
}
