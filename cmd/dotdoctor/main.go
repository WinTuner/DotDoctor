package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/WinTuner/DotDoctor/internal/scanner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/fsnotify/fsnotify"
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
	visitedFiles  map[string]bool

	activeTab    int // 0 = Binaries, 1 = Fonts
	cursor       int
	scrollOffset int

	terminalWidth  int
	terminalHeight int

	quitting     bool
	notification string
}

type clearNotificationMsg struct{}

type reloadMsg struct {
	binaries     []DependencyItem
	fonts        []DependencyItem
	visitedFiles map[string]bool
}

type installFinishedMsg struct {
	itemIndex int
	tab       int
	err       error
}

type watcherState struct {
	watcher    *fsnotify.Watcher
	watched    map[string]bool
	configPath string
	program    *tea.Program
}

func startConfigWatcher(configPath string, initialFiles map[string]bool, p *tea.Program) (*watcherState, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ws := &watcherState{
		watcher:    w,
		watched:    make(map[string]bool),
		configPath: configPath,
		program:    p,
	}

	ws.updateWatchList(initialFiles)
	go ws.watchLoop()

	return ws, nil
}

func (ws *watcherState) updateWatchList(newFiles map[string]bool) {
	targets := make(map[string]bool)
	for f := range newFiles {
		targets[f] = true
	}
	absConfig, err := filepath.Abs(scanner.ExpandPath(ws.configPath))
	if err == nil {
		targets[absConfig] = true
	}

	// Add new watched files
	for f := range targets {
		if !ws.watched[f] {
			err := ws.watcher.Add(f)
			if err == nil {
				ws.watched[f] = true
			}
		}
	}

	// Remove untracked watched files
	for f := range ws.watched {
		if !targets[f] {
			_ = ws.watcher.Remove(f)
			delete(ws.watched, f)
		}
	}
}

func (ws *watcherState) watchLoop() {
	var timer *time.Timer
	const debounceDuration = 100 * time.Millisecond

	for {
		select {
		case event, ok := <-ws.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) {
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(debounceDuration, func() {
					ws.handleReload()
				})
			}
		case err, ok := <-ws.watcher.Errors:
			if !ok {
				return
			}
			_ = err
		}
	}
}

func (ws *watcherState) handleReload() {
	s := scanner.NewScanner()
	err := s.Scan(ws.configPath)
	if err != nil {
		return
	}

	var binariesList []DependencyItem
	sortedCmds := sortMapKeys(s.Binaries)
	for _, cmd := range sortedCmds {
		found := checkBinaryFound(cmd)
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
		found := checkFontFound(font, hasFontConfig)
		pkg := archPackageMap[font]
		fontsList = append(fontsList, DependencyItem{
			Name:        font,
			Type:        "font",
			Found:       found,
			PackageName: pkg,
		})
	}

	ws.updateWatchList(s.VisitedFiles)

	ws.program.Send(reloadMsg{
		binaries:     binariesList,
		fonts:        fontsList,
		visitedFiles: s.VisitedFiles,
	})
}

func (ws *watcherState) Close() {
	if ws.watcher != nil {
		_ = ws.watcher.Close()
	}
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
	// Suggestions/Status/Notification: 3 lines
	// Help menu: 1 line
	// Total overhead: ~10 lines
	h := m.terminalHeight - 10
	if h < 3 {
		return 3
	}
	return h
}

func (m model) clearNotificationCmd() tea.Cmd {
	return tea.Tick(4*time.Second, func(t time.Time) tea.Msg {
		return clearNotificationMsg{}
	})
}

func checkBinaryFound(cmd string) bool {
	expandedPath := scanner.ExpandPath(cmd)
	if filepath.IsAbs(expandedPath) {
		_, err := os.Stat(expandedPath)
		return err == nil
	}
	_, err := exec.LookPath(expandedPath)
	return err == nil
}

