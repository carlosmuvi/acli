package main

import (
	"fmt"
	"os"

	"github.com/carlosmuvi/acli/internal/doctor"
	"github.com/carlosmuvi/acli/internal/sdk"
	"github.com/carlosmuvi/acli/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// Build metadata, injected by GoReleaser via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "--version":
			fmt.Printf("acli %s (commit %s, built %s)\n", version, commit, date)
			return
		}
	}

	tools := sdk.Discover()
	report := doctor.Run(tools)

	// Hard-fail before the TUI if a required tool (adb) is missing.
	if report.HasHardFailure() {
		printDoctor(report)
		fmt.Fprintln(os.Stderr, "\nA required tool is missing — cannot continue.")
		os.Exit(1)
	}

	p := tea.NewProgram(tui.New(tools, report), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "acli:", err)
		os.Exit(1)
	}
}

func printDoctor(r doctor.Report) {
	fmt.Println("acli doctor")
	for _, c := range r.Checks {
		mark := "✓"
		if !c.OK {
			mark = "✗"
		}
		fmt.Printf("  %s %-18s %s\n", mark, c.Name, c.Detail)
		if !c.OK {
			fmt.Printf("      fix: %s\n", c.Fix)
		}
	}
}
