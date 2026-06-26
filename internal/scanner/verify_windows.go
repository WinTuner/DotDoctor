//go:build windows

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

	sysRoot := os.Getenv("SystemRoot")
	if sysRoot == "" {
		sysRoot = `C:\Windows`
	}
	fontDir := filepath.Join(sysRoot, "Fonts")

	if _, err := os.Stat(fontDir); os.IsNotExist(err) {
		return false
	}

	found := false
	_ = filepath.WalkDir(fontDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}

		name := filepath.Base(path)
		ext := filepath.Ext(name)
		nameNoExt := strings.TrimSuffix(name, ext)

		simName := simplifyFontName(nameNoExt)
		if simName == target || strings.Contains(simName, target) || strings.Contains(target, simName) {
			found = true
			return filepath.SkipDir
		}
		return nil
	})

	return found
}

func CheckFontConfig() bool {
	return true
}
