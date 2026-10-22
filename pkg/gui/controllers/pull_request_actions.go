package controllers

import (
	goContext "context"
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// The actions on a pull request beyond looking at it: the ones `gh pr merge`,
// `gh pr close`, `gh pr reopen`, `gh pr ready`, `gh pr update-branch`,
// `gh pr comment` and `gh pr review` take.

func prPlaceholders(pr *models.GithubPullRequest) map[string]string {
	return map[string]string{
		"number": fmt.Sprint(pr.Number),
		"title":  pr.Title,
		"base":   pr.BaseRefName,
		"head":   pr.HeadRefName,
	}
}

func (self *PullRequestsController) isOpen(pr *models.GithubPullRequest) *types.DisabledReason {
	if !pr.IsOpen() {
		return &types.DisabledReason{Text: self.c.Tr.PullRequests.NotOpen}
	}
	return nil
}

func (self *PullRequestsController) notMerged(pr *models.GithubPullRequest) *types.DisabledReason {
	if pr.State == "MERGED" {
		return &types.DisabledReason{Text: self.c.Tr.PullRequests.AlreadyMerged}
	}
	return nil
}

func (self *PullRequestsController) merge(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests

	return helper.WithDetails(pr, func(loaded *helpers.GithubPullRequestDetails) error {
		pr := loaded.PullRequest
		details := loaded.Details
		settings := details.Repository
		placeholders := prPlaceholders(pr)

		methodItem := func(label string, method string, allowed bool) *types.MenuItem {
			item := &types.MenuItem{
				Label:     label,
				OpensMenu: true,
				OnPress:   func() error { return self.confirmMerge(pr, details, method) },
			}
			if method == settings.ViewerDefaultMethod {
				item.Label += " " + tr.DefaultMethod
			}
			switch {
			case !allowed:
				item.DisabledReason = &types.DisabledReason{Text: tr.MergeMethodNotAllowed}
			case pr.IsDraft():
				item.DisabledReason = &types.DisabledReason{Text: tr.DraftCannotBeMerged}
			}
			return item
		}

		items := []*types.MenuItem{
			methodItem(tr.MergeCommit, github_commands.MergeMethodMerge, settings.MergeCommitAllowed),
			methodItem(tr.SquashAndMerge, github_commands.MergeMethodSquash, settings.SquashMergeAllowed),
			methodItem(tr.RebaseAndMerge, github_commands.MergeMethodRebase, settings.RebaseMergeAllowed),
		}

		if details.AutoMerge != nil {
			items = append(items, &types.MenuItem{
				Label: tr.DisableAutoMerge,
				OnPress: func() error {
					return helper.RunAction(pr, tr.DisablingAutoMerge,
						utils.ResolvePlaceholderString(tr.AutoMergeDisabled, placeholders),
						func(ctx goContext.Context, client *github_commands.Client) error {
							return client.DisableAutoMerge(ctx, pr.NodeId)
						})
				},
			})
		} else {
			autoMerge := &types.MenuItem{
				Label:     tr.EnableAutoMerge,
				Tooltip:   tr.EnableAutoMergeTooltip,
				OpensMenu: true,
				OnPress:   func() error { return self.enableAutoMerge(pr, details) },
			}
			switch {
			case !settings.AutoMergeAllowed:
				autoMerge.DisabledReason = &types.DisabledReason{Text: tr.AutoMergeNotAllowed}
			case details.MergeStateStatus == "CLEAN" || details.MergeStateStatus == "HAS_HOOKS":
				autoMerge.DisabledReason = &types.DisabledReason{Text: tr.AutoMergeNotNeeded}
			}
			items = append(items, autoMerge)
		}

		return self.c.Menu(types.CreateMenuOptions{
			Title: utils.ResolvePlaceholderString(tr.MergeTitle, placeholders),
			Items: items,
		})
	})
}

// confirmMerge asks whether to delete the head branch along with merging, and
// merges.
func (self *PullRequestsController) confirmMerge(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails, method string) error {
	tr := self.c.Tr.PullRequests
	localBranch := context.LocalBranchesOfPullRequests(self.c.Model())[pr.Number]

	mergeItem := func(deleteBranch bool) *types.MenuItem {
		label := tr.MergeKeepBranch
		if deleteBranch {
			label = tr.MergeDeleteBranch
			if localBranch != "" {
				label = utils.ResolvePlaceholderString(tr.MergeDeleteBranches, map[string]string{"branch": localBranch})
			}
		}
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return self.doMerge(pr, details, method, deleteBranch, localBranch)
			},
		}
	}

	deleteFirst := details.Repository.DeleteBranchOnMerge || localBranch != ""
	items := []*types.MenuItem{mergeItem(deleteFirst), mergeItem(!deleteFirst)}
	return self.c.Menu(types.CreateMenuOptions{
		Title: utils.ResolvePlaceholderString(tr.MergeConfirmTitle, prPlaceholders(pr)),
		Items: items,
	})
}

