package tui

import (
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
