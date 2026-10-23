package controllers

import (
	goContext "context"
	"errors"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// reviewThreads lists a pull request's review threads, unresolved ones first,
// to reply to them and resolve them.
func (self *PullRequestsController) reviewThreads(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	return self.c.Helpers().GithubPullRequests.WithDetails(pr, func(loaded *helpers.GithubPullRequestDetails) error {
		threads := loaded.Details.ReviewThreads
		if len(threads) == 0 {
			return errors.New(tr.NoReviewThreads)
		}
		ordered := append(loaded.Details.UnresolvedThreads(), lo.Filter(threads, func(thread *models.GithubReviewThread, _ int) bool {
			return thread.IsResolved
		})...)

		items := lo.Map(ordered, func(thread *models.GithubReviewThread, _ int) *types.MenuItem {
			state := lo.Ternary(thread.IsResolved, style.FgGreen.Sprint("✓"), style.FgYellow.Sprint("●"))
			line := lo.Ternary(thread.Line != 0, thread.Line, thread.OriginalLine)
			summary := ""
			if len(thread.Comments) > 0 {
				first := thread.Comments[0]
				summary = style.FgMagenta.Sprint(first.Author) + " " +
					utils.TruncateWithEllipsis(strings.Join(strings.Fields(first.Body), " "), 60)
			}
			return &types.MenuItem{
				LabelColumns: []string{state + " " + style.FgYellow.Sprintf("%s:%d", thread.Path, line), summary},
				Tooltip:      presentation.FormatReviewThread(self.c.Tr, thread, true),
				OpensMenu:    true,
				OnPress:      func() error { return self.reviewThreadMenu(pr, loaded.Details, thread) },
			}
		})
		return self.c.Menu(types.CreateMenuOptions{
			Title: utils.ResolvePlaceholderString(tr.ReviewThreadsTitle, prPlaceholders(pr)),
			Items: items,
		})
	})
}

func (self *PullRequestsController) reviewThreadMenu(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails, thread *models.GithubReviewThread) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	line := lo.Ternary(thread.Line != 0, thread.Line, thread.OriginalLine)
	location := fmt.Sprintf("%s:%d", thread.Path, line)
	pendingId := ""
	if details.PendingReview != nil {
		pendingId = details.PendingReview.NodeId
	}

	items := []*types.MenuItem{
		{
			Label: tr.Reply,
			OnPress: func() error {
				return helper.EditText(helpers.EditTextOpts{
					Title: utils.ResolvePlaceholderString(tr.ReplyTitle, map[string]string{"location": location}),
				}, func(body string) error {
					return helper.RunAction(pr, tr.Commenting, lo.Ternary(pendingId != "", tr.ReplyAddedToPendingReview, tr.ReplyAdded),
						func(ctx goContext.Context, client *github_commands.Client) error {
							return client.ReplyToReviewThread(ctx, thread.NodeId, pendingId, body)
						})
				})
			},
		},
	}
	if thread.IsResolved {
		items = append(items, &types.MenuItem{
			Label: tr.Unresolve,
			OnPress: func() error {
				return helper.RunAction(pr, tr.Updating, "", func(ctx goContext.Context, client *github_commands.Client) error {
					return client.UnresolveReviewThread(ctx, thread.NodeId)
				})
			},
		})
	} else {
		items = append(items, &types.MenuItem{
			Label: tr.Resolve,
			OnPress: func() error {
				return helper.RunAction(pr, tr.Updating, utils.ResolvePlaceholderString(tr.ThreadResolved, map[string]string{"location": location}),
					func(ctx goContext.Context, client *github_commands.Client) error {
						return client.ResolveReviewThread(ctx, thread.NodeId)
					})
			},
		})
	}
	if len(thread.Comments) > 0 && thread.Comments[0].Url != "" {
		items = append(items, &types.MenuItem{
			Label:   tr.OpenThreadInBrowser,
			OnPress: func() error { return self.c.OS().OpenLink(thread.Comments[0].Url) },
		})
	}
	return self.c.Menu(types.CreateMenuOptions{Title: location, Items: items})
}
