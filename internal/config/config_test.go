package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanIssuesRepoRules(t *testing.T) {
	toml := `
[[issues_repo]]
match = "myorg/*"
repos = ["myorg/specs", "myorg/other-specs"]

[[issues_repo]]
match = "some-org/legacy-app"
repos = ["some-org/legacy-specs"]

[other_section]
match = "should-not-be-picked-up"
repos = ["nope"]
`
	rules := scanIssuesRepoRules(strings.NewReader(toml))
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d: %+v", len(rules), rules)
	}
	if rules[0].Match != "myorg/*" || !reflect.DeepEqual(rules[0].Repos, []string{"myorg/specs", "myorg/other-specs"}) {
		t.Errorf("unexpected first rule: %+v", rules[0])
	}
	if rules[1].Match != "some-org/legacy-app" || !reflect.DeepEqual(rules[1].Repos, []string{"some-org/legacy-specs"}) {
		t.Errorf("unexpected second rule: %+v", rules[1])
	}
}

func TestScanIssuesRepoRulesEmpty(t *testing.T) {
	if rules := scanIssuesRepoRules(strings.NewReader("")); len(rules) != 0 {
		t.Errorf("expected no rules, got %+v", rules)
	}
}

func TestParseStringArray(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{`["a/b"]`, []string{"a/b"}},
		{`["a/b", "c/d"]`, []string{"a/b", "c/d"}},
		{`['a/b', 'c/d']`, []string{"a/b", "c/d"}},
		{`[]`, nil},
		{`["a/b",   "c/d"  ]`, []string{"a/b", "c/d"}},
	}
	for _, tc := range cases {
		got := parseStringArray(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseStringArray(%q) = %#v, want %#v", tc.in, got, tc.want)
		}
	}
}

func TestMatchGlob(t *testing.T) {
	cases := []struct {
		pattern, repo string
		want          bool
	}{
		{"myorg/*", "myorg/app", true},
		{"myorg/*", "other/app", false},
		{"myorg/*", "myorg/nested/notmatched", false},
		{"exact/match", "exact/match", true},
		{"exact/match", "exact/other", false},
	}
	for _, tc := range cases {
		if got := matchGlob(tc.pattern, tc.repo); got != tc.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tc.pattern, tc.repo, got, tc.want)
		}
	}
}
