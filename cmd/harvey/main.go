// Package main is the entry point for Harvey, an AI-powered assistant for
// Arch Linux + Hyprland. It wires together domain interfaces, application
// use cases, and infrastructure adapters, then launches the Bubbletea TUI.
//
// In PR 1, this is a minimal skeleton that will be expanded in PR 5.
package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "harvey: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	fmt.Println("Harvey — Arch Linux AI Assistant")
	fmt.Println("PR 1: Domain interfaces and model verification")
	return nil
}
