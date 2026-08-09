// Package config reads lazyworktree's own user-level configuration file at
// ~/.config/lazyworktree/config.toml.
package config

import (
	"bufio"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// issuesRepoRule is one [[issues_repo]] entry: repos whose "owner/repo"
// matches Match (a path.Match glob, e.g. "myorg/*") get Repos as additional
// Issues-tab sources, alongside the repo itself.
type issuesRepoRule struct {
	Match string
	Repos []string
}

// Path returns the path to lazyworktree's config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lazyworktree", "config.toml"), nil
}

// IssuesRepos returns the additional issues repos ("owner/repo") configured
// for currentRepo (also "owner/repo"), from the first matching
// [[issues_repo]] rule (rules are checked in file order). The current repo's
// own issues remain accessible regardless — this only ever adds sources, it
// never hides currentRepo's own issues. Missing or unreadable config is
// treated as "no rules" rather than an error, since this is an optional
// convenience feature.
func IssuesRepos(currentRepo string) []string {
	p, err := Path()
	if err != nil {
		return nil
	}
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	rules := scanIssuesRepoRules(f)
	for _, rule := range rules {
		if rule.Match == "" || len(rule.Repos) == 0 {
			continue
		}
		if matchGlob(rule.Match, currentRepo) {
			return rule.Repos
		}
	}
	return nil
}

// matchGlob reports whether repo ("owner/repo") matches pattern, a
// path.Match-style glob where "*" does not cross "/" (e.g. "myorg/*" matches
// "myorg/foo" but not "myorg/foo/bar").
func matchGlob(pattern, repo string) bool {
	ok, err := path.Match(pattern, repo)
	return ok && err == nil
}

// scanIssuesRepoRules does a minimal best-effort scan for `[[issues_repo]]`
// array-of-tables entries, each expected to set `match` (a string) and
// `repos` (a single-line TOML string array, e.g. `["a/b", "c/d"]`).
func scanIssuesRepoRules(r io.Reader) []issuesRepoRule {
	var rules []issuesRepoRule
	var cur *issuesRepoRule

	flush := func() {
		if cur != nil {
			rules = append(rules, *cur)
			cur = nil
		}
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch {
		case line == "" || strings.HasPrefix(line, "#"):
			continue
		case line == "[[issues_repo]]":
			flush()
			cur = &issuesRepoRule{}
		case strings.HasPrefix(line, "["):
			flush()
		case cur != nil:
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			switch strings.TrimSpace(key) {
			case "match":
				cur.Match = strings.Trim(value, `"'`)
			case "repos":
				cur.Repos = parseStringArray(value)
			}
		}
	}
	flush()
	return rules
}

// parseStringArray parses a single-line TOML string array like
// `["a/b", "c/d"]` into its elements. Multi-line arrays aren't supported —
// this is a minimal scanner for lazyworktree's own small config schema, not
// a general TOML parser.
func parseStringArray(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.Trim(strings.TrimSpace(part), `"'`)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
