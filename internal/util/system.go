package util

import (
	"net/url"
	"strings"
)

// SanitizeFolderName makes a string safe to use as a folder name on most OSes.
// It removes control chars and replaces common bad chars with '-'.
func SanitizeFolderName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Unknown"
	}

	badChars := `<>:"/\|?*`
	return strings.Map(func(r rune) rune {
		// Drop control characters
		if r < 32 {
			return -1
		}
		if strings.ContainsRune(badChars, r) {
			return '-'
		}
		return r
	}, name)
}

// GuessSystemFromURL tries to derive a "system/console" name from
// the file URL relative to a root URL.
//
// Strategy:
//   - If file path starts with root path, take the first path segment
//     under the root as the system name.
//   - If that fails, fall back to the parent directory name.
func GuessSystemFromURL(rootStr, fileStr string) string {
	root, err1 := url.Parse(rootStr)
	fileURL, err2 := url.Parse(fileStr)
	if err1 != nil || err2 != nil {
		return "Unknown"
	}

	rootPath := root.Path
	filePath := fileURL.Path

	// Normalize
	rootPath = strings.TrimRight(rootPath, "/")
	filePath = strings.TrimRight(filePath, "/")

	// If file path is under root path, use the first segment after root.
	if strings.HasPrefix(filePath, rootPath) {
		rel := strings.TrimPrefix(filePath, rootPath)
		rel = strings.Trim(rel, "/")
		if rel != "" {
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" {
				return SanitizeFolderName(parts[0])
			}
		}
	}

	// Fallback: use the parent directory of the file path.
	parts := strings.Split(strings.Trim(filePath, "/"), "/")
	if len(parts) >= 2 {
		system := parts[len(parts)-2]
		return SanitizeFolderName(system)
	}

	return "Unknown"
}
