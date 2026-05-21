package scanner

import "testing"

func TestAnalyzeReturnsPlaceholderResult(t *testing.T) {
	root := "/tmp/dotfiles"

	result := Analyze(root)

	if result.Root != root {
		t.Fatalf("expected root %q, got %q", root, result.Root)
	}

	if result.Status != "scanner placeholder initialized" {
		t.Fatalf("expected placeholder status, got %q", result.Status)
	}
}
