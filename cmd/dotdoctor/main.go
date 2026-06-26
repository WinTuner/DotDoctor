package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/WinTuner/DotDoctor/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

var archPackageMap = map[string]string{
	// Binaries
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

	// Fonts
	"JetBrainsMono Nerd Font": "ttf-jetbrains-mono-nerd",
	"FiraCode Nerd Font":      "ttf-firacode-nerd",
	"MesloLGS NF":             "ttf-meslo-nerd-font-powerlevel10k",
	"CascadiaCode Nerd Font":  "ttf-cascadia-code-nerd",
	"Hack Nerd Font":          "ttf-hack-nerd",
	"Noto Sans":               "noto-fonts",
	"Noto Color Emoji":        "noto-fonts-emoji",
}

type DependencyItem struct {
	Name        string
	Type        string // "binary" or "font"
	Found       bool
	PackageName string // archPackageMap suggestion
}

type model struct {
	configPath    string
	binaries      []DependencyItem
	fonts         []DependencyItem
	hasFontConfig bool

	activeTab    int // 0 = Binaries, 1 = Fonts
	cursor       int
	scrollOffset int

	terminalWidth  int
	terminalHeight int

	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) getActiveItems() []DependencyItem {
	if m.activeTab == 0 {
		return m.binaries
	}
	return m.fonts
}

func (m model) getMaxVisibleItems() int {
	// Layout overhead:
	// Header: 2 lines
	// Tabs: 2 lines
	// Body Border: 2 lines
	// Suggestions/Status: 3 lines
	// Total overhead: ~9 lines
	h := m.terminalHeight - 9
	if h < 3 {
		return 3
	}
	return h
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "tab", "right", "l":
			m.activeTab = (m.activeTab + 1) % 2
			m.cursor = 0
			m.scrollOffset = 0
			return m, nil

		case "left", "h":
			m.activeTab = (m.activeTab - 1 + 2) % 2
			m.cursor = 0
			m.scrollOffset = 0
			return m, nil

		case "up", "k":
			items := m.getActiveItems()
			numItems := len(items)
			if numItems == 0 {
				return m, nil
			}
			m.cursor--
			if m.cursor < 0 {
				m.cursor = numItems - 1
			}
			maxVisible := m.getMaxVisibleItems()
			if m.cursor < m.scrollOffset {
				m.scrollOffset = m.cursor
			} else if m.cursor >= m.scrollOffset+maxVisible {
				m.scrollOffset = m.cursor - maxVisible + 1
			}
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil

		case "down", "j":
			items := m.getActiveItems()
			numItems := len(items)
			if numItems == 0 {
				return m, nil
			}
			m.cursor++
			if m.cursor >= numItems {
				m.cursor = 0
			}
			maxVisible := m.getMaxVisibleItems()
			if m.cursor < m.scrollOffset {
				m.scrollOffset = m.cursor
			} else if m.cursor >= m.scrollOffset+maxVisible {
				m.scrollOffset = m.cursor - maxVisible + 1
			}
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		maxVisible := m.getMaxVisibleItems()
		items := m.getActiveItems()
		if m.cursor >= m.scrollOffset+maxVisible {
			m.scrollOffset = m.cursor - maxVisible + 1
		}
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}
		if len(items) > 0 && m.scrollOffset > len(items)-maxVisible {
			m.scrollOffset = len(items) - maxVisible
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		}
	}

	return m, nil
}

