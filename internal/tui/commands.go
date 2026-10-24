package tui

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dev-shimada/lazyworktree/internal/config"
	"github.com/dev-shimada/lazyworktree/internal/git"
	"github.com/dev-shimada/lazyworktree/internal/github"
	"github.com/dev-shimada/lazyworktree/internal/herdr"
)

type worktreesLoadedMsg struct {
	worktrees []git.Worktree
}

type issuesLoadedMsg struct {
	issues []github.Issue
	// sources are the available issue repos: any configured "owner/repo"
	// alternates first, then "" (the current repo, resolved via cwd) last —
	// always present, so it's reachable via 's' no matter what's configured.
	sources   []string
	sourceIdx int
	mine      bool
	query     string
	gen       int
}

type prsLoadedMsg struct {
	prs   []github.PullRequest
	mine  bool
	query string
	gen   int
}

type herdrLabelsLoadedMsg struct {
	labelByPath map[string]string
}

type worktreeRefsLoadedMsg struct {
	refByPath map[string]github.Ref
}

type errMsg struct {
	err error
	// tab and gen identify which Issues/PRs load this error belongs to, so a
	// superseded load's error can be dropped; the zero tab (Worktrees) is
	// never checked.
	tab tab
	gen int
}

type actionDoneMsg struct {
	status string
	err    error
	// quit, when true and err is nil, exits the program after showing status
	// — used for actions that hand off control elsewhere (a new worktree to
	// cd into, a herdr pane taking over), as opposed to in-app list actions
	// like delete/lock/rename that the user stays around to keep using.
	quit bool
	// cdPath, when quit is set and non-empty, becomes Model.SelectedPath so
	// the shell wrapper cds into it after exit.
	cdPath string
}

func loadWorktreesCmd(repoDir string) tea.Cmd {
	return func() tea.Msg {
		wts, err := git.ListWorktrees(repoDir)
		if err != nil {
			return errMsg{err: err}
		}
		return worktreesLoadedMsg{worktrees: wts}
	}
}

// loadHerdrLabelsCmd resolves each worktree's live herdr workspace label
// (keyed by checkout path), for worktrees that currently have one open.
// Errors are swallowed to an empty map: label display is a nice-to-have and
// shouldn't block the worktree list itself.
func loadHerdrLabelsCmd(repoRoot string) tea.Cmd {
	return func() tea.Msg {
		labelByPath := map[string]string{}
		worktrees, err := herdr.ListWorktrees(repoRoot)
		if err != nil {
			return herdrLabelsLoadedMsg{labelByPath: labelByPath}
		}
		labelsByID, err := herdr.WorkspaceLabels()
		if err != nil {
			return herdrLabelsLoadedMsg{labelByPath: labelByPath}
		}
		for _, w := range worktrees {
			if w.OpenWorkspaceID == "" {
				continue
			}
			if label, ok := labelsByID[w.OpenWorkspaceID]; ok && label != "" {
				labelByPath[w.Path] = label
			}
		}
		return herdrLabelsLoadedMsg{labelByPath: labelByPath}
	}
}

// branchRefPattern matches the "pr-<number>" / "issue-<number>" branch names
// lazyworktree itself creates when checking out a PR or issue (see
// checkoutPRWorktreeCmd / checkoutIssueWorktreeCmd) — these carry no hint of
// what they're about beyond the number.
var branchRefPattern = regexp.MustCompile(`^(pr|issue)-(\d+)$`)

// loadWorktreeRefsCmd resolves, for every worktree, the issue or PR its
// branch tracks and that item's state (open / closed / merged), keyed by
// checkout path. pr-N / issue-N branches are looked up by number; other
// branches are matched by head branch name against PRs in any state. Each
// lookup is a separate `gh` call, run concurrently (bounded); failures are
// swallowed since this is display-only, not worth blocking or erroring the
// list over.
func loadWorktreeRefsCmd(repoRoot string, worktrees []git.Worktree) tea.Cmd {
	return func() tea.Msg {
		refByPath := map[string]github.Ref{}
		if !github.Available() {
			return worktreeRefsLoadedMsg{refByPath: refByPath}
		}

		var mu sync.Mutex
		var wg sync.WaitGroup
		sem := make(chan struct{}, 8)
		for _, w := range worktrees {
			if w.Branch == "" {
				continue
			}
			wg.Add(1)
			go func(path, branch string) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				ref, ok := lookupBranchRef(repoRoot, branch)
				if !ok {
					return
				}
				mu.Lock()
				refByPath[path] = ref
				mu.Unlock()
			}(w.Path, w.Branch)
		}
		wg.Wait()
		return worktreeRefsLoadedMsg{refByPath: refByPath}
	}
}