func (self *PullRequestsController) doMerge(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails, method string, deleteBranch bool, localBranch string) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	self.c.LogAction(self.c.Tr.Actions.MergePullRequest)
	deleteLocalBranch := self.deleteLocalBranchFunc(localBranch, pr, details)

	return helper.RunAction(pr, tr.Merging, utils.ResolvePlaceholderString(tr.Merged, prPlaceholders(pr)),
		func(ctx goContext.Context, client *github_commands.Client) error {
			if err := client.MergePullRequest(ctx, pr.NodeId, github_commands.MergePullRequestOptions{
				Method:          method,
				ExpectedHeadOid: pr.HeadRefOid,
			}); err != nil {
				return err
			}
			if !deleteBranch {
				return nil
			}
			// GitHub deletes the head branch itself if the repository is set
			// up to, and a fork's branch isn't ours to delete.
			if !details.Repository.DeleteBranchOnMerge && !pr.IsCrossRepository {
				if err := client.DeleteBranch(ctx, pr.HeadRefName); err != nil && !github_commands.IsUnprocessable(err) {
					self.c.Log.Warnf("Failed to delete branch %s: %v", pr.HeadRefName, err)
				}
			}
			return deleteLocalBranch()
		})
}

// deleteLocalBranchFunc returns a function that deletes the local branch of a
// merged pull request on a worker, checking out its base branch first if it
// is checked out, like `gh pr merge --delete-branch` does. What to do is
// worked out here, on the UI thread, from the model.
func (self *PullRequestsController) deleteLocalBranchFunc(localBranch string, pr *models.GithubPullRequest, details *models.GithubPullRequestDetails) func() error {
	if localBranch == "" {
		return func() error { return nil }
	}
	checkOutFirst := ""
	if localBranch == self.c.Model().CheckedOutBranch {
		target, found := lo.Find([]string{pr.BaseRefName, details.Repository.DefaultBranch}, func(name string) bool {
			return lo.SomeBy(self.c.Model().Branches, func(branch *models.Branch) bool { return branch.Name == name })
		})
		if !found {
			// There's nothing to switch to, so keep the branch.
			return func() error { return nil }
		}
		checkOutFirst = target
	}

	return func() error {
		if checkOutFirst != "" {
			if err := self.c.Git().Branch.Checkout(checkOutFirst, git_commands.CheckoutOptions{}); err != nil {
				return err
			}
		}
		err := self.c.Git().Branch.LocalDelete([]string{localBranch}, true)
		self.c.RefreshFromWorker(types.RefreshOptions{
			Scope: []types.RefreshableView{types.BRANCHES, types.COMMITS, types.FILES, types.REFLOG, types.WORKTREES},
		})
		return err
	}
}

