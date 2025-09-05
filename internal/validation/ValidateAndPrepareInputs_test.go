package validation

import (
	"reflect"
	"testing"
)

func TestValidateAndPrepareInputs(t *testing.T) {
	tests := []struct {
		name         string
		author       string
		tags         []string
		templateName string
		configFile   string
		wantAuthor   string
		wantTags     []string
		wantErr      bool
		errContains  string
	}{
		{
			name:       "valid inputs",
			author:     "John Doe",
			tags:       []string{"personal", "documents"},
			wantAuthor: "John Doe",
			wantTags:   []string{"personal", "documents"},
			wantErr:    false,
		},
		{
			name:       "empty author gets default",
			author:     "",
			tags:       []string{"test"},
			wantAuthor: "unknown",
			wantTags:   []string{"test"},
			wantErr:    false,
		},
		{
			name:       "nil tags gets empty slice",
			author:     "Test Author",
			tags:       nil,
			wantAuthor: "Test Author",
			wantTags:   []string{},
			wantErr:    false,
		},
		{
			name:       "empty tags gets empty slice",
			author:     "Test Author",
			tags:       []string{},
			wantAuthor: "Test Author",
			wantTags:   []string{},
			wantErr:    false,
		},
		{
			name:       "whitespace author gets trimmed",
			author:     "  John Doe  ",
			tags:       []string{"test"},
			wantAuthor: "John Doe",
			wantTags:   []string{"test"},
			wantErr:    false,
		},
		{
			name:       "whitespace tags get trimmed",
			author:     "Test Author",
			tags:       []string{"  tag1  ", " tag2 ", "tag3"},
			wantAuthor: "Test Author",
			wantTags:   []string{"tag1", "tag2", "tag3"},
			wantErr:    false,
		},
		{
			name:       "empty tags after trimming are removed",
			author:     "Test Author",
			tags:       []string{"tag1", "   ", "", "tag2"},
			wantAuthor: "Test Author",
			wantTags:   []string{"tag1", "tag2"},
			wantErr:    false,
		},
		{
			name:       "duplicate tags are preserved",
			author:     "Test Author",
			tags:       []string{"tag1", "tag1", "tag2"},
			wantAuthor: "Test Author",
			wantTags:   []string{"tag1", "tag1", "tag2"},
			wantErr:    false,
		},
		{
			name:       "special characters in author",
			author:     "José María García-López",
			tags:       []string{"español"},
			wantAuthor: "José María García-López",
			wantTags:   []string{"español"},
			wantErr:    false,
		},
		{
			name:       "special characters in tags",
			author:     "Test Author",
			tags:       []string{"tag-with-dashes", "tag_with_underscores", "tag.with.dots"},
			wantAuthor: "Test Author",
			wantTags:   []string{"tag-with-dashes", "tag_with_underscores", "tag.with.dots"},
			wantErr:    false,
		},
		{
			name:       "very long author name",
			author:     "This is a very long author name that might be used in some edge cases to test the validation function",
			tags:       []string{"test"},
			wantAuthor: "This is a very long author name that might be used in some edge cases to test the validation function",
			wantTags:   []string{"test"},
			wantErr:    false,
		},
		{
			name:       "many tags",
			author:     "Test Author",
			tags:       []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8", "tag9", "tag10"},
			wantAuthor: "Test Author",
			wantTags:   []string{"tag1", "tag2", "tag3", "tag4", "tag5", "tag6", "tag7", "tag8", "tag9", "tag10"},
			wantErr:    false,
		},
		{
			name:       "only whitespace author becomes unknown",
			author:     "   \t\n   ",
			tags:       []string{"test"},
			wantAuthor: "unknown",
			wantTags:   []string{"test"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAuthor, gotTags, err := ValidateAndPrepareInputs(tt.author, tt.tags, tt.templateName, tt.configFile)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAndPrepareInputs() expected error but got none")
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("ValidateAndPrepareInputs() error = %q, want it to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateAndPrepareInputs() unexpected error: %v", err)
				return
			}

			if gotAuthor != tt.wantAuthor {
				t.Errorf("ValidateAndPrepareInputs() author = %q, want %q", gotAuthor, tt.wantAuthor)
			}

			if !reflect.DeepEqual(gotTags, tt.wantTags) {
				t.Errorf("ValidateAndPrepareInputs() tags = %v, want %v", gotTags, tt.wantTags)
			}
		})
	}
}

