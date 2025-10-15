/*
Copyright © 2025 SubstantialCattle5, nilaysharan.com
*/
package add

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/substantialcattle5/sietch/internal/fs"
)

// ParseFileArguments parses command line arguments into source-destination pairs
// Supports two patterns:
// 1. Paired: source1 dest1 source2 dest2 ... (even number of args)
// 2. Single destination: source1 source2 ... dest (odd number of args, last is dest)
func ParseFileArguments(args []string) ([]FilePair, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("must provide at least one source and one destination")
	}

	// Check if even number of arguments (paired pattern)
	if len(args)%2 == 0 {
		// Paired pattern: source1 dest1 source2 dest2 ...
		var pairs []FilePair
		for i := 0; i < len(args); i += 2 {
			pairs = append(pairs, FilePair{
				Source:      args[i],
				Destination: args[i+1],
			})
		}
		return pairs, nil
	}

	// Odd number of arguments (single destination pattern)
	// Last argument is the destination for all sources
	destination := args[len(args)-1]
	var pairs []FilePair

	for i := 0; i < len(args)-1; i++ {
		pairs = append(pairs, FilePair{
			Source:      args[i],
			Destination: destination,
		})
	}

	return pairs, nil
}

// ExpandDirectories expands directories into file pairs if recursive flag is set
func ExpandDirectories(pairs []FilePair, recursive bool, includeHidden bool) ([]FilePair, error) {
	var expandedPairs []FilePair

	for _, pair := range pairs {
		// Get path info to determine type
		fileInfo, pathType, err := fs.GetPathInfo(pair.Source)
		if err != nil {
			return nil, err
		}

		switch pathType {
		case fs.PathTypeFile:
			// Regular file - add as is
			expandedPairs = append(expandedPairs, pair)

		case fs.PathTypeSymlink:
			// Symlink - will be handled in processing loop, add as is
			expandedPairs = append(expandedPairs, pair)

		case fs.PathTypeDir:
			// Directory - expand if recursive, otherwise error
			if !recursive {
				return nil, fmt.Errorf("'%s' is a directory. Use --recursive flag to add directories", pair.Source)
			}

			// Walk the directory tree
			err := filepath.WalkDir(pair.Source, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}

				// Skip hidden files/directories if includeHidden is false
				if fs.ShouldSkipHidden(d.Name(), includeHidden) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}

				// Only add regular files and symlinks
				if !d.IsDir() {
					// Compute relative path from source directory
					relPath, err := filepath.Rel(pair.Source, path)
					if err != nil {
						return fmt.Errorf("failed to compute relative path: %v", err)
					}

					// Preserve directory structure in destination
					destPath := filepath.Join(pair.Destination, relPath)

					expandedPairs = append(expandedPairs, FilePair{
						Source:      path,
						Destination: destPath,
					})
				}

				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("error walking directory '%s': %v", pair.Source, err)
			}

		default:
			return nil, fmt.Errorf("'%s' is not a regular file, directory, or symlink", pair.Source)
		}

		_ = fileInfo // fileInfo might be used for verbose output later
	}

	return expandedPairs, nil
}
