package gpgkey

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// checks if GPG is installed and available
func IsGPGAvailable() bool {
	cmd := exec.Command("gpg", "--version")
	return cmd.Run() == nil
}

// listGPGKeys retrieves available GPG keys from the keyring
func ListGPGKeys() ([]*GPGKeyInfo, error) {
	cmd := exec.Command("gpg", "--list-keys", "--with-colons")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute gpg --list-keys: %w", err)
	}

	return ParseGPGKeyList(string(output)), nil
}

// parseGPGKeyList parses GPG key list output and extracts key information
func ParseGPGKeyList(output string) []*GPGKeyInfo {
	var keys []*GPGKeyInfo
	lines := strings.Split(output, "\n")

	var currentKey *GPGKeyInfo

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}

		recordType := fields[0]

		switch recordType {
		case "pub":
			// Public key record
			if len(fields) >= 5 {
				currentKey = &GPGKeyInfo{
					KeyID:   fields[4],
					KeyType: fields[3],
					Expired: fields[1] == "e",
				}
			}
		case "fpr":
			// Fingerprint record
			if currentKey != nil && len(fields) >= 10 {
				currentKey.Fingerprint = fields[9]
			}
		case "uid":
			// User ID record
			if currentKey != nil && len(fields) >= 10 {
				userID := fields[9]
				currentKey.UserID = userID

				// Extract email from user ID
				emailRegex := regexp.MustCompile(`<([^>]+)>`)
				matches := emailRegex.FindStringSubmatch(userID)
				if len(matches) > 1 {
					currentKey.Email = matches[1]
				}

				// Only add the key when we have complete information
				if currentKey.KeyID != "" && !currentKey.Expired {
					keys = append(keys, currentKey)
				}
				currentKey = nil
			}
		}
	}

	return keys
}
