// Command lazyworktree is a terminal UI for managing git worktrees.
//
// On exit, if a worktree was selected (Enter), its path is printed to stdout
// so it can be consumed by a shell wrapper, e.g.:
//
//	lazyworktree() {
//		local dir
//		dir=$(command lazyworktree) && [ -n "$dir" ] && cd -- "$dir"
//	}
//
// `lazyworktree --default-config` prints a starter
// ~/.config/lazyworktree/config.toml, with every setting commented out.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-shimada/lazyworktree/internal/config"
	"github.com/dev-shimada/lazyworktree/internal/tui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--default-config" {
		fmt.Print(config.ExampleConfig)
		return
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "lazyworktree:", err)
		os.Exit(1)
	}
}

func run() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	model, err := tui.New(cwd)
	if err != nil {
		return err
	}

	// Render to stderr so the TUI stays visible when stdout is captured by a
	// shell wrapper (e.g. `dir=$(command lazyworktree)`); only the final
	// selected path goes to stdout.
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := final.(tui.Model); ok && m.SelectedPath != "" {
		fmt.Println(m.SelectedPath)
	}
	return nil
}
