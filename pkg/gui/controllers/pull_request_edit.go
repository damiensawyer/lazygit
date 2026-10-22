package controllers

import (
	goContext "context"
	"errors"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// Editing a pull request, the way `gh pr edit` does.

func (self *PullRequestsController) edit(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests

	return helper.WithDetails(pr, func(loaded *helpers.GithubPullRequestDetails) error {
		pr := loaded.PullRequest
		details := loaded.Details
		cannotEdit := lo.Ternary(details.ViewerCanEdit, nil, &types.DisabledReason{Text: tr.CannotEdit})

		return self.c.Menu(types.CreateMenuOptions{
			Title: utils.ResolvePlaceholderString(tr.EditTitle, prPlaceholders(pr)),
			Items: []*types.MenuItem{
				{
					Label:          tr.EditTitleAndDescription,
					OnPress:        func() error { return self.editTitleAndDescription(pr, details) },
					DisabledReason: cannotEdit,
				},
				{
					Label:          tr.ChangeBase,
					OnPress:        func() error { return self.changeBase(pr) },
					DisabledReason: cannotEdit,
				},
				{
					Label:          tr.Labels,
					OpensMenu:      true,
					OnPress:        func() error { return self.editLabels(pr) },
					DisabledReason: cannotEdit,
				},
				{
					Label:          tr.Assignees,
					OpensMenu:      true,
					OnPress:        func() error { return self.editAssignees(pr, details) },
					DisabledReason: cannotEdit,
				},
				{
					Label:          tr.Reviewers,
					OpensMenu:      true,
					OnPress:        func() error { return self.editReviewers(pr, details) },
					DisabledReason: cannotEdit,
				},
			},
		})
	})
}

func (self *PullRequestsController) editTitleAndDescription(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	return helper.EditTitleAndBody(helpers.EditTitleAndBodyOpts{
		Title:        utils.ResolvePlaceholderString(tr.EditTitle, prPlaceholders(pr)),
		InitialTitle: pr.Title,
		InitialBody:  strings.ReplaceAll(details.Body, "\r\n", "\n"),
	}, func(title string, body string) error {
		return helper.RunAction(pr, tr.Updating, utils.ResolvePlaceholderString(tr.Edited, prPlaceholders(pr)),
			func(ctx goContext.Context, client *github_commands.Client) error {
				return client.UpdatePullRequest(ctx, pr.Number, github_commands.UpdatePullRequestOptions{Title: &title, Body: &body})
			})
	})
}

func (self *PullRequestsController) changeBase(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	self.c.Prompt(types.PromptOpts{
		Title:               utils.ResolvePlaceholderString(tr.ChangeBasePrompt, prPlaceholders(pr)),
		InitialContent:      pr.BaseRefName,
		FindSuggestionsFunc: self.c.Helpers().Suggestions.GetRemoteBranchesForRemoteSuggestionsFunc(self.baseRemoteName()),
		HandleConfirm: func(base string) error {
			base = strings.TrimSpace(base)
			if base == pr.BaseRefName {
				return nil
			}
			placeholders := prPlaceholders(pr)
			placeholders["base"] = base
			return self.c.Helpers().GithubPullRequests.RunAction(pr, tr.Updating,
				utils.ResolvePlaceholderString(tr.BaseChanged, placeholders),
				func(ctx goContext.Context, client *github_commands.Client) error {
					return client.UpdatePullRequest(ctx, pr.Number, github_commands.UpdatePullRequestOptions{Base: &base})
				})
		},
	})
	return nil
}

// withRepoData loads something about the repository on a worker and passes
// it to f on the UI thread.
func withRepoData[T any](self *PullRequestsController, load func(ctx goContext.Context, client *github_commands.Client) (T, error), f func(T) error) error {
	return self.c.Helpers().GithubClient.WithClient(func(client *github_commands.Client) error {
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.LoadingDetails, func(gocui.Task) error {
			data, err := load(goContext.Background(), client)
			self.c.OnUIThread(func() error {
				if err != nil {
					return self.c.Helpers().GithubClient.HandleError(err, self.c.Helpers().GithubPullRequests.Refresh)
				}
				return f(data)
			})
			return nil
		})
	})
}