func (m model) renderItem(item DependencyItem, selected bool) string {
	var cursor string
	if selected {
		cursor = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89b4fa")).Render("> ")
	} else {
		cursor = "  "
	}

	var status string
	if item.Found {
		status = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a6e3a1")).Render("[FOUND]  ")
	} else {
		status = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f38ba8")).Render("[MISSING]")
	}

	name := item.Name
	if selected {
		name = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f5e0dc")).Render(name)
	}

	return fmt.Sprintf("%s%s %s", cursor, status, name)
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder

	// 1. Header
	headerText := "🩺 DotDoctor | System Health Check"
	sb.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#11111b")).
		Background(lipgloss.Color("#cba6f7")).
		Padding(0, 2).
		MarginBottom(1).
		Render(headerText))
	sb.WriteString("\n")

	// 2. Tabs
	binariesTotal := len(m.binaries)
	binariesFound := 0
	for _, b := range m.binaries {
		if b.Found {
			binariesFound++
		}
	}

	fontsTotal := len(m.fonts)
	fontsFound := 0
	for _, f := range m.fonts {
		if f.Found {
			fontsFound++
		}
	}

	tab1Title := fmt.Sprintf("Binaries & Scripts (%d/%d)", binariesFound, binariesTotal)
	tab2Title := fmt.Sprintf("Fonts & Icons (%d/%d)", fontsFound, fontsTotal)

	var tab1, tab2 string
	activeTabStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#89b4fa")).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("#89b4fa")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#585b70")).
		Padding(0, 2)

	if m.activeTab == 0 {
		tab1 = activeTabStyle.Render(tab1Title)
		tab2 = inactiveTabStyle.Render(tab2Title)
	} else {
		tab1 = inactiveTabStyle.Render(tab1Title)
		tab2 = activeTabStyle.Render(tab2Title)
	}

	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, tab1, tab2)
	sb.WriteString(tabsRow)
	sb.WriteString("\n\n")

	// 3. Body
	items := m.getActiveItems()
	var bodyContent string

	if len(items) == 0 {
		bodyContent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a6adc8")).
			Render("No items detected in configuration.")
	} else {
		maxVisible := m.getMaxVisibleItems()
		var lines []string

		// Handle fontconfig check fail specifically on the fonts tab
		if m.activeTab == 1 && !m.hasFontConfig {
			warningBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#f38ba8")).
				Foreground(lipgloss.Color("#f38ba8")).
				Padding(0, 1).
				MarginBottom(1).
				Render("⚠️ Error: 'fc-list' not found. Install 'fontconfig' to verify fonts.")
			lines = append(lines, warningBox)
		}

		endIndex := m.scrollOffset + maxVisible
		if endIndex > len(items) {
			endIndex = len(items)
		}

		for i := m.scrollOffset; i < endIndex; i++ {
			lines = append(lines, m.renderItem(items[i], i == m.cursor))
		}

		// Fill remaining lines with empty space to avoid layout jumping
		visibleCount := endIndex - m.scrollOffset
		for i := visibleCount; i < maxVisible; i++ {
			lines = append(lines, "")
		}

		bodyContent = strings.Join(lines, "\n")
	}

	width := m.terminalWidth - 4
	if width < 10 {
		width = 10
	}

	bodyStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475a")).
		Padding(1, 2).
		Width(width)

	sb.WriteString(bodyStyle.Render(bodyContent))
	sb.WriteString("\n")

	// 4. Status and Help Bar
	var hint string
	if len(items) > 0 && m.cursor < len(items) {
		selectedItem := items[m.cursor]
		if !selectedItem.Found && selectedItem.PackageName != "" {
			hintStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f9e2af")).
				Padding(0, 1)
			hint = hintStyle.Render(fmt.Sprintf("💡 Suggestion: Run 'sudo pacman -S %s'", selectedItem.PackageName))
		} else if !selectedItem.Found {
			hintStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f9e2af")).
				Padding(0, 1)
			hint = hintStyle.Render("💡 Suggestion: Dependency not found, search your package manager.")
		} else {
			successStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#a6e3a1")).
				Padding(0, 1)
			hint = successStyle.Render("✨ Dependency verified and healthy!")
		}
	}
	if hint != "" {
		sb.WriteString(hint)
		sb.WriteString("\n")
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#585b70"))
	helpText := "Tab: Switch Tab • ↑/↓: Scroll • Q: Quit"
	sb.WriteString(helpStyle.Render(helpText))

	return sb.String()
}

func main() {
	configPath := getConfigPath()

	s := scanner.NewScanner()

	color.Cyan("🔍 Initiating scan starting from: %s\n", configPath)

	err := s.Scan(configPath)
	if err != nil {
		color.Red("❌ Error scanning configuration: %v", err)
		os.Exit(1)
	}

	if len(s.Binaries) == 0 && len(s.Fonts) == 0 {
		color.Yellow("⚠️ No dependencies found across your configuration files.")
		os.Exit(0)
	}

	// Verify dependencies and build list
	var binariesList []DependencyItem
	sortedCmds := sortMapKeys(s.Binaries)
	for _, cmd := range sortedCmds {
		expandedPath := scanner.ExpandPath(cmd)

		found := false
		if filepath.IsAbs(expandedPath) {
			_, err := os.Stat(expandedPath)
			if err == nil {
				found = true
			}
		} else {
			_, err := exec.LookPath(expandedPath)
			if err == nil {
				found = true
			}
		}

		pkg := archPackageMap[cmd]
		binariesList = append(binariesList, DependencyItem{
			Name:        cmd,
			Type:        "binary",
			Found:       found,
			PackageName: pkg,
		})
	}

	var fontsList []DependencyItem
	hasFontConfig := scanner.CheckFontConfig()
	sortedFonts := sortMapKeys(s.Fonts)
	for _, font := range sortedFonts {
		found := false
		if hasFontConfig {
			found = scanner.VerifyFont(font)
		}
		pkg := archPackageMap[font]
		fontsList = append(fontsList, DependencyItem{
			Name:        font,
			Type:        "font",
			Found:       found,
			PackageName: pkg,
		})
	}

	m := model{
		configPath:     configPath,
		binaries:       binariesList,
		fonts:          fontsList,
		hasFontConfig:  hasFontConfig,
		terminalWidth:  80,
		terminalHeight: 24,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Print summary to stdout upon exit
	fm := finalModel.(model)

	totalBin := len(fm.binaries)
	totalFont := len(fm.fonts)

	fBin := 0
	mBin := 0
	for _, b := range fm.binaries {
		if b.Found {
			fBin++
		} else {
			mBin++
		}
	}

	fFont := 0
	mFont := 0
	for _, f := range fm.fonts {
		if f.Found {
			fFont++
		} else {
			mFont++
		}
	}

	printSummary(totalBin, fBin, mBin, totalFont, fFont, mFont)
}

func sortMapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
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

func printSummary(totalBin, foundBin, missBin, totalFont, foundFont, missFont int) {
	fmt.Println(strings.Repeat("-", 50))
	color.White("📊 DEEP ITERATIVE SCAN SUMMARY:")

	fmt.Printf("   %-20s Total: %-3d  ✅ %-3d  ❌ %-3d\n", "Binaries:", totalBin, foundBin, missBin)
	fmt.Printf("   %-20s Total: %-3d  ✅ %-3d  ❌ %-3d\n", "Fonts:", totalFont, foundFont, missFont)

	fmt.Println(strings.Repeat("-", 50))

	if missBin > 0 || missFont > 0 {
		color.Yellow("\n💡 Action Needed: Install missing dependencies to fix your config environment.")
		os.Exit(1)
	} else {
		color.Cyan("\n✨ Perfect Score! Every single dependency is verified and healthy.")
		os.Exit(0)
	}
}
