//go:build linux

package scanner

import (
	"os/exec"
	"strings"
)

func VerifyFont(fontFamily string) bool {
	// Try fc-list
	cmd := exec.Command("fc-list", ":", "family")
	output, err := cmd.Output()
	if err == nil {
		target := simplifyFontName(fontFamily)
		if target == "" {
			return false
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			families := strings.Split(line, ",")
			for _, f := range families {
				f = strings.TrimSpace(f)
				if f == "" {
					continue
				}

				simF := simplifyFontName(f)
				if simF == "" {
					continue
				}

				if simF == target || strings.Contains(simF, target) || strings.Contains(target, simF) {
					return true
				}
			}
		}
	}

	return false
}

func CheckFontConfig() bool {
	_, err := exec.LookPath("fc-list")
	return err == nil
}