func (self *PullRequestsController) editLabels(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	return withRepoData(self, func(ctx goContext.Context, client *github_commands.Client) ([]models.GithubLabel, error) {
		return client.ListLabels(ctx)
	}, func(labels []models.GithubLabel) error {
		if len(labels) == 0 {
			return errors.New(tr.NoLabels)
		}
		has := lo.SliceToMap(pr.Labels, func(label models.GithubLabel) (string, bool) { return label.Name, true })
		items := lo.Map(labels, func(label models.GithubLabel, _ int) *types.MenuItem {
			name := label.Name
			return &types.MenuItem{
				Label:  name,
				Widget: types.MakeMenuCheckBox(has[name]),
				OnPress: func() error {
					return helper.RunAction(pr, tr.Updating, "", func(ctx goContext.Context, client *github_commands.Client) error {
						if has[name] {
							return client.RemoveLabel(ctx, pr.Number, name)
						}
						return client.AddLabels(ctx, pr.Number, []string{name})
					})
				},
			}
		})
		return self.c.Menu(types.CreateMenuOptions{Title: tr.Labels, Items: items})
	})
}

func (self *PullRequestsController) editAssignees(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	change := func(add bool, logins []string) error {
		return helper.RunAction(pr, tr.Updating, "", func(ctx goContext.Context, client *github_commands.Client) error {
			if add {
				return client.AddAssignees(ctx, pr.Number, logins)
			}
			return client.RemoveAssignees(ctx, pr.Number, logins)
		})
	}

	items := lo.Map(details.Assignees, func(login string, _ int) *types.MenuItem {
		return &types.MenuItem{
			Label:   login,
			Widget:  types.MakeMenuCheckBox(true),
			Tooltip: tr.PressToRemove,
			OnPress: func() error { return change(false, []string{login}) },
		}
	})
	if !slices.Contains(details.Assignees, details.ViewerLogin) {
		items = append(items, &types.MenuItem{
			Label:   tr.AssignMe,
			OnPress: func() error { return change(true, []string{details.ViewerLogin}) },
		})
	}
	items = append(items, &types.MenuItem{
		Label: tr.AssignSomeone,
		OnPress: func() error {
			return self.promptForUser(tr.AssignSomeonePrompt, func(login string) error {
				return change(true, []string{login})
			})
		},
	})
	return self.c.Menu(types.CreateMenuOptions{Title: tr.Assignees, Items: items})
}

func (self *PullRequestsController) editReviewers(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails) error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	change := func(add bool, reviewers []string) error {
		done := lo.Ternary(add, utils.ResolvePlaceholderString(tr.ReviewRequested, map[string]string{
			"reviewer": strings.Join(reviewers, ", "),
		}), "")
		return helper.RunAction(pr, tr.Updating, done, func(ctx goContext.Context, client *github_commands.Client) error {
			if add {
				return client.RequestReviews(ctx, pr.Number, reviewers)
			}
			return client.RemoveReviewRequests(ctx, pr.Number, reviewers)
		})
	}

	items := lo.Map(details.ReviewRequests, func(reviewer string, _ int) *types.MenuItem {
		return &types.MenuItem{
			Label:   reviewer,
			Widget:  types.MakeMenuCheckBox(true),
			Tooltip: tr.PressToRemove,
			OnPress: func() error { return change(false, []string{reviewer}) },
		}
	})
	// Those who have reviewed already can be asked to review again.
	for _, review := range details.LatestReviews {
		if details.ReviewRequestedFrom(review.Author) || strings.EqualFold(review.Author, details.ViewerLogin) {
			continue
		}
		author := review.Author
		items = append(items, &types.MenuItem{
			Label:   utils.ResolvePlaceholderString(tr.RerequestReview, map[string]string{"reviewer": author}),
			OnPress: func() error { return change(true, []string{author}) },
		})
	}
	items = append(items, &types.MenuItem{
		Label: tr.RequestReviewFrom,
		OnPress: func() error {
			return self.promptForUser(tr.RequestReviewFromPrompt, func(reviewer string) error {
				return change(true, []string{reviewer})
			})
		},
	})
	return self.c.Menu(types.CreateMenuOptions{Title: tr.Reviewers, Items: items})
}

// promptForUser asks for a login, suggesting the users that can be assigned
// to the repository's pull requests.
func (self *PullRequestsController) promptForUser(title string, onConfirm func(login string) error) error {
	return withRepoData(self, func(ctx goContext.Context, client *github_commands.Client) ([]string, error) {
		return client.ListAssignableUsers(ctx)
	}, func(users []string) error {
		self.c.Prompt(types.PromptOpts{
			Title:               title,
			FindSuggestionsFunc: helpers.FilterFunc(users, self.c.UserConfig().Gui.UseFuzzySearch()),
			HandleConfirm: func(login string) error {
				login = strings.TrimPrefix(strings.TrimSpace(login), "@")
				if login == "" {
					return nil
				}
				return onConfirm(login)
			},
		})
		return nil
	})
}
