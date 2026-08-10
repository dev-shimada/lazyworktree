package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Worktree describes a single entry from `git worktree list --porcelain`.
type Worktree struct {
	Path           string
	Head           string
	Branch         string // short branch name, e.g. "main"; empty if detached or bare
	Bare           bool
	Detached       bool
	Locked         bool
	LockedReason   string
	Prunable       bool
	PrunableReason string
}

// ListWorktrees returns all worktrees registered on the repository containing dir.
func ListWorktrees(dir string) ([]Worktree, error) {
	out, err := run(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parseWorktreePorcelain(out), nil
}

// MainWorktreeRoot returns the path of the repository's main worktree —
// unlike RepoRoot, which returns whichever worktree currently contains dir,
// this returns the same repo-anchored path no matter which of the repo's
// worktrees dir is inside. `git worktree list` always lists the main
// worktree first, regardless of cwd.
//
// Conventions that should be anchored to the repo itself rather than to
// "whichever worktree happens to be current" (where new worktrees get
// created, herdr's worktree open/create — which reject a linked worktree's
// path outright) should use this instead of RepoRoot.
func MainWorktreeRoot(dir string) (string, error) {
	worktrees, err := ListWorktrees(dir)
	if err != nil {
		return "", err
	}
	if len(worktrees) == 0 {
		return "", fmt.Errorf("git worktree list returned no worktrees")
	}
	return worktrees[0].Path, nil
}

func parseWorktreePorcelain(out string) []Worktree {
	var worktrees []Worktree
	var cur *Worktree

	flush := func() {
		if cur != nil {
			worktrees = append(worktrees, *cur)
			cur = nil
		}
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			flush()
			continue
		}
		key, rest, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			flush()
			cur = &Worktree{Path: rest}
		case "HEAD":
			if cur != nil {
				cur.Head = rest
			}
		case "branch":
			if cur != nil {
				cur.Branch = strings.TrimPrefix(rest, "refs/heads/")
			}
		case "bare":
			if cur != nil {
				cur.Bare = true
			}
		case "detached":
			if cur != nil {
				cur.Detached = true
			}
		case "locked":
			if cur != nil {
				cur.Locked = true
				cur.LockedReason = rest
			}
		case "prunable":
			if cur != nil {
				cur.Prunable = true
				cur.PrunableReason = rest
			}
		}
	}
	flush()
	return worktrees
}

// AddOptions configures AddWorktree.
type AddOptions struct {
	// Branch is the branch to check out in the new worktree.
	Branch string
	// CreateBranch, when true, creates Branch as a new branch (git worktree add -b).
	CreateBranch bool
	// Base is the starting point for a newly created branch. Ignored unless CreateBranch is true.
	// Defaults to HEAD when empty.
	Base string
}

// AddWorktree creates a new worktree at path, rooted at the repository containing dir.
func AddWorktree(dir, path string, opts AddOptions) error {
	args := []string{"worktree", "add"}
	if opts.CreateBranch {
		args = append(args, "-b", opts.Branch, path)
		if opts.Base != "" {
			args = append(args, opts.Base)
		}
	} else {
		args = append(args, path, opts.Branch)
	}
	_, err := run(dir, args...)
	return err
}

// RemoveWorktree removes the worktree at path. If force is true, uncommitted changes are discarded.
func RemoveWorktree(dir, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	_, err := run(dir, args...)
	return err
}

// PruneWorktrees removes stale administrative files for worktrees deleted outside of git.
func PruneWorktrees(dir string) error {
	_, err := run(dir, "worktree", "prune")
	return err
}

// LockWorktree marks the worktree at path as locked, optionally with a reason.
func LockWorktree(dir, path, reason string) error {
	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, path)
	_, err := run(dir, args...)
	return err
}

// UnlockWorktree removes the locked marker from the worktree at path.
func UnlockWorktree(dir, path string) error {
	_, err := run(dir, "worktree", "unlock", path)
	return err
}

// RenameBranch renames a branch. It works regardless of which worktree (if any)
// currently has it checked out, and never touches worktree checkout directories.
func RenameBranch(dir, oldName, newName string) error {
	_, err := run(dir, "branch", "-m", oldName, newName)
	return err
}

// FetchPRRef fetches a GitHub pull request's head commit into a local branch,
// without requiring the PR's source branch to exist as a remote-tracking ref
// (works for PRs from forks).
func FetchPRRef(dir string, number int, localBranch string) error {
	refspec := fmt.Sprintf("refs/pull/%d/head:%s", number, localBranch)
	_, err := run(dir, "fetch", "origin", refspec)
	return err
}

// FetchBranch updates the remote-tracking ref for a single branch.
func FetchBranch(dir, branch string) error {
	_, err := run(dir, "fetch", "origin", branch)
	return err
}

// Branch is a local or remote-tracking branch usable as a worktree checkout target.
type Branch struct {
	Name   string
	Remote bool
}

// ListBranches returns local and remote-tracking branches, excluding those already
// checked out in a worktree.
func ListBranches(dir string) ([]Branch, error) {
	out, err := run(dir, "for-each-ref", "--format=%(refname)", "refs/heads", "refs/remotes")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	worktrees, err := ListWorktrees(dir)
	if err != nil {
		return nil, err
	}
	checkedOut := make(map[string]bool, len(worktrees))
	for _, w := range worktrees {
		if w.Branch != "" {
			checkedOut[w.Branch] = true
		}
	}

	var branches []Branch
	seen := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "refs/heads/"):
			name := strings.TrimPrefix(line, "refs/heads/")
			if !checkedOut[name] && !seen[name] {
				seen[name] = true
				branches = append(branches, Branch{Name: name})
			}
		case strings.HasPrefix(line, "refs/remotes/"):
			name := strings.TrimPrefix(line, "refs/remotes/")
			if strings.HasSuffix(name, "/HEAD") {
				continue
			}
			short := name
			if i := strings.Index(name, "/"); i >= 0 {
				short = name[i+1:]
			}
			if !checkedOut[short] && !seen[name] {
				seen[name] = true
				branches = append(branches, Branch{Name: name, Remote: true})
			}
		}
	}
	return branches, nil
}

// BranchExists reports whether name resolves to a local branch.
func BranchExists(dir, name string) bool {
	_, err := run(dir, "show-ref", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

// SanitizeBranchForPath converts a branch name into a filesystem-safe path segment.
// Slashes are preserved so branches like "feature/foo" map to nested directories.
func SanitizeBranchForPath(branch string) string {
	replacer := strings.NewReplacer("\\", "-", " ", "-")
	return replacer.Replace(branch)
}

// DefaultWorktreePath suggests a location for a new worktree of branch, nested
// under the repository root at .worktrees/<branch> (mirroring how `git worktree
// add` is conventionally invoked with a repo-relative path).
func DefaultWorktreePath(repoRoot, branch string) string {
	return filepath.Join(repoRoot, ".worktrees", SanitizeBranchForPath(branch))
}
