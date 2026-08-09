package herdr

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// defaultWorktreesDir mirrors herdr's own built-in default (see
// `herdr --default-config`'s commented-out `[worktrees] directory` setting).
const defaultWorktreesDir = "~/.herdr/worktrees"

// DefaultWorktreePath returns the location herdr itself would use for a new
// worktree checkout of branch in the repository at repoRoot:
// <worktrees.directory>/<repo name>/<branch>, honoring a customized
// `[worktrees] directory` in herdr's config.toml.
func DefaultWorktreePath(repoRoot, branch string) string {
	dir := expandHome(configuredWorktreesDir())
	repoName := filepath.Base(repoRoot)
	return filepath.Join(dir, repoName, sanitizeBranchForPath(branch))
}

// configuredWorktreesDir does a minimal best-effort scan of herdr's
// config.toml for `[worktrees] directory = "..."`, since herdr's CLI has no
// command to query resolved config. Falls back to herdr's documented default
// when the file is missing, unreadable, or leaves the setting commented out.
func configuredWorktreesDir() string {
	path, err := configPath()
	if err != nil {
		return defaultWorktreesDir
	}
	f, err := os.Open(path)
	if err != nil {
		return defaultWorktreesDir
	}
	defer func() { _ = f.Close() }()

	if v, ok := scanWorktreesDirective(f); ok {
		return v
	}
	return defaultWorktreesDir
}

// scanWorktreesDirective scans TOML-ish config text for an active (non-comment)
// `directory = "..."` line inside a `[worktrees]` table.
func scanWorktreesDirective(r io.Reader) (string, bool) {
	inSection := false
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		case strings.HasPrefix(line, "["):
			inSection = line == "[worktrees]"
		case inSection:
			key, value, ok := strings.Cut(line, "=")
			if ok && strings.TrimSpace(key) == "directory" {
				value = strings.Trim(strings.TrimSpace(value), `"'`)
				if value != "" {
					return value, true
				}
			}
		}
	}
	return "", false
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "herdr", "config.toml"), nil
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}

// sanitizeBranchForPath mirrors internal/git's helper: slashes are preserved
// (branches like "feature/foo" become nested directories), but characters
// that can't appear in a path segment are replaced.
func sanitizeBranchForPath(branch string) string {
	replacer := strings.NewReplacer("\\", "-", " ", "-")
	return replacer.Replace(branch)
}
