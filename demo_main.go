//go:build demo

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ericonyourphone/sidea/app"
	"github.com/ericonyourphone/sidea/cli"
	"github.com/ericonyourphone/sidea/demo"
	"github.com/ericonyourphone/sidea/ui"
)

// demoUsage is the help line for `sidea demo` (only in demo builds).
const demoUsage = "  sidea demo     Launch in demo mode, on built-in sample data\n"

// runDemo launches the TUI against the demo source. "sidea demo <cmd> ..."
// runs a CLI subcommand against the same demo data instead (for recordings
// and manual testing without a Spotify account).
func runDemo() {
	src := demo.NewDemoSource()
	if len(os.Args) > 2 && os.Args[2] == "snapshot" {
		flags := flag.NewFlagSet("sidea demo snapshot", flag.ContinueOnError)
		width := flags.Int("width", 140, "terminal columns")
		height := flags.Int("height", 30, "terminal rows")
		theme := flags.String("theme", "spotify", "color theme")
		color := flags.Bool("color", false, "include true-color escape sequences")
		search := flags.String("search", "", "show a demo search popup")
		if err := flags.Parse(os.Args[3:]); err != nil {
			os.Exit(2)
		}
		if *width < 60 || *width > 500 || *height < 20 || *height > 200 {
			fmt.Fprintln(os.Stderr, "Preview size must be 60–500 columns and 20–200 rows")
			os.Exit(2)
		}
		palette, err := app.ResolveTheme(*theme, app.Palette{})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		app.ApplyTheme(palette)
		if *color {
			lipgloss.SetColorProfile(termenv.TrueColor)
		}
		fmt.Println(ui.DemoPreview(src, *width, *height, *search))
		return
	}
	if len(os.Args) > 2 && cli.IsCommand(os.Args[2]) {
		r := &cli.Runner{Src: src, Out: os.Stdout, Err: os.Stderr}
		os.Exit(r.Run(context.Background(), os.Args[2:]))
	}

	if len(os.Args) > 2 && os.Args[2] == "setup" {
		// The guided setup, against stand-ins (see demo_setup.go).
		deps, cleanup := demoSetupDeps()
		defer cleanup()
		s := app.NewSetup(deps, app.SetupFull)
		if _, err := tea.NewProgram(s, tea.WithAltScreen()).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	fmt.Println("  Starting sidea in demo mode...")
	fmt.Println("")

	// Demo mode ignores the user's config (recordings must be reproducible)
	// and never persists :theme changes. SIDEA_ART still applies, for trying
	// kitty art without a Spotify account.
	configureArt(os.Getenv("SIDEA_ART"), "SIDEA_ART")
	m := ui.NewModel(src, app.Options{}).WithDemo()
	p := tea.NewProgram(m, programOptions()...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
