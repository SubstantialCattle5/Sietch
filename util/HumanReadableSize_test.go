package util

import (
	"testing"
)

func TestHumanReadableSize(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		// Bytes
		{
			name:  "zero bytes",
			input: 0,
			want:  "0 B",
		},
		{
			name:  "single byte",
			input: 1,
			want:  "1 B",
		},
		{
			name:  "multiple bytes",
			input: 500,
			want:  "500 B",
		},
		{
			name:  "bytes at KB boundary",
			input: 1023,
			want:  "1023 B",
		},

		// Kilobytes
		{
			name:  "exactly 1 KB",
			input: 1024,
			want:  "1.0 KB",
		},
		{
			name:  "1.5 KB",
			input: 1536,
			want:  "1.5 KB",
		},
		{
			name:  "multiple KB",
			input: 5120,
			want:  "5.0 KB",
		},
		{
			name:  "KB with decimals",
			input: 1500,
			want:  "1.5 KB",
		},
		{
			name:  "KB at MB boundary",
			input: 1048575,
			want:  "1024.0 KB",
		},

		// Megabytes
		{
			name:  "exactly 1 MB",
			input: 1048576,
			want:  "1.0 MB",
		},
		{
			name:  "1.5 MB",
			input: 1572864,
			want:  "1.5 MB",
		},
		{
			name:  "multiple MB",
			input: 10485760,
			want:  "10.0 MB",
		},
		{
			name:  "MB with decimals",
			input: 2621440,
			want:  "2.5 MB",
		},
		{
			name:  "MB at GB boundary",
			input: 1073741823,
			want:  "1024.0 MB",
		},

		// Gigabytes
		{
			name:  "exactly 1 GB",
			input: 1073741824,
			want:  "1.0 GB",
		},
		{
			name:  "1.5 GB",
			input: 1610612736,
			want:  "1.5 GB",
		},
		{
			name:  "multiple GB",
			input: 10737418240,
			want:  "10.0 GB",
		},
		{
			name:  "GB with decimals",
			input: 2684354560,
			want:  "2.5 GB",
		},

		// Terabytes
		{
			name:  "exactly 1 TB",
			input: 1099511627776,
			want:  "1.0 TB",
		},
		{
			name:  "1.5 TB",
			input: 1649267441664,
			want:  "1.5 TB",
		},
		{
			name:  "multiple TB",
			input: 10995116277760,
			want:  "10.0 TB",
		},

		// Large values
		{
			name:  "very large TB",
			input: 109951162777600,
			want:  "100.0 TB",
		},
		{
			name:  "extremely large value",
			input: 1099511627776000,
			want:  "1000.0 TB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HumanReadableSize(tt.input)
			if got != tt.want {
				t.Errorf("HumanReadableSize(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHumanReadableSizeEdgeCases(t *testing.T) {
	t.Run("negative values", func(t *testing.T) {
		// The function should handle negative values gracefully
		result := HumanReadableSize(-1024)
		// The exact behavior for negative values depends on implementation
		// but it should not panic
		if result == "" {
			t.Error("HumanReadableSize(-1024) returned empty string")
		}
		t.Logf("HumanReadableSize(-1024) = %q", result)
	})

	t.Run("maximum int64", func(t *testing.T) {
		// Test with maximum int64 value
		maxInt64 := int64(9223372036854775807)
		result := HumanReadableSize(maxInt64)
		if result == "" {
			t.Error("HumanReadableSize(maxInt64) returned empty string")
		}
		t.Logf("HumanReadableSize(maxInt64) = %q", result)
	})

	t.Run("minimum int64", func(t *testing.T) {
		// Test with minimum int64 value
		minInt64 := int64(-9223372036854775808)
		result := HumanReadableSize(minInt64)
		if result == "" {
			t.Error("HumanReadableSize(minInt64) returned empty string")
		}
		t.Logf("HumanReadableSize(minInt64) = %q", result)
	})
}

func TestHumanReadableSizeRounding(t *testing.T) {
	// Test specific rounding behavior
	tests := []struct {
		name      string
		input     int64
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:  "rounding to 1 decimal place",
			input: 1536, // 1.5 KB exactly
			checkFunc: func(t *testing.T, result string) {
				if result != "1.5 KB" {
					t.Errorf("Expected '1.5 KB', got %q", result)
				}
			},
		},
		{
			name:  "rounding small fractions",
			input: 1025, // ~1.001 KB
			checkFunc: func(t *testing.T, result string) {
				// Should round to 1.0 KB or similar
				if !containsStringHelper2(result, "1.0 KB") && !containsStringHelper2(result, "1.1 KB") {
					t.Errorf("Expected around 1.0 KB, got %q", result)
				}
			},
		},
		{
			name:  "rounding larger fractions",
			input: 1587, // ~1.55 KB
			checkFunc: func(t *testing.T, result string) {
				// Should be around 1.5 or 1.6 KB
				if !containsStringHelper2(result, "1.5") && !containsStringHelper2(result, "1.6") {
					t.Errorf("Expected around 1.5-1.6 KB, got %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanReadableSize(tt.input)
			tt.checkFunc(t, result)
		})
	}
}

func TestHumanReadableSizeConsistency(t *testing.T) {
	// Test that same input always produces same output
	testInputs := []int64{
		0, 1, 1024, 1048576, 1073741824,
		500, 1500, 1572864, 2684354560,
	}

	for _, input := range testInputs {
		t.Run("consistency", func(t *testing.T) {
			results := make([]string, 10)
			for i := 0; i < 10; i++ {
				results[i] = HumanReadableSize(input)
			}

			// All results should be identical
			firstResult := results[0]
			for i, result := range results {
				if result != firstResult {
					t.Errorf("HumanReadableSize(%d) iteration %d = %q, want %q",
						input, i, result, firstResult)
				}
			}
		})
	}
}

func TestHumanReadableSizeFormat(t *testing.T) {
	// Test that the format is consistent
	formatTests := []struct {
		name      string
		input     int64
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:  "bytes format",
			input: 500,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, " B") {
					t.Errorf("Bytes result should contain ' B', got %q", result)
				}
			},
		},
		{
			name:  "KB format",
			input: 1024,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, " KB") {
					t.Errorf("KB result should contain ' KB', got %q", result)
				}
				if !containsStringHelper2(result, ".") {
					t.Errorf("KB result should contain decimal point, got %q", result)
				}
			},
		},
		{
			name:  "MB format",
			input: 1048576,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, " MB") {
					t.Errorf("MB result should contain ' MB', got %q", result)
				}
				if !containsStringHelper2(result, ".") {
					t.Errorf("MB result should contain decimal point, got %q", result)
				}
			},
		},
		{
			name:  "GB format",
			input: 1073741824,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, " GB") {
					t.Errorf("GB result should contain ' GB', got %q", result)
				}
				if !containsStringHelper2(result, ".") {
					t.Errorf("GB result should contain decimal point, got %q", result)
				}
			},
		},
		{
			name:  "TB format",
			input: 1099511627776,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, " TB") {
					t.Errorf("TB result should contain ' TB', got %q", result)
				}
				if !containsStringHelper2(result, ".") {
					t.Errorf("TB result should contain decimal point, got %q", result)
				}
			},
		},
	}

	for _, tt := range formatTests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanReadableSize(tt.input)
			tt.checkFunc(t, result)
		})
	}
}

