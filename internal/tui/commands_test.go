package tui

import (
	"errors"
	"testing"

	"github.com/dev-shimada/lazyworktree/internal/github"
)

func TestPRForBranch(t *testing.T) {
	prs := []github.PullRequest{
		{Number: 5, HeadRefName: "main", State: "CLOSED", IsCrossRepository: true},
		{Number: 4, HeadRefName: "feat", State: "MERGED"},
		{Number: 3, HeadRefName: "feat", State: "OPEN"},
		{Number: 2, HeadRefName: "old", State: "CLOSED"},
	}

	if pr, ok := prForBranch(prs, "feat"); !ok || pr.Number != 3 {
		t.Errorf("feat: want open PR #3, got %+v ok=%v", pr, ok)
	}
	if pr, ok := prForBranch(prs, "old"); !ok || pr.Number != 2 {
		t.Errorf("old: want closed PR #2, got %+v ok=%v", pr, ok)
	}
	if _, ok := prForBranch(prs, "main"); ok {
		t.Errorf("main: fork PR must not match a local branch")
	}
	if _, ok := prForBranch(prs, "none"); ok {
		t.Errorf("none: unexpected match")
	}
}

func TestStaleLoadResponsesAreDropped(t *testing.T) {
	m, err := New(".")
	if err != nil {
		t.Skip("not in a git repository")
	}
	m.issueGen = 2
	m.prGen = 2

	next, _ := m.Update(issuesLoadedMsg{gen: 1, query: "stale", issues: []github.Issue{{Number: 1}}})
	got := next.(Model)
	if got.issuesLoaded || got.issueQuery != "" || len(got.issueList.Items()) != 0 {
		t.Errorf("stale issues response was applied: %+v", got.issueQuery)
	}
	next, _ = m.Update(issuesLoadedMsg{gen: 2, query: "fresh", issues: []github.Issue{{Number: 2}}})
	got = next.(Model)
	if !got.issuesLoaded || got.issueQuery != "fresh" || len(got.issueList.Items()) != 1 {
		t.Errorf("current issues response was dropped")
	}

	next, _ = m.Update(prsLoadedMsg{gen: 1, query: "stale", prs: []github.PullRequest{{Number: 1}}})
	if got = next.(Model); got.prsLoaded || got.prQuery != "" {
		t.Errorf("stale PRs response was applied")
	}

	next, _ = m.Update(errMsg{err: errors.New("boom"), tab: tabIssues, gen: 1})
	if got = next.(Model); got.status != "" {
		t.Errorf("stale error was shown: %q", got.status)
	}
	next, _ = m.Update(errMsg{err: errors.New("boom"), tab: tabIssues, gen: 2})
	if got = next.(Model); got.status != "boom" {
		t.Errorf("current error was dropped")
	}
}
