package scanner

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type ScanResult struct {
	Binaries map[string]bool
	Fonts    map[string]bool
}

type Scanner struct {
	VisitedFiles map[string]bool
	Variables    map[string]string
	Binaries     map[string]bool
	Fonts        map[string]bool
}

func NewScanner() *Scanner {
	return &Scanner{
		VisitedFiles: make(map[string]bool),
		Variables:    make(map[string]string),
		Binaries:     make(map[string]bool),
		Fonts:        make(map[string]bool),
	}
}

func (s *Scanner) Scan(configPath string) error {
	absPath, err := filepath.Abs(ExpandPath(configPath))
	if err != nil {
		return err
	}

	if s.VisitedFiles[absPath] {
		return nil
	}
	s.VisitedFiles[absPath] = true

	file, err := os.Open(absPath)
	if err != nil {
		return nil // Skip unreadable files
	}
	defer file.Close()

	varRegex := regexp.MustCompile(`^\s*(\$[a-zA-Z0-9_-]+)\s*=\s*(.+)$`)
	sourceRegex := regexp.MustCompile(`^\s*source\s*=\s*([^\s#]+)`)
	execRegex := regexp.MustCompile(`^\s*exec(?:-once)?\s*=\s*(?:\[[^\]]+\]\s*)?([^\s&;]+)`)
	bindRegex := regexp.MustCompile(`^\s*bind[ems]?\s*=\s*[^,]+,\s*[^,]+,\s*exec\s*,\s*([^\s,]+)`)
	
	// Font patterns
	fontRegex := regexp.MustCompile(`(?i)^\s*(?:font|font-family|font_family)\s*=\s*(.+)$`)

	currentDir := filepath.Dir(absPath)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 1. Variable Detection
		if match := varRegex.FindStringSubmatch(line); len(match) > 1 {
			varName := match[1]
			varValue := strings.Trim(match[2], "\"' ")
			
			for k, v := range s.Variables {
				varValue = strings.ReplaceAll(varValue, k, v)
			}
			
			s.Variables[varName] = varValue
			
			// If the variable name itself suggests it's a font, we might want to track its value
			if strings.Contains(strings.ToLower(varName), "font") {
				s.Fonts[cleanFontName(varValue)] = true
			}
			continue
		}

		// 2. Variable Resolution
		for i := 0; i < 5; i++ {
			oldLine := line
			for vName, vValue := range s.Variables {
				if strings.Contains(line, vName) {
					line = strings.ReplaceAll(line, vName, vValue)
				}
			}
			if line == oldLine {
				break
			}
		}

		// 3. Source command
		if match := sourceRegex.FindStringSubmatch(line); len(match) > 1 {
			targetPath := match[1]
			if !strings.HasPrefix(targetPath, "~") && !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(currentDir, targetPath)
			}
			s.Scan(targetPath)
			continue
		}

		// 4. Binary commands
		var cmd string
		if match := execRegex.FindStringSubmatch(line); len(match) > 1 {
			cmd = match[1]
		} else if match := bindRegex.FindStringSubmatch(line); len(match) > 1 {
			cmd = match[1]
		}

		cmd = strings.Trim(cmd, "\"'")
		if cmd != "" && !strings.HasPrefix(cmd, "$") {
			s.Binaries[cmd] = true
		}

		// 5. Font detection
		if match := fontRegex.FindStringSubmatch(line); len(match) > 1 {
			fontName := cleanFontName(match[1])
			if fontName != "" {
				s.Fonts[fontName] = true
			}
		}
	}

	return scanner.Err()
}

func cleanFontName(name string) string {
	name = strings.Split(name, "#")[0] // Remove comments
	name = strings.Trim(name, "\"' ")
	// Handle complex font strings like "JetBrainsMono Nerd Font 12" or "family: JetBrainsMono"
	// For simplicity, we just take the part before any space + number (size)
	reSize := regexp.MustCompile(`\s+\d+(\.\d+)?$`)
	name = reSize.ReplaceAllString(name, "")
	
	// Handle Rofi style "JetBrainsMono Nerd Font 12" -> "JetBrainsMono Nerd Font"
	// Handle CSS style "JetBrainsMono Nerd Font, sans-serif"
	name = strings.Split(name, ",")[0]
	name = strings.TrimSpace(name)
	
	return name
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func VerifyFont(fontFamily string) bool {
	// 1. Try fc-list
	cmd := exec.Command("fc-list", ":", "family")
	output, err := cmd.Output()
	if err == nil {
		target := simplifyFontName(fontFamily)
		if target == "" {
			return false
		}

		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			// fc-list output is comma separated families
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

				// Check for exact simplified match or substring match
				if simF == target || strings.Contains(simF, target) || strings.Contains(target, simF) {
					return true
				}
			}
		}
	}

	return false
}

func simplifyFontName(name string) string {
	name = strings.ToLower(name)
	// Remove common suffixes/parts that vary
	name = strings.ReplaceAll(name, "nerd font", "")
	name = strings.ReplaceAll(name, "nf", "")
	// Remove separators
	name = strings.ReplaceAll(name, " ", "")
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return strings.TrimSpace(name)
}

func CheckFontConfig() bool {
	_, err := exec.LookPath("fc-list")
	return err == nil
}
