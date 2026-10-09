// Package github provides read-only access to issues and pull requests via
// the gh CLI, reusing its existing authentication (gh auth login / GH_TOKEN).
package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Available reports whether the gh CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// Author identifies the user or bot that created an issue or pull request.
type Author struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

// Label is an issue label.
type Label struct {
	Name string `json:"name"`
}

// Issue is a GitHub issue, as reported by `gh issue list --json ...`.
type Issue struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	Author    Author  `json:"author"`
	URL       string  `json:"url"`
	UpdatedAt string  `json:"updatedAt"`
	Labels    []Label `json:"labels"`
	State     string  `json:"state"` // OPEN or CLOSED
}

// PullRequest is a GitHub pull request, as reported by `gh pr list --json ...`.
type PullRequest struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Author      Author `json:"author"`
	HeadRefName string `json:"headRefName"`
	URL         string `json:"url"`
	IsDraft     bool   `json:"isDraft"`
	UpdatedAt   string `json:"updatedAt"`
	State       string `json:"state"` // OPEN, CLOSED, or MERGED
	// IsCrossRepository is true for PRs opened from a fork, whose head branch
	// name says nothing about any local branch of the same name.
	IsCrossRepository bool `json:"isCrossRepository"`
}

// Ref describes the issue or pull request a worktree's branch tracks, in any
// state (open, closed, merged).
type Ref struct {
	Kind    string // "pr" or "issue"
	Number  int
	Title   string
	State   string // OPEN, CLOSED, or MERGED
	IsDraft bool
}

// ListOpts narrows ListIssues / ListPullRequests.
type ListOpts struct {
	// State is "open" (default) or "closed" (for PRs this includes merged).
	State string
	// Mine restricts to items assigned to the authenticated user (for PRs,
	// also those with their review requested).
	Mine bool
	// Query is a GitHub search expression applied server-side across every
	// matching item, not just the first page.
	Query string
	Limit int
}

func (o ListOpts) state() string {
	if o.State == "" {
		return "open"
	}
	return o.State
}

func (o ListOpts) limit() int {
	if o.Limit > 0 {
		return o.Limit
	}
	return listLimit
}

const listLimit = 50

func run(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("gh %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.Bytes(), nil
}

// ListIssues returns issues for the repository containing dir. If
// repoOverride ("owner/repo") is non-empty, issues are read from that repo
// instead — for projects that track issues in a separate repository.
func ListIssues(dir, repoOverride string, opts ListOpts) ([]Issue, error) {
	args := []string{"issue", "list",
		"--state", opts.state(),
		"--json", "number,title,author,url,updatedAt,labels,state",
		"--limit", fmt.Sprint(opts.limit()),
	}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	if opts.Mine {
		args = append(args, "--assignee", "@me")
	}
	if opts.Query != "" {
		args = append(args, "--search", opts.Query)
	}
	out, err := run(dir, args...)
	if err != nil {
		return nil, err
	}
	return parseIssues(out)
}

func parseIssues(data []byte) ([]Issue, error) {
	var issues []Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("parsing gh issue list output: %w", err)
	}
	return issues, nil
}

// ListPullRequests returns pull requests for the repository containing dir.
func ListPullRequests(dir string, opts ListOpts) ([]PullRequest, error) {
	args := []string{"pr", "list",
		"--json", prListFields,
		"--limit", fmt.Sprint(opts.limit()),
	}
	if opts.Mine {
		// --assignee alone misses PRs where you're a requested reviewer but
		// not the assignee, which is the common case; --search lets us
		// combine both. is:<state> replaces --state here since gh treats
		// --search as its own query.
		search := "is:" + opts.state() + " (assignee:@me OR review-requested:@me)"
		if opts.Query != "" {
			search = opts.Query + " " + search
		}
		args = append(args, "--search", search)
	} else {
		args = append(args, "--state", opts.state())
		if opts.Query != "" {
			args = append(args, "--search", opts.Query)
		}
	}
	out, err := run(dir, args...)
	if err != nil {
		return nil, err
	}
	return parsePullRequests(out)
}

const prListFields = "number,title,author,headRefName,url,isDraft,updatedAt,state,isCrossRepository"

// ListRecentPullRequests returns the most recent pull requests in any state,
// for matching local branches to their PR once it has been closed or merged.
func ListRecentPullRequests(dir string) ([]PullRequest, error) {
	out, err := run(dir, "pr", "list", "--state", "all", "--json", prListFields, "--limit", "100")
	if err != nil {
		return nil, err
	}
	return parsePullRequests(out)
}

func parsePullRequests(data []byte) ([]PullRequest, error) {
	var prs []PullRequest
	if err := json.Unmarshal(data, &prs); err != nil {
		return nil, fmt.Errorf("parsing gh pr list output: %w", err)
	}
	return prs, nil
}

// OpenIssueInBrowser opens the given issue number in the default web browser.
// repoOverride works the same as in ListIssues.
func OpenIssueInBrowser(dir, repoOverride string, number int) error {
	args := []string{"issue", "view", fmt.Sprint(number), "--web"}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	_, err := run(dir, args...)
	return err
}

// OpenPRInBrowser opens the given pull request number in the default web browser.
func OpenPRInBrowser(dir string, number int) error {
	_, err := run(dir, "pr", "view", fmt.Sprint(number), "--web")
	return err
}

// DefaultBranch returns the repository's default branch name (e.g. "main").
func DefaultBranch(dir string) (string, error) {
	out, err := run(dir, "repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// PRRef returns the title and state of the given pull request number.
func PRRef(dir string, number int) (Ref, error) {
	out, err := run(dir, "pr", "view", fmt.Sprint(number), "--json", "title,state,isDraft")
	if err != nil {
		return Ref{}, err
	}
	var v struct {
		Title   string `json:"title"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return Ref{}, fmt.Errorf("parsing gh pr view output: %w", err)
	}
	return Ref{Kind: "pr", Number: number, Title: v.Title, State: v.State, IsDraft: v.IsDraft}, nil
}

// IssueRef returns the title and state of the given issue number.
func IssueRef(dir string, number int) (Ref, error) {
	out, err := run(dir, "issue", "view", fmt.Sprint(number), "--json", "title,state")
	if err != nil {
		return Ref{}, err
	}
	var v struct {
		Title string `json:"title"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return Ref{}, fmt.Errorf("parsing gh issue view output: %w", err)
	}
	return Ref{Kind: "issue", Number: number, Title: v.Title, State: v.State}, nil
}

// OpenBranchPRInBrowser opens the pull request for the given branch name,
// whatever its state, in the default web browser.
func OpenBranchPRInBrowser(dir, branch string) error {
	_, err := run(dir, "pr", "view", branch, "--web")
	return err
}

// CurrentRepo returns the "owner/repo" identity of the repository at dir.
func CurrentRepo(dir string) (string, error) {
	out, err := run(dir, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