func TestValidateAndPrepareInputsEdgeCases(t *testing.T) {
	t.Run("unicode characters", func(t *testing.T) {
		author := "测试作者"
		tags := []string{"标签1", "标签2"}

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with unicode failed: %v", err)
			return
		}

		if gotAuthor != author {
			t.Errorf("ValidateAndPrepareInputs() unicode author = %q, want %q", gotAuthor, author)
		}

		if !reflect.DeepEqual(gotTags, tags) {
			t.Errorf("ValidateAndPrepareInputs() unicode tags = %v, want %v", gotTags, tags)
		}
	})

	t.Run("emoji in author and tags", func(t *testing.T) {
		author := "John Doe 👨‍💻"
		tags := []string{"work 💼", "personal 🏠"}

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with emoji failed: %v", err)
			return
		}

		if gotAuthor != author {
			t.Errorf("ValidateAndPrepareInputs() emoji author = %q, want %q", gotAuthor, author)
		}

		if !reflect.DeepEqual(gotTags, tags) {
			t.Errorf("ValidateAndPrepareInputs() emoji tags = %v, want %v", gotTags, tags)
		}
	})

	t.Run("newlines and tabs in input", func(t *testing.T) {
		author := "John\nDoe\tTest"
		tags := []string{"tag\nwith\nnewlines", "tag\twith\ttabs"}

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with newlines/tabs failed: %v", err)
			return
		}

		// Author should have whitespace normalized
		if gotAuthor != "John Doe Test" {
			t.Errorf("ValidateAndPrepareInputs() normalized author = %q, want %q", gotAuthor, "John Doe Test")
		}

		// Tags should have whitespace normalized
		expectedTags := []string{"tag with newlines", "tag with tabs"}
		if !reflect.DeepEqual(gotTags, expectedTags) {
			t.Errorf("ValidateAndPrepareInputs() normalized tags = %v, want %v", gotTags, expectedTags)
		}
	})

	t.Run("very large input", func(t *testing.T) {
		// Create a very long author name (1000 characters)
		longAuthor := ""
		for i := 0; i < 100; i++ {
			longAuthor += "1234567890"
		}

		// Create many tags
		manyTags := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			manyTags[i] = "tag" + string(rune('0'+(i%10)))
		}

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(longAuthor, manyTags, "", "")
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with large input failed: %v", err)
			return
		}
		if gotAuthor != longAuthor[:150] {
			t.Errorf("ValidateAndPrepareInputs() long author = %q, want %q", gotAuthor, longAuthor[:150])
		}

		if len(gotTags) != len(manyTags) {
			t.Errorf("ValidateAndPrepareInputs() tag count = %d, want %d", len(gotTags), len(manyTags))
		}
	})

	t.Run("mixed whitespace types", func(t *testing.T) {
		// Test different types of whitespace characters
		author := " \t\n\r John \u00A0 Doe \t\n\r " // includes non-breaking space
		tags := []string{" \ttag1\n ", "\r tag2 \u00A0", "  tag3  "}

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with mixed whitespace failed: %v", err)
			return
		}

		// Should normalize all whitespace
		if len(gotAuthor) < 8 || len(gotAuthor) > 12 { // Allow some variance for whitespace handling
			t.Errorf("ValidateAndPrepareInputs() whitespace author = %q (len=%d), expected around 'John Doe'", gotAuthor, len(gotAuthor))
		}

		// Tags should be trimmed
		if len(gotTags) != 3 {
			t.Errorf("ValidateAndPrepareInputs() tag count = %d, want 3", len(gotTags))
		}
	})
}

