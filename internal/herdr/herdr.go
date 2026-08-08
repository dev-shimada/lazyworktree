// Package herdr wraps the herdr CLI's git-worktree-backed workspace commands
// (`herdr worktree ...`), so lazyworktree can hand a worktree off to herdr
// for display as a pane/popup instead of managing panes itself.
package herdr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Available reports whether the herdr binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("herdr")
	return err == nil
}

// InHerdr reports whether the current process is running inside a herdr pane
// (e.g. launched as a popup via `[[keys.command]]`).
func InHerdr() bool {
	return os.Getenv("HERDR_ENV") == "1"
}

// CurrentWorkspaceID returns the herdr workspace ID hosting the current pane.
// It first checks $HERDR_WORKSPACE_ID, which most panes get, but popup
// commands (`type = "popup"` in config.toml — how lazyworktree itself is
// normally launched) do not reliably inherit it, so this falls back to
// asking the socket API for the currently focused workspace.
func CurrentWorkspaceID() (string, error) {
	if id := os.Getenv("HERDR_WORKSPACE_ID"); id != "" {
		return id, nil
	}
	return focusedWorkspaceID()
}

func run(args ...string) error {
	_, err := runOutput(args...)
	return err
}

func runOutput(args ...string) ([]byte, error) {
	cmd := exec.Command("herdr", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("herdr %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

type workspaceListResult struct {
	Result struct {
		Workspaces []struct {
			WorkspaceID string `json:"workspace_id"`
			Label       string `json:"label"`
			Focused     bool   `json:"focused"`
		} `json:"workspaces"`
	} `json:"result"`
}

func focusedWorkspaceID() (string, error) {
	out, err := runOutput("workspace", "list")
	if err != nil {
		return "", err
	}
	var res workspaceListResult
	if err := json.Unmarshal(out, &res); err != nil {
		return "", fmt.Errorf("parsing herdr workspace list output: %w", err)
	}
	for _, w := range res.Result.Workspaces {
		if w.Focused {
			return w.WorkspaceID, nil
		}
	}
	return "", fmt.Errorf("no focused herdr workspace found")
}

// Worktree is one entry from `herdr worktree list --cwd`.
type Worktree struct {
	Branch           string `json:"branch"`
	Path             string `json:"path"`
	Label            string `json:"label"`
	IsLinkedWorktree bool   `json:"is_linked_worktree"`
	IsBare           bool   `json:"is_bare"`
	IsDetached       bool   `json:"is_detached"`
	IsPrunable       bool   `json:"is_prunable"`
	OpenWorkspaceID  string `json:"open_workspace_id"`
}

type worktreeListResult struct {
	Result struct {
		Worktrees []Worktree `json:"worktrees"`
	} `json:"result"`
}

// ListWorktrees returns herdr's view of the worktrees registered for the
// repository at cwd, including which ones already have an open workspace.
func ListWorktrees(cwd string) ([]Worktree, error) {
	out, err := runOutput("worktree", "list", "--cwd", cwd)
	if err != nil {
		return nil, err
	}
	var res worktreeListResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("parsing herdr worktree list output: %w", err)
	}
	return res.Result.Worktrees, nil
}

// OpenWorkspaceForPath returns the workspace ID already open for the given
// worktree path, and whether one was found.
func OpenWorkspaceForPath(cwd, path string) (string, bool) {
	worktrees, err := ListWorktrees(cwd)
	if err != nil {
		return "", false
	}
	for _, w := range worktrees {
		if w.Path == path && w.OpenWorkspaceID != "" {
			return w.OpenWorkspaceID, true
		}
	}
	return "", false
}

// FocusWorkspace focuses an already-open herdr workspace.
func FocusWorkspace(workspaceID string) error {
	return run("workspace", "focus", workspaceID)
}

// OpenWorktree opens an existing worktree checkout as a herdr pane, focusing it.
// cwd is the repository the worktree belongs to; path is the worktree's checkout
// path. --path and --branch are mutually exclusive in herdr's CLI, so this
// identifies the worktree by path only.
func OpenWorktree(cwd, path string) error {
	return run("worktree", "open", "--cwd", cwd, "--path", path, "--focus")
}

// RemoveWorkspaceWorktree closes the herdr workspace for a worktree and removes
// its git worktree checkout in one step.
func RemoveWorkspaceWorktree(workspaceID string, force bool) error {
	args := []string{"worktree", "remove", "--workspace", workspaceID}
	if force {
		args = append(args, "--force")
	}
	return run(args...)
}

// RenameWorkspace renames a herdr workspace's display label. It only affects
// how herdr displays the workspace — the underlying worktree checkout
// directory and its branch name are left untouched.
func RenameWorkspace(workspaceID, label string) error {
	return run("workspace", "rename", workspaceID, label)
}
