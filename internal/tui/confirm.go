package tui

import "github.com/dev-shimada/lazyworktree/internal/git"

// confirmDelete holds the state for the "remove worktree" confirmation dialog.
type confirmDelete struct {
	target git.Worktree
	force  bool
}

func (c confirmDelete) View() string {
	branch := c.target.Branch
	if branch == "" {
		branch = "(detached)"
	}
	body := "Remove worktree for " + branchStyle.Render(branch) + "?\n" +
		footerStyle.Render(c.target.Path)

	forceLine := "\n\n[ ] force (discard uncommitted changes)"
	if c.force {
		forceLine = "\n\n[x] force (discard uncommitted changes)"
	}
	body += forceLine

	body += "\n\n" + footerStyle.Render("y: confirm  •  f: toggle force  •  n/esc: cancel")
	return confirmBoxStyle.Render(titleStyle.Render("Confirm delete") + "\n\n" + body)
}
