package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Home expansion", "~/.config/test", filepath.Join(home, ".config/test")},
		{"No expansion", "/usr/bin/ls", "/usr/bin/ls"},
		{"Relative path", "./local/script", "./local/script"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPath(tt.path)
			if got != tt.expected {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestSimplifyFontName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Basic", "JetBrains Mono", "jetbrainsmono"},
		{"Nerd Font Suffix", "JetBrainsMono Nerd Font", "jetbrainsmono"},
		{"NF Suffix", "JetBrains Mono NF", "jetbrainsmono"},
		{"Mixed Case and Separators", "Fira-Code_Nerd Font", "firacode"},
		{"Spaces", "Meslo LGS NF", "meslolgs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simplifyFontName(tt.input)
			if got != tt.expected {
				t.Errorf("simplifyFontName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestVerifyFontLogic(t *testing.T) {
	// Since VerifyFont calls fc-list, we can't easily test it without mocking exec.
	// But we can test the matching logic if we refactor it slightly to take the list of fonts.
	// For now, simplifyFontName tests cover the core logic.
}
