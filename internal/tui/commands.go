package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-shimada/lazyworktree/internal/git"
	"github.com/dev-shimada/lazyworktree/internal/github"
	"github.com/dev-shimada/lazyworktree/internal/herdr"
)

type worktreesLoadedMsg struct {
	worktrees []git.Worktree
}

type issuesLoadedMsg struct {
	issues []github.Issue
}

type prsLoadedMsg struct {
	prs []github.PullRequest
}

type errMsg struct {
	err error
}

type actionDoneMsg struct {
	status string
	err    error
}

func loadWorktreesCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		wts, err := git.ListWorktrees(repoDir)
		if err != nil {
			return errMsg{err}
		}
		return worktreesLoadedMsg{worktrees: wts}
	}
}

// openOrFocusHerdr opens path as a herdr pane, focusing an already-open
// workspace for it instead of asking herdr to open it again.
func openOrFocusHerdr(repoRoot, path string) (string, error) {
	if id, ok := herdr.OpenWorkspaceForPath(repoRoot, path); ok {
		if err := herdr.FocusWorkspace(id); err != nil {
			return "", err
		}
		return "focused herdr workspace: " + path, nil
	}
	if err := herdr.OpenWorktree(repoRoot, path); err != nil {
		return "", err
	}
	return "opened in herdr: " + path, nil
}

func addWorktreeCmd(repoDir, path string, opts git.AddOptions, openHerdr bool) tea.Cmd {
	return func() tea.Msg {
		if err := git.AddWorktree(repoDir, path, opts); err != nil {
			return actionDoneMsg{err: err}
		}
		status := "created worktree at " + path
		if openHerdr {
			if s, err := openOrFocusHerdr(repoDir, path); err != nil {
				status += " (herdr open failed: " + err.Error() + ")"
			} else {
				status += ", " + s
			}
		}
		return actionDoneMsg{status: status}
	}
}

// removeWorktreeCmd removes a worktree. If herdr has an open workspace for
// path, it closes that workspace and removes the checkout in one step
// (`herdr worktree remove --workspace`); otherwise it falls back to a plain
// `git worktree remove`, so no herdr pane is left pointing at a deleted
// directory.
func removeWorktreeCmd(repoDir, path string, force bool) tea.Cmd {
	return func() tea.Msg {
		if herdr.Available() {
			if id, ok := herdr.OpenWorkspaceForPath(repoDir, path); ok {
				if err := herdr.RemoveWorkspaceWorktree(id, force); err != nil {
					return actionDoneMsg{err: err}
				}
				return actionDoneMsg{status: "removed (via herdr): " + path}
			}
		}
		if err := git.RemoveWorktree(repoDir, path, force); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: "removed " + path}
	}
}

func pruneWorktreesCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		if err := git.PruneWorktrees(repoDir); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: "pruned stale worktrees"}
	}
}

func toggleLockCmd(repoDir, path string, lock bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if lock {
			err = git.LockWorktree(repoDir, path, "")
		} else {
			err = git.UnlockWorktree(repoDir, path)
		}
		if err != nil {
			return actionDoneMsg{err: err}
		}
		if lock {
			return actionDoneMsg{status: "locked " + path}
		}
		return actionDoneMsg{status: "unlocked " + path}
	}
}

// openInHerdrCmd opens a worktree as a herdr pane. If it already has an open
// workspace, that workspace is focused instead of asking herdr to open it
// again.
func openInHerdrCmd(repoRoot, path string) tea.Cmd {
	return func() tea.Msg {
		status, err := openOrFocusHerdr(repoRoot, path)
		if err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: status}
	}
}

func renameBranchCmd(repoDir, oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		if err := git.RenameBranch(repoDir, oldName, newName); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: fmt.Sprintf("renamed branch %s -> %s", oldName, newName)}
	}
}

func renameHerdrWorkspaceCmd(label string) tea.Cmd {
	return func() tea.Msg {
		id, err := herdr.CurrentWorkspaceID()
		if err != nil {
			return actionDoneMsg{err: fmt.Errorf("resolving herdr workspace: %w", err)}
		}
		if err := herdr.RenameWorkspace(id, label); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: "renamed herdr workspace label to " + label}
	}
}

func loadIssuesCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		if !github.Available() {
			return errMsg{fmt.Errorf("gh CLI not found in PATH")}
		}
		issues, err := github.ListIssues(repoDir)
		if err != nil {
			return errMsg{err}
		}
		return issuesLoadedMsg{issues: issues}
	}
}

func loadPRsCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		if !github.Available() {
			return errMsg{fmt.Errorf("gh CLI not found in PATH")}
		}
		prs, err := github.ListPullRequests(repoDir)
		if err != nil {
			return errMsg{err}
		}
		return prsLoadedMsg{prs: prs}
	}
}

func openIssueBrowserCmd(repoDir string, number int) tea.Cmd {
	return func() tea.Msg {
		if err := github.OpenIssueInBrowser(repoDir, number); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: fmt.Sprintf("opened issue #%d in browser", number)}
	}
}

func openPRBrowserCmd(repoDir string, number int) tea.Cmd {
	return func() tea.Msg {
		if err := github.OpenPRInBrowser(repoDir, number); err != nil {
			return actionDoneMsg{err: err}
		}
		return actionDoneMsg{status: fmt.Sprintf("opened PR #%d in browser", number)}
	}
}

// existingWorktreePath returns the checkout path already registered for
// branch, if any.
func existingWorktreePath(worktrees []git.Worktree, branch string) (string, bool) {
	for _, w := range worktrees {
		if w.Branch == branch {
			return w.Path, true
		}
	}
	return "", false
}

// checkoutPRWorktreeCmd checks out a pull request's head commit into a new
// worktree (branch "pr-<number>"), fetching it directly via its PR ref so it
// works for PRs from forks. If a worktree for that branch already exists, the
// fetch/checkout is skipped and the existing one is reused.
func checkoutPRWorktreeCmd(repoRoot string, worktrees []git.Worktree, pr github.PullRequest, openHerdr bool) tea.Cmd {
	return func() tea.Msg {
		branch := fmt.Sprintf("pr-%d", pr.Number)
		path, exists := existingWorktreePath(worktrees, branch)
		if !exists {
			path = git.DefaultWorktreePath(repoRoot, branch)
			if err := git.FetchPRRef(repoRoot, pr.Number, branch); err != nil {
				return actionDoneMsg{err: err}
			}
			if err := git.AddWorktree(repoRoot, path, git.AddOptions{Branch: branch}); err != nil {
				return actionDoneMsg{err: err}
			}
		}

		status := fmt.Sprintf("checked out PR #%d at %s", pr.Number, path)
		if openHerdr {
			if s, err := openOrFocusHerdr(repoRoot, path); err != nil {
				status += " (herdr open failed: " + err.Error() + ")"
			} else {
				status += ", " + s
			}
		}
		return actionDoneMsg{status: status}
	}
}

// checkoutIssueWorktreeCmd creates a new branch ("issue-<number>") off the
// repository's default branch and checks it out into a new worktree. If a
// worktree for that branch already exists, this is skipped and the existing
// one is reused.
func checkoutIssueWorktreeCmd(repoRoot string, worktrees []git.Worktree, issue github.Issue, openHerdr bool) tea.Cmd {
	return func() tea.Msg {
		branch := fmt.Sprintf("issue-%d", issue.Number)
		path, exists := existingWorktreePath(worktrees, branch)
		if !exists {
			path = git.DefaultWorktreePath(repoRoot, branch)
			base, err := github.DefaultBranch(repoRoot)
			if err != nil {
				return actionDoneMsg{err: fmt.Errorf("resolving default branch: %w", err)}
			}
			if err := git.FetchBranch(repoRoot, base); err != nil {
				return actionDoneMsg{err: err}
			}
			opts := git.AddOptions{Branch: branch, CreateBranch: true, Base: "origin/" + base}
			if err := git.AddWorktree(repoRoot, path, opts); err != nil {
				return actionDoneMsg{err: err}
			}
		}

		status := fmt.Sprintf("checked out issue #%d at %s", issue.Number, path)
		if openHerdr {
			if s, err := openOrFocusHerdr(repoRoot, path); err != nil {
				status += " (herdr open failed: " + err.Error() + ")"
			} else {
				status += ", " + s
			}
		}
		return actionDoneMsg{status: status}
	}
}
