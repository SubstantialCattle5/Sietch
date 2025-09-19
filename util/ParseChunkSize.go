package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func ParseChunkSize(chunkSize string) (int64, error) {
	if chunkSize == "" {
		return 0, fmt.Errorf("size cannot be empty")
	}

	// Regular expression to parse size with optional unit (including negative numbers)
	re := regexp.MustCompile(`^(-?\d+(?:\.\d+)?)\s*([a-zA-Z]*)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(chunkSize))

	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %s", chunkSize)
	}

	// Parse the numeric part
	sizeFloat, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", matches[1])
	}

	// Parse the unit part (case-insensitive)
	unit := strings.ToUpper(strings.TrimSpace(matches[2]))

	var multiplier int64
	switch unit {
	case "", "B", "BYTES":
		multiplier = 1
	case "K", "KB", "KILOBYTES":
		multiplier = 1024
	case "M", "MB", "MEGABYTES":
		multiplier = 1024 * 1024
	case "G", "GB", "GIGABYTES":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB", "TERABYTES":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unit)
	}

	// Calculate final size in bytes
	result := int64(sizeFloat * float64(multiplier))

	if result < 0 {
		return 0, fmt.Errorf("size cannot be negative")
	}

	return result, nil
}
