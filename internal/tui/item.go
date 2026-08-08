package tui

import (
	"fmt"

	"github.com/dev-shimada/lazyworktree/internal/git"
	"github.com/dev-shimada/lazyworktree/internal/github"
)

type worktreeItem struct {
	wt git.Worktree
	pr *github.PullRequest // matching open PR for wt.Branch, if any
}

func (i worktreeItem) Title() string {
	branch := i.wt.Branch
	switch {
	case i.wt.Bare:
		branch = "(bare)"
	case i.wt.Detached:
		branch = "(detached)"
	}
	title := branchStyle.Render(branch)
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
	return fmt.Sprintf("%s  %s", i.wt.Path, head)
}

func (i worktreeItem) FilterValue() string {
	return i.wt.Branch + " " + i.wt.Path
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

func (i issueItem) FilterValue() string {
	return i.issue.Title
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

func (i prItem) FilterValue() string {
	return i.pr.Title + " " + i.pr.HeadRefName
}
