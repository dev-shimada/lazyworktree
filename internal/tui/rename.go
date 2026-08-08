package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// renameForm collects a new name for either a branch (git mode) or the
// hosting herdr workspace's label (herdr mode). Neither mode touches the
// worktree's checkout directory.
type renameForm struct {
	input  textinput.Model
	herdr  bool
	target string // branch being renamed; unused in herdr mode
}

func newRenameForm(herdrMode bool, currentBranch string) renameForm {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 50
	if herdrMode {
		ti.Placeholder = "new herdr label"
	} else {
		ti.Placeholder = "new branch name"
		ti.SetValue(currentBranch)
	}
	return renameForm{input: ti, herdr: herdrMode, target: currentBranch}
}

func (f renameForm) Update(msg tea.Msg) (renameForm, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

func (f renameForm) View() string {
	label := "New branch name"
	hint := "renames the branch only — the worktree directory is left as-is"
	if f.herdr {
		label = "New herdr label"
		hint = "renames only the herdr workspace label — nothing on disk changes"
	}
	body := formLabelStyle.Render(label) + "\n" + f.input.View() + "\n\n" + footerStyle.Render(hint)
	body += "\n\n" + footerStyle.Render("enter: rename  •  esc: cancel")
	return formBoxStyle.Render(titleStyle.Render("Rename") + "\n\n" + body)
}
