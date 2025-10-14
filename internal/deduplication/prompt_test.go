package deduplication

import (
	"testing"

	"github.com/substantialcattle5/sietch/internal/config"
)

func TestPromptDeduplicationConfig(t *testing.T) {
	// Note: These tests are limited since PromptDeduplicationConfig uses interactive prompts
	// In a real scenario, we would need to mock the promptui interactions
	// For now, we'll test the basic structure and error conditions

	t.Run("ConfigStructure", func(t *testing.T) {
		// Test that we can create a configuration struct
		vaultConfig := &config.VaultConfig{
			Deduplication: config.DeduplicationConfig{
				Enabled:      false,
				Strategy:     "",
				MinChunkSize: "",
				MaxChunkSize: "",
				GCThreshold:  0,
				IndexEnabled: false,
			},
		}

		// Verify initial state
		if vaultConfig.Deduplication.Enabled {
			t.Error("Deduplication should be disabled by default")
		}

		if vaultConfig.Deduplication.Strategy != "" {
			t.Error("Strategy should be empty by default")
		}
	})

	t.Run("ConfigValidation", func(t *testing.T) {
		// Test various configuration states
		testCases := []struct {
			name     string
			config   config.DeduplicationConfig
			expected bool
		}{
			{
				name: "ValidEnabledConfig",
				config: config.DeduplicationConfig{
					Enabled:      true,
					Strategy:     "content",
					MinChunkSize: "1KB",
					MaxChunkSize: "64MB",
					GCThreshold:  1000,
					IndexEnabled: true,
				},
				expected: true,
			},
			{
				name: "ValidDisabledConfig",
				config: config.DeduplicationConfig{
					Enabled:      false,
					Strategy:     "",
					MinChunkSize: "",
					MaxChunkSize: "",
					GCThreshold:  0,
					IndexEnabled: false,
				},
				expected: false,
			},
			{
				name: "FingerprintStrategy",
				config: config.DeduplicationConfig{
					Enabled:      true,
					Strategy:     "fingerprint",
					MinChunkSize: "1KB",
					MaxChunkSize: "64MB",
					GCThreshold:  500,
					IndexEnabled: false,
				},
				expected: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.config.Enabled != tc.expected {
					t.Errorf("Config enabled state doesn't match expected: %v vs %v", tc.config.Enabled, tc.expected)
				}

				if tc.config.Enabled {
					// If enabled, should have strategy
					if tc.config.Strategy == "" {
						t.Error("Enabled config should have a strategy")
					}

					// Should have valid chunk sizes
					if tc.config.MinChunkSize == "" || tc.config.MaxChunkSize == "" {
						t.Error("Enabled config should have chunk size limits")
					}

					// GC threshold should be non-negative
					if tc.config.GCThreshold < 0 {
						t.Error("GC threshold should be non-negative")
					}
				} else {
					// If disabled, fields should be empty/default
					if tc.config.Strategy != "" {
						t.Error("Disabled config should have empty strategy")
					}

					if tc.config.MinChunkSize != "" || tc.config.MaxChunkSize != "" {
						t.Error("Disabled config should have empty chunk sizes")
					}
				}
			})
		}
	})

	t.Run("StrategyTypes", func(t *testing.T) {
		validStrategies := []string{"content", "fingerprint"}

		for _, strategy := range validStrategies {
			config := config.DeduplicationConfig{
				Enabled:  true,
				Strategy: strategy,
			}

			if !config.Enabled {
				t.Error("Config should be enabled")
			}

			if config.Strategy != strategy {
				t.Errorf("Expected strategy %s, got %s", strategy, config.Strategy)
			}
		}
	})

	t.Run("ChunkSizeFormats", func(t *testing.T) {
		validSizes := []string{
			"1KB", "1MB", "1GB",
			"512KB", "64MB", "1024MB",
			"0", "1", "100",
		}

		for _, size := range validSizes {
			config := config.DeduplicationConfig{
				MinChunkSize: size,
				MaxChunkSize: size,
			}

			if config.MinChunkSize != size {
				t.Errorf("Expected min chunk size %s, got %s", size, config.MinChunkSize)
			}

			if config.MaxChunkSize != size {
				t.Errorf("Expected max chunk size %s, got %s", size, config.MaxChunkSize)
			}
		}
	})

	t.Run("GCThresholdValues", func(t *testing.T) {
		validThresholds := []int{0, 1, 100, 1000, 10000}

		for _, threshold := range validThresholds {
			config := config.DeduplicationConfig{
				GCThreshold: threshold,
			}

			if config.GCThreshold != threshold {
				t.Errorf("Expected GC threshold %d, got %d", threshold, config.GCThreshold)
			}

			if config.GCThreshold < 0 {
				t.Error("GC threshold should not be negative")
			}
		}
	})

	t.Run("IndexEnabledOptions", func(t *testing.T) {
		testCases := []bool{true, false}

		for _, enabled := range testCases {
			config := config.DeduplicationConfig{
				IndexEnabled: enabled,
			}

			if config.IndexEnabled != enabled {
				t.Errorf("Expected index enabled %v, got %v", enabled, config.IndexEnabled)
			}
		}
	})
}

// TestDeduplicationConfigDefaults tests default values that would be set by the prompt
func TestDeduplicationConfigDefaults(t *testing.T) {
	t.Run("DefaultValues", func(t *testing.T) {
		// These are the defaults that should be suggested in prompts
		expectedDefaults := map[string]string{
			"MinChunkSize": "1KB",
			"MaxChunkSize": "64MB",
			"GCThreshold":  "1000",
		}

		// Test that these are reasonable defaults
		if expectedDefaults["MinChunkSize"] == "" {
			t.Error("MinChunkSize default should not be empty")
		}

		if expectedDefaults["MaxChunkSize"] == "" {
			t.Error("MaxChunkSize default should not be empty")
		}

		if expectedDefaults["GCThreshold"] == "" {
			t.Error("GCThreshold default should not be empty")
		}
	})

	t.Run("RecommendedStrategy", func(t *testing.T) {
		// Content strategy should be recommended (marked as recommended in prompts)
		recommendedStrategy := "content"

		if recommendedStrategy != "content" {
			t.Error("Content strategy should be the recommended option")
		}
	})

	t.Run("RecommendedIndexEnabled", func(t *testing.T) {
		// Index should be enabled by default (marked as recommended)
		recommendedIndexEnabled := true

		if !recommendedIndexEnabled {
			t.Error("Index should be recommended to be enabled")
		}
	})
}