func (self *PullRequestsController) enableAutoMerge(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails) error {
	tr := self.c.Tr.PullRequests
	settings := details.Repository
	item := func(label string, method string, allowed bool) *types.MenuItem {
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return self.c.Helpers().GithubPullRequests.RunAction(pr, tr.EnablingAutoMerge,
					utils.ResolvePlaceholderString(tr.AutoMergeEnabledToast, prPlaceholders(pr)),
					func(ctx goContext.Context, client *github_commands.Client) error {
						return client.EnableAutoMerge(ctx, pr.NodeId, method)
					})
			},
			DisabledReason: lo.Ternary(allowed, nil, &types.DisabledReason{Text: tr.MergeMethodNotAllowed}),
		}
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: tr.EnableAutoMerge,
		Items: []*types.MenuItem{
			item(tr.MergeCommit, github_commands.MergeMethodMerge, settings.MergeCommitAllowed),
			item(tr.SquashAndMerge, github_commands.MergeMethodSquash, settings.SquashMergeAllowed),
			item(tr.RebaseAndMerge, github_commands.MergeMethodRebase, settings.RebaseMergeAllowed),
		},
	})
}

// changeState offers to close or reopen a pull request, and to mark it ready
// for review or turn it back into a draft.
func (self *PullRequestsController) changeState(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	placeholders := prPlaceholders(pr)

	action := func(label string, waiting string, done string, f func(ctx goContext.Context, client *github_commands.Client) error) *types.MenuItem {
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return helper.RunAction(pr, waiting, utils.ResolvePlaceholderString(done, placeholders), f)
			},
		}
	}

	var items []*types.MenuItem
	if pr.IsOpen() {
		closeItem := &types.MenuItem{
			Label: tr.Close,
			OnPress: func() error {
				self.c.Confirm(types.ConfirmOpts{
					Title:  tr.Close,
					Prompt: utils.ResolvePlaceholderString(tr.CloseConfirm, placeholders),
					HandleConfirm: func() error {
						return helper.RunAction(pr, tr.Closing, utils.ResolvePlaceholderString(tr.Closed, placeholders),
							func(ctx goContext.Context, client *github_commands.Client) error {
								return client.ClosePullRequest(ctx, pr.NodeId)
							})
					},
				})
				return nil
			},
		}
		items = append(items, closeItem)
		if pr.IsDraft() {
			items = append(items, action(tr.MarkReady, tr.Updating, tr.MarkedReady,
				func(ctx goContext.Context, client *github_commands.Client) error {
					return client.MarkPullRequestReadyForReview(ctx, pr.NodeId)
				}))
		} else {
			items = append(items, action(tr.ConvertToDraft, tr.Updating, tr.ConvertedToDraft,
				func(ctx goContext.Context, client *github_commands.Client) error {
					return client.ConvertPullRequestToDraft(ctx, pr.NodeId)
				}))
		}
	} else {
		items = append(items, action(tr.Reopen, tr.Reopening, tr.Reopened,
			func(ctx goContext.Context, client *github_commands.Client) error {
				return client.ReopenPullRequest(ctx, pr.NodeId)
			}))
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: utils.ResolvePlaceholderString(tr.StateTitle, placeholders),
		Items: items,
	})
}

func (self *PullRequestsController) updateBranch(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	placeholders := prPlaceholders(pr)
	item := func(label string, rebase bool) *types.MenuItem {
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return self.c.Helpers().GithubPullRequests.RunAction(pr, tr.UpdatingBranch,
					utils.ResolvePlaceholderString(tr.BranchUpdated, placeholders),
					func(ctx goContext.Context, client *github_commands.Client) error {
						return client.UpdatePullRequestBranch(ctx, pr.NodeId, rebase)
					})
			},
		}
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title:  utils.ResolvePlaceholderString(tr.UpdateBranchTitle, placeholders),
		Prompt: utils.ResolvePlaceholderString(tr.UpdateBranchPrompt, placeholders),
		Items: []*types.MenuItem{
			item(tr.UpdateWithMerge, false),
			item(tr.UpdateWithRebase, true),
		},
	})
}

