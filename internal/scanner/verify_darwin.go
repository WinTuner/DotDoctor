//go:build darwin

package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func VerifyFont(fontFamily string) bool {
	target := simplifyFontName(fontFamily)
	if target == "" {
		return false
	}

	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, "Library/Fonts"),
		"/Library/Fonts",
		"/System/Library/Fonts",
	}

	found := false
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // continue
			}
			if d.IsDir() {
				return nil
			}

			// Get the file name without directories
			name := filepath.Base(path)
			
			// Remove extension (e.g. ".ttf")
			ext := filepath.Ext(name)
			nameNoExt := strings.TrimSuffix(name, ext)

			// Simplify name (e.g. "JetBrainsMonoNerdFont-Regular" -> "jetbrainsmono")
			simName := simplifyFontName(nameNoExt)
			
			if simName == target || strings.Contains(simName, target) || strings.Contains(target, simName) {
				found = true
				return filepath.SkipDir // Stop walking this directory
			}
			return nil
		})

		if found {
			return true
		}
	}

	return false
}

func CheckFontConfig() bool {
	return true
}
