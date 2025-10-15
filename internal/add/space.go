/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package add

import (
	"github.com/substantialcattle5/sietch/internal/config"
)

// SpaceSavings represents space savings statistics for a file
type SpaceSavings struct {
	OriginalSize   int64
	CompressedSize int64
	SpaceSaved     int64
	SpaceSavedPct  float64
}

// CalculateSpaceSavings calculates space savings for a file based on its chunks
func CalculateSpaceSavings(chunks []config.ChunkRef) SpaceSavings {
	originalSize := int64(0)
	compressedSize := int64(0)

	for _, chunk := range chunks {
		originalSize += chunk.Size
		if chunk.CompressedSize > 0 {
			compressedSize += chunk.CompressedSize
		} else {
			// If no compressed size is recorded, use original size
			compressedSize += chunk.Size
		}
	}

	spaceSaved := originalSize - compressedSize
	var spaceSavedPct float64
	if originalSize > 0 {
		spaceSavedPct = float64(spaceSaved) / float64(originalSize) * 100
	}

	return SpaceSavings{
		OriginalSize:   originalSize,
		CompressedSize: compressedSize,
		SpaceSaved:     spaceSaved,
		SpaceSavedPct:  spaceSavedPct,
	}
}

// CalculateTotalSpaceSavings calculates total space savings from multiple files
func CalculateTotalSpaceSavings(results []ProcessResult) SpaceSavings {
	total := SpaceSavings{}

	for _, result := range results {
		if result.Success {
			total.OriginalSize += result.SpaceSavings.OriginalSize
			total.CompressedSize += result.SpaceSavings.CompressedSize
			total.SpaceSaved += result.SpaceSavings.SpaceSaved
		}
	}

	if total.OriginalSize > 0 {
		total.SpaceSavedPct = float64(total.SpaceSaved) / float64(total.OriginalSize) * 100
	}

	return total
}