func lookupBranchRef(repoRoot, branch string) (github.Ref, bool) {
	if m := branchRefPattern.FindStringSubmatch(branch); m != nil {
		number, err := strconv.Atoi(m[2])
		if err != nil {
			return github.Ref{}, false
		}
		var ref github.Ref
		if m[1] == "pr" {
			ref, err = github.PRRef(repoRoot, number)
		} else {
			ref, err = github.IssueRef(repoRoot, number)
		}
		return ref, err == nil
	}
	prs, err := github.ListPullRequestsForBranch(repoRoot, branch)
	if err != nil {
		return github.Ref{}, false
	}
	pr, ok := prForBranch(prs, branch)
	if !ok {
		return github.Ref{}, false
	}
	return github.Ref{Kind: "pr", Number: pr.Number, Title: pr.Title, State: pr.State, IsDraft: pr.IsDraft}, true
}

// prForBranch picks the PR whose head is branch, preferring an open one
// (a branch can be reused after its earlier PR was closed) and ignoring
// fork PRs, whose head names are unrelated to local branches.
func prForBranch(prs []github.PullRequest, branch string) (github.PullRequest, bool) {
	var found github.PullRequest
	ok := false
	for _, pr := range prs {
		if pr.HeadRefName != branch || pr.IsCrossRepository {
			continue
		}
		if pr.State == "OPEN" {
			return pr, true
		}
		if !ok {
			found, ok = pr, true
		}
	}
	return found, ok
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
		return actionDoneMsg{status: status, quit: true, cdPath: path}
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
		return actionDoneMsg{status: status, quit: true}
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

// recentClosedLimit caps how many closed items are appended after the open
// ones in the default (non-search) list, so recently finished work stays
// visible without crowding anything out.
const recentClosedLimit = 20

// searchLimit is how many open matches a server-side search returns.
const searchLimit = 100

// loadIssuesCmd resolves the available issue sources for repoDir — any
// configured alternates first, then the current repo (last, but always
// present) — and loads issues from the source at sourceIdx (clamped if it's
// now out of range, e.g. because the config changed since the last load).
// So the default (index 0) is a configured issues_repo when one is set, and
// the current repo otherwise; 's' cycles the rest, ending back at "own
// repo" before wrapping. mine restricts to issues assigned to the
// authenticated user. A non-empty query is searched server-side across all
// open issues; without one, recently closed issues follow the open ones.
func loadIssuesCmd(repoDir string, sourceIdx int, mine bool, query string, gen int) tea.Cmd {
	return func() tea.Msg {
		if !github.Available() {
			return errMsg{err: fmt.Errorf("gh CLI not found in PATH"), tab: tabIssues, gen: gen}
		}

		var sources []string
		if currentRepo, err := github.CurrentRepo(repoDir); err == nil {
			for _, repo := range config.IssuesRepos(currentRepo) {
				if repo != "" && repo != currentRepo && !containsStr(sources, repo) {
					sources = append(sources, repo)
				}
			}
		}
		sources = append(sources, "") // current repo: always present, always last
		if sourceIdx < 0 || sourceIdx >= len(sources) {
			sourceIdx = 0
		}
		source := sources[sourceIdx]

		openOpts := github.ListOpts{Mine: mine, Query: query}
		if query != "" {
			openOpts.Limit = searchLimit
		}
		var closed []github.Issue
		var wg sync.WaitGroup
		if query == "" {
			wg.Go(func() {
				closed, _ = github.ListIssues(repoDir, source, github.ListOpts{State: "closed", Mine: mine, Limit: recentClosedLimit})
			})
		}
		issues, err := github.ListIssues(repoDir, source, openOpts)
		wg.Wait()
		if err != nil {
			return errMsg{err: err, tab: tabIssues, gen: gen}
		}
		issues = append(issues, closed...)
		return issuesLoadedMsg{issues: issues, sources: sources, sourceIdx: sourceIdx, mine: mine, query: query, gen: gen}
	}
}

func containsStr(list []string, s string) bool {
	return slices.Contains(list, s)
}

// loadPRsCmd is the pull-request counterpart of loadIssuesCmd: a non-empty
// query searches all open PRs server-side; otherwise recently closed and
// merged PRs follow the open ones.
func loadPRsCmd(repoDir string, mine bool, query string, gen int) tea.Cmd {
	return func() tea.Msg {
		if !github.Available() {
			return errMsg{err: fmt.Errorf("gh CLI not found in PATH"), tab: tabPRs, gen: gen}
		}
		openOpts := github.ListOpts{Mine: mine, Query: query}
		if query != "" {
			openOpts.Limit = searchLimit
		}
		var closed []github.PullRequest
		var wg sync.WaitGroup
		if query == "" {
			wg.Go(func() {
				closed, _ = github.ListPullRequests(repoDir, github.ListOpts{State: "closed", Mine: mine, Limit: recentClosedLimit})
			})
		}
		prs, err := github.ListPullRequests(repoDir, openOpts)
		wg.Wait()
		if err != nil {
			return errMsg{err: err, tab: tabPRs, gen: gen}
		}
		prs = append(prs, closed...)
		return prsLoadedMsg{prs: prs, mine: mine, query: query, gen: gen}
	}
}

// openWorktreeBrowserCmd opens the issue or PR a worktree's branch tracks.
// If no tracked item is known yet, it asks gh for the branch's PR directly.
func openWorktreeBrowserCmd(repoRoot, branch string, ref *github.Ref) tea.Cmd {
	return func() tea.Msg {
		switch {
		case ref != nil && ref.Kind == "issue":
			if err := github.OpenIssueInBrowser(repoRoot, "", ref.Number); err != nil {
				return actionDoneMsg{err: err}
			}
			return actionDoneMsg{status: fmt.Sprintf("opened issue #%d in browser", ref.Number)}
		case ref != nil:
			if err := github.OpenPRInBrowser(repoRoot, ref.Number); err != nil {
				return actionDoneMsg{err: err}
			}
			return actionDoneMsg{status: fmt.Sprintf("opened PR #%d in browser", ref.Number)}
		}
		if err := github.OpenBranchPRInBrowser(repoRoot, branch); err != nil {
			return actionDoneMsg{err: fmt.Errorf("no issue or PR found for branch %s: %w", branch, err)}
		}
		return actionDoneMsg{status: "opened PR for " + branch + " in browser"}
	}
}

func openIssueBrowserCmd(repoDir, repoOverride string, number int) tea.Cmd {
	return func() tea.Msg {
		if err := github.OpenIssueInBrowser(repoDir, repoOverride, number); err != nil {
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

// defaultWorktreePath picks herdr's configured worktree location when
// herdrMode is set (running as a herdr popup), otherwise lazyworktree's own
// repo-relative convention.
func defaultWorktreePath(herdrMode bool, repoRoot, branch string) string {
	if herdrMode {
		return herdr.DefaultWorktreePath(repoRoot, branch)
	}
	return git.DefaultWorktreePath(repoRoot, branch)
}

// checkoutPRWorktreeCmd checks out a pull request's head commit into a new
// worktree (branch "pr-<number>"), fetching it directly via its PR ref so it
// works for PRs from forks. If a worktree for that branch already exists, the
// fetch/checkout is skipped and the existing one is reused.
func checkoutPRWorktreeCmd(repoRoot string, worktrees []git.Worktree, pr github.PullRequest, herdrMode, openHerdr bool) tea.Cmd {
	return func() tea.Msg {
		branch := fmt.Sprintf("pr-%d", pr.Number)
		path, exists := existingWorktreePath(worktrees, branch)
		if !exists {
			path = defaultWorktreePath(herdrMode, repoRoot, branch)
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
		return actionDoneMsg{status: status, quit: true, cdPath: path}
	}
}

// checkoutIssueWorktreeCmd creates a new branch ("issue-<number>") off the
// repository's default branch and checks it out into a new worktree. If a
// worktree for that branch already exists, this is skipped and the existing
// one is reused.
func checkoutIssueWorktreeCmd(repoRoot string, worktrees []git.Worktree, issue github.Issue, herdrMode, openHerdr bool) tea.Cmd {
	return func() tea.Msg {
		branch := fmt.Sprintf("issue-%d", issue.Number)
		path, exists := existingWorktreePath(worktrees, branch)
		if !exists {
			path = defaultWorktreePath(herdrMode, repoRoot, branch)
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
		return actionDoneMsg{status: status, quit: true, cdPath: path}
	}
}
