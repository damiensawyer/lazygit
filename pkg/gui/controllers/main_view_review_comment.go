package controllers

import (
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// Commenting on the selected lines of a pull request's diff in the focused main
// view, adding the comment to the viewer's pending review, as GitHub's web UI
// does with "Start a review".

// reviewCommentDisabledReason disables commenting where the pull request has no
// view of what is on screen, for the same reasons as opening the selected line
// in the pull request; and where the selection isn't on lines of one file.
func (self *MainViewController) reviewCommentDisabledReason() *types.DisabledReason {
	if reason := self.diffSelectionDisabledReason(); reason != nil {
		return reason
	}
	if self.c.Modes().Diffing.Active() {
		return &types.DisabledReason{Text: self.c.Tr.NotAvailableInDiffingMode}
	}
	if self.c.Helpers().DiffLine.ShowsCustomPatch(self.context.GetView()) {
		return &types.DisabledReason{Text: self.c.Tr.NotAvailableForCustomPatch}
	}
	diff := self.pullRequestDiff()
	if diff.PullRequestNumber == 0 {
		if diff.SpansBranches {
			return &types.DisabledReason{Text: self.c.Tr.CommitsInSeveralPullRequests}
		}
		if reason := self.c.Helpers().Host.NoPullRequestDisabledReason(diff.Branch); reason != nil {
			return reason
		}
		if reason := self.commitsOutsidePullRequestDisabledReason(diff.Commits); reason != nil {
			return reason
		}
	}
	if diff.BaseHash != "" {
		return &types.DisabledReason{Text: self.c.Tr.PullRequests.CommentNeedsWholePullRequest}
	}
	if _, err := self.reviewCommentLocation(); err != nil {
		return &types.DisabledReason{Text: err.Error()}
	}
	return nil
}

// reviewCommentLocation returns where on the diff the selected lines are, in the
// terms GitHub's review comments use.
func (self *MainViewController) reviewCommentLocation() (github_commands.ReviewComment, error) {
	tr := self.c.Tr.PullRequests
	view := self.context.GetView()
	start, end := view.SelectedLineRange()
	first, okFirst := self.c.Helpers().DiffLine.GetDiffLineInfo(view, start)
	last, okLast := self.c.Helpers().DiffLine.GetDiffLineInfo(view, end)
	if !okFirst || !okLast || first.Type == types.DiffLineHunkHeader || last.Type == types.DiffLineHunkHeader {
		return github_commands.ReviewComment{}, errors.New(tr.CommentNeedsDiffLines)
	}
	if first.Path != last.Path {
		return github_commands.ReviewComment{}, errors.New(tr.CommentNeedsOneFile)
	}
	path := repoRelativePath(self.c.Git().RepoPaths.WorktreePath(), first.Path)
	if path == "" {
		return github_commands.ReviewComment{}, diffLinesFileNotFoundError(self.c, first.Path)
	}

	side := func(info types.DiffLineInfo) (string, int) {
		if info.Type == types.DiffLineDeleted {
			return "LEFT", info.OldLine
		}
		return "RIGHT", info.NewLine
	}
	startSide, startLine := side(first)
	endSide, endLine := side(last)
	return github_commands.ReviewComment{
		Path:      path,
		Side:      endSide,
		Line:      endLine,
		StartSide: startSide,
		StartLine: startLine,
	}, nil
}

func (self *MainViewController) addReviewComment() error {
	diff := self.pullRequestDiff()
	number := diff.PullRequestNumber
	if number == 0 {
		pr, ok := self.c.Helpers().Host.PullRequestForBranch(diff.Branch)
		if !ok {
			return errors.New(self.c.Tr.NoPullRequestForBranch)
		}
		number = pr.Number
	}
	if len(diff.Commits) == 0 {
		return nil
	}
	location, err := self.reviewCommentLocation()
	if err != nil {
		return err
	}

	lines := fmt.Sprint(location.Line)
	if location.StartLine != location.Line {
		lines = fmt.Sprintf("%d-%d", location.StartLine, location.Line)
	}
	return self.c.Helpers().GithubPullRequests.AddReviewComment(number, diff.Commits[0].Hash(), location, lines)
}
