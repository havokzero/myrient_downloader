// internal/util/unzip.go
package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnzipZipFileInPlace extracts zipPath into the directory where the zip lives
// (e.g. /roms/SNES/Game.zip -> /roms/SNES/<zip contents>).
// It returns that directory path and deletes the .zip afterwards.
func UnzipZipFileInPlace(zipPath string) (string, error) {
	dir := filepath.Dir(zipPath)

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Avoid zip slip
		cleanName := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanName, "..") {
			continue
		}

		outPath := filepath.Join(dir, cleanName)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(outPath, f.Mode()); err != nil {
				return "", fmt.Errorf("mkdir %s: %w", outPath, err)
			}
			continue
		}

		// Ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
			return "", fmt.Errorf("mkdir %s: %w", filepath.Dir(outPath), err)
		}

		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("open entry %s: %w", f.Name, err)
		}

		if err := func() error {
			defer rc.Close()
			outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
			if err != nil {
				return fmt.Errorf("create %s: %w", outPath, err)
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, rc)
			return err
		}(); err != nil {
			return "", err
		}
	}

	// Delete the original .zip to save space
	if err := os.Remove(zipPath); err != nil {
		// Not fatal for extraction, but report it
		return dir, fmt.Errorf("remove zip %s: %w", zipPath, err)
	}

	return dir, nil
}