func (self *PullRequestsController) comment(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	return helper.EditText(helpers.EditTextOpts{
		Title: utils.ResolvePlaceholderString(tr.CommentTitle, prPlaceholders(pr)),
	}, func(body string) error {
		return helper.RunAction(pr, tr.Commenting, utils.ResolvePlaceholderString(tr.CommentAdded, prPlaceholders(pr)),
			func(ctx goContext.Context, client *github_commands.Client) error {
				return client.AddComment(ctx, pr.NodeId, body)
			})
	})
}

func (self *PullRequestsController) review(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests

	return helper.WithDetails(pr, func(loaded *helpers.GithubPullRequestDetails) error {
		details := loaded.Details
		pendingId := ""
		if details.PendingReview != nil {
			pendingId = details.PendingReview.NodeId
		}
		ownPullRequest := strings.EqualFold(pr.Author, details.ViewerLogin)

		submit := func(event string, body string) error {
			self.c.LogAction(self.c.Tr.Actions.ReviewPullRequest)
			return helper.RunAction(pr, tr.SubmittingReview, utils.ResolvePlaceholderString(tr.ReviewSubmitted, prPlaceholders(pr)),
				func(ctx goContext.Context, client *github_commands.Client) error {
					return client.SubmitReview(ctx, pr.NodeId, pendingId, event, body)
				})
		}
		withBody := func(title string, f func(body string) error) func() error {
			return func() error {
				return helper.EditText(helpers.EditTextOpts{Title: title}, f)
			}
		}
		notOwn := lo.Ternary(ownPullRequest, &types.DisabledReason{Text: tr.CannotReviewOwnPullRequest}, nil)

		items := []*types.MenuItem{
			{
				Label:          tr.Approve,
				OnPress:        func() error { return submit(github_commands.ReviewEventApprove, "") },
				DisabledReason: notOwn,
			},
			{
				Label:          tr.ApproveWithComment,
				OnPress:        withBody(tr.ApproveWithComment, func(body string) error { return submit(github_commands.ReviewEventApprove, body) }),
				DisabledReason: notOwn,
			},
			{
				Label:          tr.RequestChanges,
				OnPress:        withBody(tr.RequestChanges, func(body string) error { return submit(github_commands.ReviewEventRequestChanges, body) }),
				DisabledReason: notOwn,
			},
			{
				Label: tr.ReviewComment,
				OnPress: withBody(tr.ReviewComment, func(body string) error {
					return submit(github_commands.ReviewEventComment, body)
				}),
			},
		}
		if details.PendingReview != nil {
			items = append(items, &types.MenuItem{
				Label: tr.SubmitPendingReview,
				OnPress: func() error {
					return submit(github_commands.ReviewEventComment, "")
				},
			})
		}
		if details.PendingReview != nil {
			items = append(items, &types.MenuItem{
				Label: utils.ResolvePlaceholderString(tr.DiscardPendingReview, map[string]string{
					"count": fmt.Sprint(details.PendingReview.CommentsCount),
				}),
				OnPress: func() error {
					self.c.Confirm(types.ConfirmOpts{
						Title:  tr.DiscardPendingReviewTitle,
						Prompt: tr.DiscardPendingReviewPrompt,
						HandleConfirm: func() error {
							return helper.RunAction(pr, tr.Updating, tr.PendingReviewDiscarded,
								func(ctx goContext.Context, client *github_commands.Client) error {
									return client.DeletePendingReview(ctx, pendingId)
								})
						},
					})
					return nil
				},
			})
		}

		title := tr.ReviewTitle
		if details.PendingReview != nil {
			title = utils.ResolvePlaceholderString(tr.SubmitPendingReviewTitle, map[string]string{
				"count": fmt.Sprint(details.PendingReview.CommentsCount),
			})
		}
		return self.c.Menu(types.CreateMenuOptions{
			Title: utils.ResolvePlaceholderString(title, prPlaceholders(pr)),
			Items: items,
		})
	})
}
