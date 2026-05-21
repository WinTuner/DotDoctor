# DotDoctor

DotDoctor is a static analysis tool for Linux desktop users who manage modular dotfiles, especially Hyprland setups on Arch Linux and CachyOS.

It is built to walk split configuration trees, follow `source = ...` includes, resolve nested variables, extract external binary dependencies, and verify whether each command is available on the current system with Go's `exec.LookPath()`.

## Features

- Recursively scans modular dotfile layouts instead of assuming a single flat config file
- Understands Hyprland-style `source = ...` patterns for split configs
- Resolves nested variables before dependency extraction
- Identifies external binaries referenced by your config
- Designed for local verification on Linux desktop environments
- Lightweight Go codebase with a simple CLI entrypoint

## Installation

### Prerequisites

- Go 1.22 or newer
- A Linux environment with your dotfiles available locally

### Build from source

```bash
git clone https://github.com/WinTuner/DotDoctor.git
cd DotDoctor
go build ./cmd/dotdoctor
```

This produces a local `dotdoctor` binary in the repository root when used with:

```bash
go build -o dotdoctor ./cmd/dotdoctor
```

## Quick Start

The current project scaffold includes a placeholder CLI and core package so development can start from a clean, conventional layout.

Run the CLI against a config directory:

```bash
go run ./cmd/dotdoctor --path ~/.config/hypr
```

Example output:

```text
DotDoctor
Target path: /home/user/.config/hypr
Status: scanner placeholder initialized
```

## Project Layout

```text
.
├── cmd/
│   └── dotdoctor/
│       └── main.go
├── internal/
│   └── scanner/
│       ├── scanner.go
│       └── scanner_test.go
├── go.mod
├── LICENSE
└── README.md
```

## Development Notes

The scaffold is intentionally small:

- `cmd/dotdoctor` contains the CLI entrypoint
- `internal/scanner` contains the placeholder core logic for the future recursive analyzer

From here, the scanner can grow into parsing sourced files, resolving variable chains, extracting command invocations, and checking them with `exec.LookPath()`.
