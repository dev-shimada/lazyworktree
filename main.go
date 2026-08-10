// Command lazyworktree is a terminal UI for managing git worktrees.
//
// On exit, if a worktree was selected (Enter), its path is printed to stdout
// so it can be consumed by a shell wrapper, e.g.:
//
//	lazyworktree() {
//		if [ "$#" -gt 0 ]; then
//			command lazyworktree "$@"
//			return
//		fi
//		local dir
//		dir=$(command lazyworktree) && [ -n "$dir" ] && cd -- "$dir"
//	}
//
// Forwarding args (rather than always calling `command lazyworktree` bare)
// matters: without it, flags like --default-config or --help silently launch
// the interactive TUI instead.
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

const usage = `lazyworktree: a TUI for managing git worktrees

Usage:
  lazyworktree                 launch the TUI in the current repository
  lazyworktree --default-config
                                print a starter ~/.config/lazyworktree/config.toml
                                (every setting commented out)
  lazyworktree --help, -h      show this help

On exit, if a worktree was selected (Enter), its path is printed to stdout
so a shell wrapper can cd into it. See the README's "Shell integration"
section for the wrapper function — it must forward its arguments, or flags
like --default-config silently launch the TUI instead.
`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--default-config":
			fmt.Print(config.ExampleConfig)
			return
		case "--help", "-h":
			fmt.Print(usage)
			return
		}
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