func TestHumanReadableSizeProgression(t *testing.T) {
	// Test that progression through units works correctly
	progressionTests := []struct {
		name         string
		inputs       []int64
		expectedUnit string
	}{
		{
			name:         "bytes progression",
			inputs:       []int64{1, 10, 100, 1000},
			expectedUnit: "B",
		},
		{
			name:         "KB progression",
			inputs:       []int64{1024, 2048, 10240, 102400},
			expectedUnit: "KB",
		},
		{
			name:         "MB progression",
			inputs:       []int64{1048576, 2097152, 10485760, 104857600},
			expectedUnit: "MB",
		},
		{
			name:         "GB progression",
			inputs:       []int64{1073741824, 2147483648, 10737418240},
			expectedUnit: "GB",
		},
	}

	for _, tt := range progressionTests {
		t.Run(tt.name, func(t *testing.T) {
			for _, input := range tt.inputs {
				result := HumanReadableSize(input)
				if !containsStringHelper2(result, tt.expectedUnit) {
					t.Errorf("HumanReadableSize(%d) = %q, should contain %q",
						input, result, tt.expectedUnit)
				}
			}
		})
	}
}

func TestHumanReadableSizeBoundaryValues(t *testing.T) {
	// Test values right at the boundaries between units
	boundaryTests := []struct {
		name      string
		input     int64
		checkFunc func(t *testing.T, result string)
	}{
		{
			name:  "just under 1KB",
			input: 1023,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, "B") || containsStringHelper2(result, "KB") {
					t.Errorf("Expected bytes unit for 1023, got %q", result)
				}
			},
		},
		{
			name:  "exactly 1KB",
			input: 1024,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, "KB") {
					t.Errorf("Expected KB unit for 1024, got %q", result)
				}
			},
		},
		{
			name:  "just under 1MB",
			input: 1048575,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, "KB") || containsStringHelper2(result, "MB") {
					t.Errorf("Expected KB unit for 1048575, got %q", result)
				}
			},
		},
		{
			name:  "exactly 1MB",
			input: 1048576,
			checkFunc: func(t *testing.T, result string) {
				if !containsStringHelper2(result, "MB") {
					t.Errorf("Expected MB unit for 1048576, got %q", result)
				}
			},
		},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			result := HumanReadableSize(tt.input)
			tt.checkFunc(t, result)
		})
	}
}

// Helper function
func containsStringHelper2(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstringHelper2(s, substr))))
}

func containsSubstringHelper2(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkHumanReadableSize(b *testing.B) {
	testCases := []int64{
		0, 1, 1024, 1048576, 1073741824, 1099511627776,
		500, 1500, 1572864, 2684354560,
	}

	for _, testCase := range testCases {
		b.Run("", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				HumanReadableSize(testCase)
			}
		})
	}
}

func BenchmarkHumanReadableSizeLarge(b *testing.B) {
	largeValues := []int64{
		1099511627776000,    // 1000 TB
		109951162777600000,  // 100,000 TB
		1099511627776000000, // 1,000,000 TB
	}

	for _, testCase := range largeValues {
		b.Run("large", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				HumanReadableSize(testCase)
			}
		})
	}
}