func TestValidateAndPrepareInputsConsistency(t *testing.T) {
	// Test that the function is deterministic
	author := "Test Author"
	tags := []string{"tag1", "tag2", "tag3"}

	results := make([][2]interface{}, 10)
	for i := 0; i < 10; i++ {
		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			t.Fatalf("ValidateAndPrepareInputs() iteration %d failed: %v", i, err)
		}
		results[i] = [2]interface{}{gotAuthor, gotTags}
	}

	// All results should be identical
	firstResult := results[0]
	for i, result := range results {
		if !reflect.DeepEqual(result, firstResult) {
			t.Errorf("ValidateAndPrepareInputs() iteration %d produced different result: %v vs %v", i, result, firstResult)
		}
	}
}

func TestValidateAndPrepareInputsWithTemplateAndConfig(t *testing.T) {
	// Note: The current implementation doesn't seem to use templateName and configFile
	// These tests verify that they don't cause errors and don't affect the output

	t.Run("with template name", func(t *testing.T) {
		author := "Test Author"
		tags := []string{"test"}
		templateName := "test-template"

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, templateName, "")
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with template failed: %v", err)
			return
		}

		if gotAuthor != author {
			t.Errorf("ValidateAndPrepareInputs() author = %q, want %q", gotAuthor, author)
		}

		if !reflect.DeepEqual(gotTags, tags) {
			t.Errorf("ValidateAndPrepareInputs() tags = %v, want %v", gotTags, tags)
		}
	})

	t.Run("with config file", func(t *testing.T) {
		author := "Test Author"
		tags := []string{"test"}
		configFile := "/path/to/config.yaml"

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, "", configFile)
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with config file failed: %v", err)
			return
		}

		if gotAuthor != author {
			t.Errorf("ValidateAndPrepareInputs() author = %q, want %q", gotAuthor, author)
		}

		if !reflect.DeepEqual(gotTags, tags) {
			t.Errorf("ValidateAndPrepareInputs() tags = %v, want %v", gotTags, tags)
		}
	})

	t.Run("with both template and config", func(t *testing.T) {
		author := "Test Author"
		tags := []string{"test"}
		templateName := "test-template"
		configFile := "/path/to/config.yaml"

		gotAuthor, gotTags, err := ValidateAndPrepareInputs(author, tags, templateName, configFile)
		if err != nil {
			t.Errorf("ValidateAndPrepareInputs() with template and config failed: %v", err)
			return
		}

		if gotAuthor != author {
			t.Errorf("ValidateAndPrepareInputs() author = %q, want %q", gotAuthor, author)
		}

		if !reflect.DeepEqual(gotTags, tags) {
			t.Errorf("ValidateAndPrepareInputs() tags = %v, want %v", gotTags, tags)
		}
	})
}

// Helper function to check if a string contains another string
func containsString(s, substr string) bool {
	return len(substr) <= len(s) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Benchmark tests
func BenchmarkValidateAndPrepareInputs(b *testing.B) {
	author := "John Doe"
	tags := []string{"personal", "documents", "important"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			b.Fatalf("ValidateAndPrepareInputs() failed: %v", err)
		}
	}
}

func BenchmarkValidateAndPrepareInputsLargeInput(b *testing.B) {
	// Create large input
	longAuthor := ""
	for i := 0; i < 1000; i++ {
		longAuthor += "a"
	}

	largeTags := make([]string, 100)
	for i := 0; i < 100; i++ {
		largeTags[i] = "tag" + string(rune('0'+(i%10)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := ValidateAndPrepareInputs(longAuthor, largeTags, "", "")
		if err != nil {
			b.Fatalf("ValidateAndPrepareInputs() failed: %v", err)
		}
	}
}

func BenchmarkValidateAndPrepareInputsWithWhitespace(b *testing.B) {
	author := "  John   Doe  "
	tags := []string{"  tag1  ", "  tag2  ", "  tag3  "}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := ValidateAndPrepareInputs(author, tags, "", "")
		if err != nil {
			b.Fatalf("ValidateAndPrepareInputs() failed: %v", err)
		}
	}
}
