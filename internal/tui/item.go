package tui

import (
	"fmt"
	"strings"

	"github.com/dev-shimada/lazyworktree/internal/git"
	"github.com/dev-shimada/lazyworktree/internal/github"
)

type worktreeItem struct {
	wt         git.Worktree
	pr         *github.PullRequest // matching open PR for wt.Branch, if any
	herdrLabel string              // live herdr workspace label, when one is open for wt.Path
	ghTitle    string              // GitHub PR/issue title, for pr-N/issue-N branches
}

func (i worktreeItem) Title() string {
	name := i.wt.Branch
	switch {
	case i.wt.Bare:
		name = "(bare)"
	case i.wt.Detached:
		name = "(detached)"
	}
	if i.herdrLabel != "" {
		name = i.herdrLabel
	}
	title := branchStyle.Render(name)
	if i.ghTitle != "" {
		title += "  " + i.ghTitle
	}
	if i.wt.Locked {
		title += " " + lockedBadgeStyle.Render("[locked]")
	}
	if i.wt.Prunable {
		title += " " + prunableBadgeStyle.Render("[prunable]")
	}
	if i.pr != nil {
		badge := fmt.Sprintf("[PR #%d]", i.pr.Number)
		if i.pr.IsDraft {
			badge = fmt.Sprintf("[PR #%d draft]", i.pr.Number)
		}
		title += " " + prBadgeStyle.Render(badge)
	}
	return title
}

func (i worktreeItem) Description() string {
	head := i.wt.Head
	if len(head) > 8 {
		head = head[:8]
	}
	if i.herdrLabel != "" {
		return fmt.Sprintf("%s  %s  [%s]", i.wt.Path, head, i.wt.Branch)
	}
	return fmt.Sprintf("%s  %s", i.wt.Path, head)
}

func (i worktreeItem) FilterValue() string {
	return i.wt.Branch + " " + i.wt.Path + " " + i.ghTitle
}

type issueItem struct {
	issue github.Issue
}

func (i issueItem) Title() string {
	return fmt.Sprintf("#%d %s", i.issue.Number, i.issue.Title)
}

func (i issueItem) Description() string {
	date := i.issue.UpdatedAt
	if len(date) > 10 {
		date = date[:10]
	}
	return fmt.Sprintf("@%s  %s", i.issue.Author.Login, date)
}

// FilterValue includes the number, author, and labels alongside the title
// so "/" (fuzzy) search can match on any of them, not just the title text.
func (i issueItem) FilterValue() string {
	labels := make([]string, len(i.issue.Labels))
	for j, l := range i.issue.Labels {
		labels[j] = l.Name
	}
	return fmt.Sprintf("#%d %s %s %s", i.issue.Number, i.issue.Title, i.issue.Author.Login, strings.Join(labels, " "))
}

type prItem struct {
	pr github.PullRequest
}

func (i prItem) Title() string {
	title := fmt.Sprintf("#%d %s", i.pr.Number, i.pr.Title)
	if i.pr.IsDraft {
		title += " " + lockedBadgeStyle.Render("[draft]")
	}
	return title
}

func (i prItem) Description() string {
	date := i.pr.UpdatedAt
	if len(date) > 10 {
		date = date[:10]
	}
	return fmt.Sprintf("@%s  %s  %s", i.pr.Author.Login, i.pr.HeadRefName, date)
}

// FilterValue includes the number and author alongside the title and branch
// so "/" (fuzzy) search can match on any of them, not just the title text.
func (i prItem) FilterValue() string {
	return fmt.Sprintf("#%d %s %s %s", i.pr.Number, i.pr.Title, i.pr.Author.Login, i.pr.HeadRefName)
}
