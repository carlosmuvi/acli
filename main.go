package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carlosmuvi/acli/internal/doctor"
	"github.com/carlosmuvi/acli/internal/sdk"
	"github.com/carlosmuvi/acli/internal/tui"
	"github.com/carlosmuvi/acli/internal/web"

	tea "github.com/charmbracelet/bubbletea"
)

// Build metadata, injected by GoReleaser via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const usage = `acli — Android emulator & logcat manager

Usage:
  acli            launch the interactive terminal UI (default)
  acli serve      serve the web dashboard in your browser
  acli version    print version
  acli help       show this help

Run "acli serve -h" for serve options.`

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "version", "-v", "--version":
		fmt.Printf("acli %s (commit %s, built %s)\n", version, commit, date)
	case "help", "-h", "--help":
		fmt.Println(usage)
	case "serve":
		runServe(os.Args[2:])
	default:
		runTUI()
	}
}

// gate discovers tooling, runs the doctor, and exits if a required tool is
// missing. It returns the resolved tools and report on success.
func gate() (sdk.Tools, doctor.Report) {
	tools := sdk.Discover()
	report := doctor.Run(tools)
	if report.HasHardFailure() {
		printDoctor(report)
		fmt.Fprintln(os.Stderr, "\nA required tool is missing — cannot continue.")
		os.Exit(1)
	}
	return tools, report
}

func runTUI() {
	tools, report := gate()
	p := tea.NewProgram(tui.New(tools, report), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "acli:", err)
		os.Exit(1)
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 7070, "port to listen on")
	noOpen := fs.Bool("no-open", false, "don't open the browser automatically")
	_ = fs.Parse(args)

	tools, report := gate()
	srv := web.NewServer(tools, report)
	url := fmt.Sprintf("http://localhost:%d", *port)
	fmt.Printf("acli web UI → %s  (ctrl+c to stop)\n", url)
	if !*noOpen {
		web.OpenBrowser(url)
	}
	if err := srv.Listen(*port); err != nil {
		fmt.Fprintln(os.Stderr, "acli serve:", err)
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
