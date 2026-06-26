# 🩺 DotDoctor

<p align="center">
  <a href="https://github.com/WinTuner/DotDoctor/releases"><img src="https://img.shields.io/badge/Version-1.0.0-blueviolet?style=for-the-badge&logo=go" alt="Version"></a>
  <img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-blue?style=for-the-badge" alt="Platform">
  <a href="./LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License"></a>
</p>

---

**DotDoctor** is a universal systems configuration engine and diagnostic utility built for Linux, macOS, and Windows power-users who manage modular dotfiles and custom tiling window managers (Hyprland, Sway, i3, Aerospace, GlazeWM). It streamlines the friction of maintaining complex configuration environments by automating health checks across your ecosystem.

Just run the command, and the engine acts as an intelligent system auditor—recursively tracing configuration modules, resolving stacked variable strings, verifying binaries and Nerd Fonts, and empowering you to instantly orchestrate automated system repairs (`❌ [MISSING]`) via an interactive, ultra-polished **Terminal User Interface (TUI)**.

---

## ✨ Features

* 🧠 **Zero-Config Auto-Detection:** No more typing lengthy manual file paths. Running `dotdoctor` standalone dynamically probes the operating environment to detect, target, and verify the active desktop window manager configurations out of the box.
* 🔗 **5-Pass Chained Variable Resolution:** A robust iterative parsing engine designed to dynamically resolve local config variable definitions nested up to 5 layers deep, guaranteeing pristine extraction accuracy for complex multi-sourced setups.
* 🎨 **Catppuccin TUI Dashboard:** Features a clean, switchable tab arrangement crafted with `bubbletea` and `lipgloss`. Modeled after the popular Catppuccin Mocha palette preferred by the r/unixporn community.
* 🛠️ **Active Auto-Fixer Engine:** Highlight any red `[MISSING]` dependency and press `i` or `x`. The application suspends its viewport loop to natively pipe your system’s default package helper stream directly into the terminal stream without breaking application layout.
* 🔄 **Live Hot-Reloading (`fsnotify`):** Monitors targeted target nodes asynchronously in the background. Saving changes to any primary or recursively sourced configuration file dynamically pushes thread-safe reload updates straight into the TUI layout.
* 📝 **Markdown Medical Report:** Instantly hit `e` to generate a structured system status audit document saved cleanly as `dotdoctor_report.md` in your current working directory.
* 🌍 **Cross-Platform Architecture (Build Tags):** Leverages explicit Go build tags to cleanly segregate platform-specific verification implementations, compiling cleanly across three independent host operating systems:
  * **Linux (CachyOS/Arch):** Audits icon system mappings using `fc-list` and facilitates installation via `pacman`, `yay`, or `paru`.
  * **macOS (Darwin):** Traverses system and user font paths recursively and triggers package setup via `Homebrew (brew)`.
  * **Windows:** Evaluates missing system fonts straight from `%SystemRoot%\Fonts` and triggers package deployment via `Winget`.

---

## 🛠️ Installation

Install `DotDoctor` instantly using your system's native package manager or fetch from source:

### 📦 Official Channels & Source

| OS | Package Manager / Source | Command |
| :--- | :--- | :--- |
| **Linux (Arch/Cachy)** | AUR (Arch User Repository) | `yay -S dotdoctor-git` or `paru -S dotdoctor-git` |
| **macOS (Darwin)** | Homebrew Tap | `brew tap WinTuner/dotdoctor && brew install dotdoctor` |
| **Windows** | Winget | `winget install --id=WinTuner.DotDoctor -e` |
| **Any Platform** | Install from Source (Go CLI) | `go install github.com/WinTuner/DotDoctor@latest` |

### 💾 Pre-compiled Binaries
If you do not have Go or a package manager configured on your local machine, grab the optimized, pre-compiled static executables compiled for your target CPU architecture (including Apple Silicon `arm64`) directly from the:
👉 **[GitHub Releases](https://github.com/WinTuner/DotDoctor/releases)** page.

Once unpacked, move the compiled executable block to your global binary system locations:
* **Linux / macOS:** Move binary to `/usr/local/bin/`
* **Windows:** Append binary location to your Environment `PATH` variable.

---

## 🎮 Usage

Run the global shortcut command from any terminal context:

```bash
dotdoctor
