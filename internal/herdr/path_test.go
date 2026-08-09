package herdr

import (
	"os"
	"strings"
	"testing"
)

func TestScanWorktreesDirective(t *testing.T) {
	cases := []struct {
		name      string
		toml      string
		wantValue string
		wantOK    bool
	}{
		{
			name: "commented out, default",
			toml: `
onboarding = false
[keys]
prefix = "ctrl+a"

# [worktrees]
# directory = "~/.herdr/worktrees"

[ui]
sidebar_width = 26
`,
			wantOK: false,
		},
		{
			name: "customized",
			toml: `
[worktrees]
directory = "/custom/worktrees"

[ui]
`,
			wantValue: "/custom/worktrees",
			wantOK:    true,
		},
		{
			name: "single quotes",
			toml: `
[worktrees]
directory = '/custom/worktrees'
`,
			wantValue: "/custom/worktrees",
			wantOK:    true,
		},
		{
			name:   "no worktrees section at all",
			toml:   `[ui]` + "\n" + `sidebar_width = 26`,
			wantOK: false,
		},
		{
			name: "directory line outside worktrees section is ignored",
			toml: `
[other]
directory = "/should/not/match"

[worktrees]
`,
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value, ok := scanWorktreesDirective(strings.NewReader(tc.toml))
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && value != tc.wantValue {
				t.Errorf("value = %q, want %q", value, tc.wantValue)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory available")
	}
	got := expandHome("~/.herdr/worktrees")
	want := home + "/.herdr/worktrees"
	if got != want {
		t.Errorf("expandHome = %q, want %q", got, want)
	}

	if got := expandHome("/already/absolute"); got != "/already/absolute" {
		t.Errorf("expandHome should leave absolute paths unchanged, got %q", got)
	}
}

func TestSanitizeBranchForPath(t *testing.T) {
	cases := map[string]string{
		"feature/foo": "feature/foo",
		"main":        "main",
		"a b/c\\d":    "a-b/c-d",
	}
	for in, want := range cases {
		if got := sanitizeBranchForPath(in); got != want {
			t.Errorf("sanitizeBranchForPath(%q) = %q, want %q", in, got, want)
		}
	}
}
