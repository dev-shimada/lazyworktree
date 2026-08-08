package git

import "testing"

func TestParseWorktreePorcelain(t *testing.T) {
	input := `worktree /repo/main
HEAD abc123
branch refs/heads/main

worktree /repo/feature
HEAD def456
branch refs/heads/feature

worktree /repo/detached
HEAD 789abc
detached

worktree /repo/locked
HEAD 111222
branch refs/heads/locked
locked reason here

worktree /repo/prunable
HEAD 333444
branch refs/heads/gone
prunable gitdir file points to non-existent location
`

	got := parseWorktreePorcelain(input)
	if len(got) != 5 {
		t.Fatalf("expected 5 worktrees, got %d: %+v", len(got), got)
	}

	if got[0].Path != "/repo/main" || got[0].Branch != "main" || got[0].Head != "abc123" {
		t.Errorf("unexpected first worktree: %+v", got[0])
	}
	if !got[2].Detached || got[2].Branch != "" {
		t.Errorf("expected detached worktree with no branch: %+v", got[2])
	}
	if !got[3].Locked || got[3].LockedReason != "reason here" {
		t.Errorf("expected locked worktree with reason: %+v", got[3])
	}
	if !got[4].Prunable || got[4].PrunableReason != "gitdir file points to non-existent location" {
		t.Errorf("expected prunable worktree with reason: %+v", got[4])
	}
}

func TestParseWorktreePorcelainEmpty(t *testing.T) {
	if got := parseWorktreePorcelain(""); len(got) != 0 {
		t.Errorf("expected no worktrees for empty input, got %+v", got)
	}
}

func TestSanitizeBranchForPath(t *testing.T) {
	cases := map[string]string{
		"feature/foo": "feature/foo",
		"main":        "main",
		"a b/c\\d":    "a-b/c-d",
	}
	for in, want := range cases {
		if got := SanitizeBranchForPath(in); got != want {
			t.Errorf("SanitizeBranchForPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDefaultWorktreePath(t *testing.T) {
	got := DefaultWorktreePath("/home/user/myrepo", "feature/foo")
	want := "/home/user/myrepo/.worktrees/feature/foo"
	if got != want {
		t.Errorf("DefaultWorktreePath = %q, want %q", got, want)
	}
}
