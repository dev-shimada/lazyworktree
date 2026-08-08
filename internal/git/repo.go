// Package git wraps the git CLI to inspect and manage worktrees.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// run executes git with the given args in dir and returns trimmed stdout.
// If dir is empty, the current working directory is used.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// RepoRoot returns the top-level directory of the git repository containing dir.
// If dir is empty, the current working directory is used.
func RepoRoot(dir string) (string, error) {
	return run(dir, "rev-parse", "--show-toplevel")
}

// IsInsideRepo reports whether dir (or the current directory, if empty) is inside a git repository.
func IsInsideRepo(dir string) bool {
	_, err := RepoRoot(dir)
	return err == nil
}
