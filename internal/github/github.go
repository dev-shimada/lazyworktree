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

// ListIssues returns open issues for the repository containing dir. If
// repoOverride ("owner/repo") is non-empty, issues are read from that repo
// instead — for projects that track issues in a separate repository. If
// mine is true, only issues assigned to the authenticated user are returned.
func ListIssues(dir, repoOverride string, mine bool) ([]Issue, error) {
	args := []string{"issue", "list",
		"--state", "open",
		"--json", "number,title,author,url,updatedAt,labels",
		"--limit", fmt.Sprint(listLimit),
	}
	if repoOverride != "" {
		args = append(args, "--repo", repoOverride)
	}
	if mine {
		args = append(args, "--assignee", "@me")
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

// ListPullRequests returns open pull requests for the repository containing
// dir. If mine is true, only pull requests assigned to the authenticated
// user or with their review requested are returned.
func ListPullRequests(dir string, mine bool) ([]PullRequest, error) {
	args := []string{"pr", "list",
		"--json", "number,title,author,headRefName,url,isDraft,updatedAt",
		"--limit", fmt.Sprint(listLimit),
	}
	if mine {
		// --assignee alone misses PRs where you're a requested reviewer but
		// not the assignee, which is the common case; --search lets us
		// combine both. is:open replaces --state open here since gh treats
		// --search as its own query.
		args = append(args, "--search", "is:open (assignee:@me OR review-requested:@me)")
	} else {
		args = append(args, "--state", "open")
	}
	out, err := run(dir, args...)
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

// CurrentRepo returns the "owner/repo" identity of the repository at dir.
func CurrentRepo(dir string) (string, error) {
	out, err := run(dir, "repo", "view", "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
