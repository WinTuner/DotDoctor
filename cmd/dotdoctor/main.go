package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/WinTuner/DotDoctor/internal/scanner"
)

func main() {
	pathFlag := flag.String("path", ".", "Path to the dotfiles or config directory to inspect")
	flag.Parse()

	absPath, err := filepath.Abs(*pathFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve path: %v\n", err)
		os.Exit(1)
	}

	result := scanner.Analyze(absPath)

	fmt.Println("DotDoctor")
	fmt.Printf("Target path: %s\n", result.Root)
	fmt.Printf("Status: %s\n", result.Status)
}
