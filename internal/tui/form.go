package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-shimada/lazyworktree/internal/git"
	"github.com/dev-shimada/lazyworktree/internal/herdr"
)

const (
	fieldBranch = iota
	fieldPath
	fieldHerdr
	fieldCount
)

// createForm collects a branch name and a destination path for a new worktree,
// plus an option to hand the new checkout off to herdr as a pane.
type createForm struct {
	repoRoot  string
	herdrMode bool

	branch textinput.Model
	path   textinput.Model
	focus  int

	pathEdited     bool
	openHerdr      bool
	herdrAvailable bool
	err            string
}

func newCreateForm(repoRoot string, herdrMode bool) createForm {
	branch := textinput.New()
	branch.Placeholder = "branch name (existing or new)"
	branch.Focus()
	branch.CharLimit = 200
	branch.Width = 50

	path := textinput.New()
	path.Placeholder = "destination path"
	path.CharLimit = 400
	path.Width = 50

	available := herdr.Available()
	return createForm{
		repoRoot:       repoRoot,
		herdrMode:      herdrMode,
		branch:         branch,
		path:           path,
		focus:          fieldBranch,
		openHerdr:      available,
		herdrAvailable: available,
	}
}

func (f createForm) defaultWorktreePath(branch string) string {
	if f.herdrMode {
		return herdr.DefaultWorktreePath(f.repoRoot, branch)
	}
	return git.DefaultWorktreePath(f.repoRoot, branch)
}

func (f *createForm) syncFocus() {
	f.branch.Blur()
	f.path.Blur()
	switch f.focus {
	case fieldBranch:
		f.branch.Focus()
	case fieldPath:
		f.path.Focus()
	}
}

// result returns the AddOptions and destination path for the form's current values.
func (f createForm) result() (path string, opts git.AddOptions) {
	branch := f.branch.Value()
	opts.Branch = branch
	opts.CreateBranch = !git.BranchExists(f.repoRoot, branch)
	return f.path.Value(), opts
}

func (f createForm) Update(msg tea.Msg) (createForm, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "tab", "down":
			f.focus = (f.focus + 1) % fieldCount
			f.syncFocus()
			return f, nil
		case "shift+tab", "up":
			f.focus = (f.focus - 1 + fieldCount) % fieldCount
			f.syncFocus()
			return f, nil
		case " ":
			if f.focus == fieldHerdr {
				f.openHerdr = !f.openHerdr
				return f, nil
			}
		}
	}

	var cmd tea.Cmd
	switch f.focus {
	case fieldBranch:
		f.branch, cmd = f.branch.Update(msg)
		if !f.pathEdited {
			branch := f.branch.Value()
			if branch == "" {
				f.path.SetValue("")
			} else {
				f.path.SetValue(f.defaultWorktreePath(branch))
			}
		}
	case fieldPath:
		before := f.path.Value()
		f.path, cmd = f.path.Update(msg)
		if f.path.Value() != before {
			f.pathEdited = true
		}
	}
	return f, cmd
}

func (f createForm) herdrLine() string {
	if !f.herdrAvailable {
		return footerStyle.Render("herdr not found in PATH — skipping")
	}
	box := "[ ]"
	if f.openHerdr {
		box = "[x]"
	}
	line := box + " open in herdr after create"
	if f.focus == fieldHerdr {
		line = "> " + line + "  (space to toggle)"
	}
	return line
}

func (f createForm) View() string {
	body := formLabelStyle.Render("Branch") + "\n" + f.branch.View() + "\n\n" +
		formLabelStyle.Render("Path") + "\n" + f.path.View() + "\n\n" +
		f.herdrLine()
	if f.err != "" {
		body += "\n\n" + statusErrStyle.Render(f.err)
	}
	body += "\n\n" + footerStyle.Render("tab: switch field  •  enter: create  •  esc: cancel")
	return formBoxStyle.Render(titleStyle.Render("New worktree") + "\n\n" + body)
}
