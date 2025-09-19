package util

import (
	"testing"
)

func TestParseChunkSize(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        int64
		wantErr     bool
		errContains string
	}{
		// Valid cases - new implementation properly handles units
		{
			name:    "single digit (bytes)",
			input:   "1",
			want:    1, // 1 byte
			wantErr: false,
		},
		{
			name:    "multiple digits (bytes)",
			input:   "4",
			want:    4, // 4 bytes
			wantErr: false,
		},
		{
			name:    "zero",
			input:   "0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "larger number (bytes)",
			input:   "100",
			want:    100, // 100 bytes
			wantErr: false,
		},
		{
			name:    "kilobytes",
			input:   "1KB",
			want:    1024, // 1 * 1024
			wantErr: false,
		},
		{
			name:    "megabytes",
			input:   "1MB",
			want:    1048576, // 1 * 1024 * 1024
			wantErr: false,
		},
		{
			name:    "gigabytes",
			input:   "1GB",
			want:    1073741824, // 1 * 1024 * 1024 * 1024
			wantErr: false,
		},
		{
			name:    "decimal number",
			input:   "1.5KB",
			want:    1536, // 1.5 * 1024
			wantErr: false,
		},

		// Error cases
		{
			name:        "empty string",
			input:       "",
			want:        0,
			wantErr:     true,
			errContains: "size cannot be empty",
		},
		{
			name:        "non-numeric",
			input:       "abc",
			want:        0,
			wantErr:     true,
			errContains: "invalid size format",
		},
		{
			name:        "negative number",
			input:       "-1",
			want:        0,
			wantErr:     true,
			errContains: "size cannot be negative",
		},
		{
			name:        "unsupported unit",
			input:       "1XB",
			want:        0,
			wantErr:     true,
			errContains: "unsupported unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChunkSize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseChunkSize(%q) expected error but got none", tt.input)
					return
				}
				if tt.errContains != "" && !containsStringHelper(err.Error(), tt.errContains) {
					t.Errorf("ParseChunkSize(%q) error = %q, want it to contain %q", tt.input, err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseChunkSize(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got != tt.want {
				t.Errorf("ParseChunkSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseChunkSizeEdgeCases(t *testing.T) {
	t.Run("very large number", func(t *testing.T) {
		result, err := ParseChunkSize("999999")
		if err != nil {
			t.Errorf("ParseChunkSize() with large number failed: %v", err)
			return
		}
		expected := int64(999999) // Just the number itself (bytes)
		if result != expected {
			t.Errorf("ParseChunkSize('999999') = %d, want %d", result, expected)
		}
	})

	t.Run("leading zeros", func(t *testing.T) {
		result, err := ParseChunkSize("001")
		if err != nil {
			t.Errorf("ParseChunkSize() with leading zeros failed: %v", err)
			return
		}
		expected := int64(1) // Just 1 byte
		if result != expected {
			t.Errorf("ParseChunkSize('001') = %d, want %d", result, expected)
		}
	})

	t.Run("whitespace", func(t *testing.T) {
		// Current implementation handles whitespace fine
		result, err := ParseChunkSize(" 1 ")
		if err != nil {
			t.Errorf("ParseChunkSize() with whitespace failed: %v", err)
			return
		}
		expected := int64(1) // Just 1 byte
		if result != expected {
			t.Errorf("ParseChunkSize(' 1 ') = %d, want %d", result, expected)
		}
	})

	t.Run("with KB unit", func(t *testing.T) {
		result, err := ParseChunkSize("4KB")
		if err != nil {
			t.Errorf("ParseChunkSize() with KB unit failed: %v", err)
			return
		}
		expected := int64(4 * 1024) // 4 KB
		if result != expected {
			t.Errorf("ParseChunkSize('4KB') = %d, want %d", result, expected)
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		result, err := ParseChunkSize("1kb")
		if err != nil {
			t.Errorf("ParseChunkSize() with lowercase unit failed: %v", err)
			return
		}
		expected := int64(1024) // 1 KB
		if result != expected {
			t.Errorf("ParseChunkSize('1kb') = %d, want %d", result, expected)
		}
	})
}

func TestParseChunkSizeConsistency(t *testing.T) {
	// Test that same input always produces same output
	testCases := []string{
		"1",
		"4",
		"100",
		"0",
	}

	for _, testCase := range testCases {
		t.Run("consistency_"+testCase, func(t *testing.T) {
			results := make([]int64, 10)
			for i := 0; i < 10; i++ {
				result, err := ParseChunkSize(testCase)
				if err != nil {
					t.Fatalf("ParseChunkSize(%q) iteration %d failed: %v", testCase, i, err)
				}
				results[i] = result
			}

			// All results should be identical
			firstResult := results[0]
			for i, result := range results {
				if result != firstResult {
					t.Errorf("ParseChunkSize(%q) iteration %d = %d, want %d", testCase, i, result, firstResult)
				}
			}
		})
	}
}

func TestParseChunkSizeBoundaries(t *testing.T) {
	// Test boundary values for the new implementation
	boundaryTests := []struct {
		name      string
		input     string
		checkFunc func(t *testing.T, result int64, err error)
	}{
		{
			name:  "minimum value",
			input: "1",
			checkFunc: func(t *testing.T, result int64, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				expected := int64(1) // 1 byte
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
		{
			name:  "zero value",
			input: "0",
			checkFunc: func(t *testing.T, result int64, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != 0 {
					t.Errorf("Expected 0, got %d", result)
				}
			},
		},
		{
			name:  "1KB boundary",
			input: "1KB",
			checkFunc: func(t *testing.T, result int64, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				expected := int64(1024) // 1 KB
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
		{
			name:  "1MB boundary",
			input: "1MB",
			checkFunc: func(t *testing.T, result int64, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				expected := int64(1024 * 1024) // 1 MB
				if result != expected {
					t.Errorf("Expected %d, got %d", expected, result)
				}
			},
		},
	}

	for _, tt := range boundaryTests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseChunkSize(tt.input)
			tt.checkFunc(t, result, err)
		})
	}
}

// Helper function to check if a string contains another string
func containsStringHelper(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstringHelper(s, substr))))
}

func containsSubstringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test comprehensive unit support
func TestParseChunkSizeUnits(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		// Bytes
		{"1B", 1},
		{"1BYTES", 1},
		{"100", 100}, // no unit defaults to bytes

		// Kilobytes
		{"1K", 1024},
		{"1KB", 1024},
		{"1KILOBYTES", 1024},
		{"2KB", 2048},

		// Megabytes
		{"1M", 1048576},
		{"1MB", 1048576},
		{"1MEGABYTES", 1048576},
		{"2MB", 2097152},

		// Gigabytes
		{"1G", 1073741824},
		{"1GB", 1073741824},
		{"1GIGABYTES", 1073741824},

		// Terabytes
		{"1T", 1099511627776},
		{"1TB", 1099511627776},
		{"1TERABYTES", 1099511627776},

		// Decimal values
		{"0.5KB", 512},
		{"1.5MB", 1572864},
		{"2.5GB", 2684354560},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := ParseChunkSize(test.input)
			if err != nil {
				t.Errorf("ParseChunkSize(%q) unexpected error: %v", test.input, err)
				return
			}
			if result != test.expected {
				t.Errorf("ParseChunkSize(%q) = %d, want %d", test.input, result, test.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkParseChunkSize(b *testing.B) {
	testCases := []string{
		"1",
		"4KB",
		"100MB",
		"1GB",
	}

	for _, testCase := range testCases {
		b.Run(testCase, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := ParseChunkSize(testCase)
				if err != nil {
					b.Fatalf("ParseChunkSize(%q) failed: %v", testCase, err)
				}
			}
		})
	}
}
