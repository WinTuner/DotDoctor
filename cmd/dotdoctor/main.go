package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fatih/color"
)

var archPackageMap = map[string]string{
	"waybar":        "waybar",
	"rofi":          "rofi-wayland",
	"wofi":          "wofi",
	"swww":          "swww",
	"dunst":         "dunst",
	"mako":          "mako",
	"grim":          "grim",
	"slurp":         "slurp",
	"wpctl":         "pipewire-utils",
	"hyprpaper":     "hyprpaper",
	"kitty":         "kitty",
	"alacritty":     "alacritty",
	"foot":          "foot",
	"dolphin":       "dolphin",
	"pavucontrol":   "pavucontrol",
	"brightnessctl": "brightnessctl",
	"playerctl":     "playerctl",
	"swaylock":      "swaylock-effects",
	"swayidle":      "swayidle",
	"hyprpicker":    "hyprpicker",
	"wl-copy":       "wl-clipboard",
	"wl-paste":      "wl-clipboard",
	"hyprlock":      "hyprlock",
	"hypridle":      "hypridle",
}

func main() {
	printHeader()

	configPath := getConfigPath()

	commands := make(map[string]bool)
	visitedFiles := make(map[string]bool)
	variables := make(map[string]string) 

	color.Cyan("🔍 Initiating Iterative Variable Scan starting from: %s\n", configPath)

	err := scanFile(configPath, commands, visitedFiles, variables)
	if err != nil {
		color.Red("❌ Error scanning configuration: %v", err)
		os.Exit(1)
	}

	fmt.Println()

	if len(commands) == 0 {
		color.Yellow("⚠️ No external commands found across your configuration files.")
		os.Exit(0)
	}

	var sortedCmds []string
	for cmd := range commands {
		sortedCmds = append(sortedCmds, cmd)
	}
	sort.Strings(sortedCmds)

	foundCount := 0
	missingCount := 0

	for _, cmd := range sortedCmds {
		checkPath := expandPath(cmd)

		_, err := exec.LookPath(checkPath)
		if err == nil {
			color.Green("  ✅ [FOUND]    %-15s", cmd)
			foundCount++
		} else {
			color.Red("  ❌ [MISSING]  %-15s", cmd)
			missingCount++
			if pkg := archPackageMap[cmd]; pkg != "" {
				color.Yellow("       ➜ Suggestion: Run 'sudo pacman -S %s'", pkg)
			}
		}
	}

	printSummary(len(commands), foundCount, missingCount)
}

func scanFile(filePath string, commands map[string]bool, visited map[string]bool, variables map[string]string) error {
	absPath, err := filepath.Abs(expandPath(filePath))
	if err != nil {
		return err
	}

	if visited[absPath] {
		return nil
	}
	visited[absPath] = true

	file, err := os.Open(absPath)
	if err != nil {
		color.Yellow("  ⚠️ Warning: Could not open file: %s (Skipping)", filePath)
		return nil
	}
	defer file.Close()

	varRegex := regexp.MustCompile(`^\s*(\$[a-zA-Z0-9_-]+)\s*=\s*(.+)$`)
	sourceRegex := regexp.MustCompile(`^\s*source\s*=\s*([^\s#]+)`)
	execRegex := regexp.MustCompile(`^\s*exec(?:-once)?\s*=\s*(?:\[[^\]]+\]\s*)?([^\s&;]+)`)
	bindRegex := regexp.MustCompile(`^\s*bind[ems]?\s*=\s*[^,]+,\s*[^,]+,\s*exec\s*,\s*([^\s,]+)`)

	currentDir := filepath.Dir(absPath)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// 1. ตรวจจับและบันทึกตัวแปร ขยายค่าตัวแปรเก่าซ้อนเข้าไปในตัวแปรใหม่ทันทีด้วย
		if match := varRegex.FindStringSubmatch(line); len(match) > 1 {
			varName := match[1]
			varValue := strings.Trim(match[2], "\"' ")
			
			// ถ้าค่าของตัวแปรใหม่มีตัวแปรเก่าผสมอยู่ ให้เคลียร์ก่อนเซฟ
			for k, v := range variables {
				varValue = strings.ReplaceAll(varValue, k, v)
			}
			
			variables[varName] = varValue
			continue
		}

		// 2. Variable Resolution: วนลูปแก้ตัวแปรจนกว่าจะไม่มีเครื่องหมาย $ เหลืออยู่ในบรรทัด (ยกเว้นแอปพลิเคชัน)
		// สั่งวนลูปซ้ำสูงสุด 5 รอบ ป้องกัน Loop นรก เคลียร์ตัวแปรซ้อนตัวแปรให้หมดเกลี้ยง
		for i := 0; i < 5; i++ {
			oldLine := line
			for vName, vValue := range variables {
				if strings.Contains(line, vName) {
					line = strings.ReplaceAll(line, vName, vValue)
				}
			}
			if line == oldLine { // ถ้าไม่มีอะไรให้เปลี่ยนแล้ว ให้หลุดลูปได้เลย
				break
			}
		}

		// 3. ตรวจจับคำสั่ง 'source = ...'
		if match := sourceRegex.FindStringSubmatch(line); len(match) > 1 {
			targetPath := match[1]
			if !strings.HasPrefix(targetPath, "~") && !filepath.IsAbs(targetPath) {
				targetPath = filepath.Join(currentDir, targetPath)
			}
			err := scanFile(targetPath, commands, visited, variables)
			if err != nil {
				return err
			}
			continue
		}

		// 4. ตรวจจับคำสั่งทำงานโปรแกรมภายนอก
		var cmd string
		if match := execRegex.FindStringSubmatch(line); len(match) > 1 {
			cmd = match[1]
		} else if match := bindRegex.FindStringSubmatch(line); len(match) > 1 {
			cmd = match[1]
		}

		cmd = strings.Trim(cmd, "\"'")
		if cmd != "" && !strings.HasPrefix(cmd, "$") {
			commands[cmd] = true
		}
	}

	return scanner.Err()
}

func printHeader() {
	header := `
🩺 DotDoctor | Hyprland Scanner (Phase 2C)
=========================================`
	color.Magenta(header)
	fmt.Println()
}

func getConfigPath() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "hyprland.conf"
	}
	return filepath.Join(home, ".config/hyprland/hyprland.conf")
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func printSummary(total, found, missing int) {
	fmt.Println("\n" + strings.Repeat("-", 40))
	color.White("📊 DEEP ITERATIVE SCAN SUMMARY:")
	fmt.Printf("   Total Unique Binaries Found: %d\n", total)
	color.Green("   Valid/Installed Dependencies: %d", found)
	color.Red("   Missing/Broken Dependencies:  %d", missing)
	fmt.Println(strings.Repeat("-", 40))

	if missing > 0 {
		color.Yellow("\n💡 Action Needed: Install missing binaries to fix your config environment.")
		os.Exit(1)
	} else {
		color.Cyan("\n✨ Perfect Score! Every single variable-resolved dependency is verified and healthy.")
		os.Exit(0)
	}
}