func checkFontFound(font string, hasFontConfig bool) bool {
	if !hasFontConfig {
		return false
	}
	return scanner.VerifyFont(font)
}

func getInstallerCommand(pkg string) *exec.Cmd {
	if _, err := exec.LookPath("yay"); err == nil {
		return exec.Command("yay", "-S", pkg)
	}
	if _, err := exec.LookPath("paru"); err == nil {
		return exec.Command("paru", "-S", pkg)
	}
	return exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg)
}

func exportReport(m model, visitedFiles map[string]bool) error {
	f, err := os.Create("dotdoctor_report.md")
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	fmt.Fprintln(w, "# 🩺 DotDoctor | System Health Report")
	fmt.Fprintf(w, "- **Date of Execution:** %s\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	fmt.Fprintf(w, "- **Target Config:** `%s`\n\n", m.configPath)

	fmt.Fprintln(w, "## 📂 Scanned Files")
	var sortedFiles []string
	for file := range visitedFiles {
		sortedFiles = append(sortedFiles, file)
	}
	sort.Strings(sortedFiles)
	for _, file := range sortedFiles {
		fmt.Fprintf(w, "- `%s`\n", file)
	}
	fmt.Fprintln(w)

	// Summary stats
	binariesTotal := len(m.binaries)
	binariesFound := 0
	var missingBinaries []DependencyItem
	for _, b := range m.binaries {
		if b.Found {
			binariesFound++
		} else {
			missingBinaries = append(missingBinaries, b)
		}
	}

	fontsTotal := len(m.fonts)
	fontsFound := 0
	var missingFonts []DependencyItem
	for _, font := range m.fonts {
		if font.Found {
			fontsFound++
		} else {
			missingFonts = append(missingFonts, font)
		}
	}

	fmt.Fprintln(w, "## 📊 Scan Summary Score")
	fmt.Fprintf(w, "- **Binaries:** %d / %d verified healthy\n", binariesFound, binariesTotal)
	fmt.Fprintf(w, "- **Fonts:** %d / %d verified healthy\n\n", fontsFound, fontsTotal)

	if len(missingBinaries) > 0 {
		fmt.Fprintln(w, "## ❌ Missing Binaries & Scripts")
		for _, b := range missingBinaries {
			fmt.Fprintf(w, "- **`%s`**\n", b.Name)
			if b.PackageName != "" {
				fmt.Fprintf(w, "  - Recommendation: Install package `%s` (AUR/Official)\n\n", b.PackageName)
			} else {
				fmt.Fprintf(w, "  - Recommendation: Search for this binary in your package manager\n\n")
			}
		}
		fmt.Fprintln(w)
	}

	if len(missingFonts) > 0 {
		fmt.Fprintln(w, "## ❌ Missing Fonts & Icons")
		for _, font := range missingFonts {
			fmt.Fprintf(w, "- **`%s`**\n", font.Name)
			if font.PackageName != "" {
				fmt.Fprintf(w, "  - Recommendation: Install package `%s` (AUR/Official)\n\n", font.PackageName)
			} else {
				fmt.Fprintf(w, "  - Recommendation: Search for this font in your package manager\n\n")
			}
		}
		fmt.Fprintln(w)
	}

	if len(missingBinaries) == 0 && len(missingFonts) == 0 {
		fmt.Fprintln(w, "## ✨ System Status: Healthy")
		fmt.Fprintln(w, "All scanned binaries and fonts are verified and present on the system!")
	} else {
		fmt.Fprintln(w, "## 💡 Quick Fix Recommendation")
		fmt.Fprintln(w, "You can trigger auto-fix installations directly from the interactive DotDoctor TUI dashboard by highlighting any missing dependency and pressing `i` or `x`.")
	}

	return w.Flush()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearNotificationMsg:
		m.notification = ""
		return m, nil

	case reloadMsg:
		m.binaries = msg.binaries
		m.fonts = msg.fonts
		m.visitedFiles = msg.visitedFiles

		// Preserve cursor bounds
		items := m.getActiveItems()
		if m.cursor >= len(items) {
			m.cursor = len(items) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
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

		m.notification = "🔄 Config modified! Hot-reloaded successfully."
		return m, m.clearNotificationCmd()

	case installFinishedMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf("❌ Installation failed: %v", msg.err)
			return m, m.clearNotificationCmd()
		}

		if msg.tab == 0 {
			if msg.itemIndex < len(m.binaries) {
				item := &m.binaries[msg.itemIndex]
				item.Found = checkBinaryFound(item.Name)
				if item.Found {
					m.notification = fmt.Sprintf("✅ Installed and verified: %s", item.Name)
				} else {
					m.notification = fmt.Sprintf("⚠️ Finished installation, but %s is still missing", item.Name)
				}
			}
		} else {
			if msg.itemIndex < len(m.fonts) {
				item := &m.fonts[msg.itemIndex]
				item.Found = checkFontFound(item.Name, m.hasFontConfig)
				if item.Found {
					m.notification = fmt.Sprintf("✅ Installed and verified font: %s", item.Name)
				} else {
					m.notification = fmt.Sprintf("⚠️ Finished installation, but font %s is still missing", item.Name)
				}
			}
		}
		return m, m.clearNotificationCmd()

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

		case "i", "x":
			items := m.getActiveItems()
			if len(items) == 0 || m.cursor >= len(items) {
				return m, nil
			}
			item := items[m.cursor]
			if item.Found {
				m.notification = "✨ Dependency is already verified and healthy!"
				return m, m.clearNotificationCmd()
			}
			if item.PackageName == "" {
				m.notification = fmt.Sprintf("⚠️ No installation hint available for %s", item.Name)
				return m, m.clearNotificationCmd()
			}

			// Run installation command
			c := getInstallerCommand(item.PackageName)
			return m, tea.ExecProcess(c, func(err error) tea.Msg {
				return installFinishedMsg{
					itemIndex: m.cursor,
					tab:       m.activeTab,
					err:       err,
				}
			})

		case "e":
			err := exportReport(m, m.visitedFiles)
			if err != nil {
				m.notification = fmt.Sprintf("❌ Report export failed: %v", err)
			} else {
				m.notification = "📝 Report exported to dotdoctor_report.md!"
			}
			return m, m.clearNotificationCmd()
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

	// 4. Status, Notifications and Help Bar
	var hint string
	if m.notification != "" {
		notificationStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#a6e3a1")).
			Padding(0, 1)
		if strings.HasPrefix(m.notification, "❌") {
			notificationStyle = notificationStyle.Foreground(lipgloss.Color("#f38ba8"))
		} else if strings.HasPrefix(m.notification, "⚠️") {
			notificationStyle = notificationStyle.Foreground(lipgloss.Color("#f9e2af"))
		}
		hint = notificationStyle.Render(m.notification)
	} else if len(items) > 0 && m.cursor < len(items) {
		selectedItem := items[m.cursor]
		if !selectedItem.Found && selectedItem.PackageName != "" {
			hintStyle := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#f9e2af")).
				Padding(0, 1)
			hint = hintStyle.Render(fmt.Sprintf("💡 Suggestion: Run 'sudo pacman -S %s' or press 'i'/'x' to auto-install", selectedItem.PackageName))
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
	helpText := "Tab: Switch Tab • ↑/↓: Scroll • I/X: Auto-Install • E: Export Report • Q: Quit"
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
		found := checkBinaryFound(cmd)
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
		found := checkFontFound(font, hasFontConfig)
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
		visitedFiles:   s.VisitedFiles,
		terminalWidth:  80,
		terminalHeight: 24,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start configuration watcher
	ws, err := startConfigWatcher(configPath, s.VisitedFiles, p)
	if err == nil {
		defer ws.Close()
	}

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
