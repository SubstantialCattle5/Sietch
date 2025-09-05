package validation

import (
	"fmt"
	"strings"
)

func ValidateAndPrepareInputs(author string, tags []string, templateName string, configFile string) (string, []string, error) {
	// Single-pass author validation for efficiency
	author = validateAuthor(author)

	// Single-pass tags validation for efficiency
	tags = validateTags(tags)

	// Apply template configuration if specified
	if templateName != "" {
		fmt.Printf("Applying template: %s\n", templateName)
		// TODO Implement template functionality
	}

	// Load configuration from file if specified
	if configFile != "" {
		fmt.Printf("Loading configuration from: %s\n", configFile)
		// TODO Implement config loading functionality
	}

	return author, tags, nil
}

// validateAuthor performs all author validations in a single pass for efficiency
func validateAuthor(author string) string {
	// Handle empty author case first
	if author == "" {
		return "unknown"
	}

	// Normalize newlines and tabs to spaces
	author = strings.ReplaceAll(author, "\n", " ")
	author = strings.ReplaceAll(author, "\t", " ")

	// Trim whitespace
	author = strings.TrimSpace(author)

	// Check if author is only whitespace after normalization
	if author == "" {
		return "unknown"
	}

	// Truncate very long author names (only for extremely long names)
	if len(author) > 150 {
		author = author[:150]
	}

	return author
}

// validateTags performs all tag validations in a single pass for efficiency
func validateTags(tags []string) []string {
	// If nil tags, return empty slice (no default tags)
	if tags == nil {
		return []string{}
	}

	// If empty tags, return empty slice (no default tags)
	if len(tags) == 0 {
		return []string{}
	}

	// Process tags in a single pass
	validTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		// Normalize newlines and tabs to spaces
		tag = strings.ReplaceAll(tag, "\n", " ")
		tag = strings.ReplaceAll(tag, "\t", " ")

		// Trim whitespace
		tag = strings.TrimSpace(tag)

		// Skip empty tags after trimming
		if tag != "" {
			validTags = append(validTags, tag)
		}
	}

	return validTags
}
