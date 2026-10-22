package controllers

import (
	"fmt"
	"slices"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type PullRequestsController struct {
	baseController
	*ListControllerTrait[*models.GithubPullRequest]
	c *ControllerCommon

	mainView *githubMainViewRenderer
}

var _ types.IController = &PullRequestsController{}

func NewPullRequestsController(c *ControllerCommon) *PullRequestsController {
	return &PullRequestsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().PullRequests,
			c.Contexts().PullRequests.GetSelected,
			c.Contexts().PullRequests.GetSelectedItems,
		),
		c: c,
		mainView: &githubMainViewRenderer{
			c:     c,
			title: func() string { return c.Tr.PullRequests.MainTitle },
		},
	}
}

func (self *PullRequestsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	tr := self.c.Tr.PullRequests

	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.OpenInBrowser),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       tr.OpenInBrowser,
			Tooltip:           tr.OpenInBrowserTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.PullRequests.CopyToClipboard),
			Handler:           self.withItem(self.copyToClipboard),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       tr.CopyToClipboard,
			ShortDescription:  tr.CopyShort,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:             opts.GetKeys(opts.Config.PullRequests.Filter),
			Handler:          self.filterMenu,
			Description:      tr.Filter,
			ShortDescription: tr.FilterShort,
			Tooltip:          tr.FilterTooltip,
			OpensMenu:        true,
			DisplayOnScreen:  true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.c.Helpers().GithubPullRequests.Refresh,
			Description: tr.Refresh,
			Tooltip:     tr.RefreshTooltip,
		},
	}
}

func (self *PullRequestsController) GetOnFocus() func(types.OnFocusOpts) {
	return func(types.OnFocusOpts) {
		self.c.Helpers().GithubPullRequests.OnFocus()
	}
}

func (self *PullRequestsController) GetOnRenderToMain() func() {
	return func() {
		helper := self.c.Helpers().GithubPullRequests
		pr := self.context().GetSelected()
		if pr == nil {
			self.mainView.render("", helper.MainViewMessage())
			return
		}

		opts := presentation.PullRequestDetailsOpts{
			PullRequest: pr,
			LocalBranch: context.LocalBranchesOfPullRequests(self.c.Model())[pr.Number],
			Now:         time.Now(),
		}
		if details := helper.GetDetails(pr); details != nil {
			opts.DetailsErr = details.Err
			if details.Err == nil {
				opts.PullRequest = details.PullRequest
				opts.Details = details.Details
			}
		}
		self.mainView.render(pr.ID(), presentation.FormatPullRequestDetails(self.c.Tr, opts))
	}
}

func (self *PullRequestsController) context() *context.PullRequestsContext {
	return self.c.Contexts().PullRequests
}

func (self *PullRequestsController) openInBrowser(pr *models.GithubPullRequest) error {
	return self.c.OS().OpenLink(pr.Url)
}

func (self *PullRequestsController) copyToClipboard(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	return githubCopyMenu(self.c, []githubCopyItem{
		githubCopyValue(tr.CopyURL, pr.Url),
		githubCopyValue(tr.CopyNumber, fmt.Sprintf("#%d", pr.Number)),
		githubCopyValue(tr.CopyTitle, pr.Title),
		githubCopyValue(tr.CopyHeadBranch, pr.HeadRefName),
		{
			label: tr.CopyDescription,
			get: func(f func(string) error) error {
				return self.c.Helpers().GithubPullRequests.WithDetails(pr, func(details *helpers.GithubPullRequestDetails) error {
					return f(details.Details.Body)
				})
			},
		},
	})
}

func (self *PullRequestsController) filterMenu() error {
	tr := self.c.Tr.PullRequests
	helper := self.c.Helpers().GithubPullRequests
	filter := self.context().ListFilter()

	setFilter := func(change func(filter *github_commands.ListPullRequestsOptions)) error {
		newFilter := filter
		newFilter.Labels = slices.Clone(filter.Labels)
		change(&newFilter)
		return helper.SetListFilter(newFilter)
	}
	toggle := func(field *string, value string) {
		if *field == value {
			*field = ""
		} else {
			*field = value
		}
	}
	withValue := func(label string, value string) string {
		if value == "" {
			return label
		}
		return label + " " + style.FgCyan.Sprint(value)
	}
	prompt := func(title string, initial string, suggestions func(string) []*types.Suggestion, onConfirm func(filter *github_commands.ListPullRequestsOptions, value string)) func() error {
		return func() error {
			self.c.Prompt(types.PromptOpts{
				Title:               title,
				InitialContent:      initial,
				FindSuggestionsFunc: suggestions,
				AllowEmptyInput:     true,
				HandleConfirm: func(value string) error {
					return setFilter(func(filter *github_commands.ListPullRequestsOptions) { onConfirm(filter, value) })
				},
			})
			return nil
		}
	}

	stateLabel := map[string]string{
		"":                                     tr.FilterOpen,
		github_commands.PullRequestStateOpen:   tr.FilterOpen,
		github_commands.PullRequestStateClosed: tr.FilterClosed,
		github_commands.PullRequestStateMerged: tr.FilterMerged,
		github_commands.PullRequestStateAll:    tr.FilterAll,
	}[filter.State]

	return self.c.Menu(types.CreateMenuOptions{
		Title: tr.Filter,
		Items: []*types.MenuItem{
			{
				Label:     withValue(tr.FilterState, stateLabel),
				OpensMenu: true,
				OnPress:   func() error { return self.selectState(filter.State, setFilter) },
			},
			{
				Label:  tr.FilterMine,
				Widget: types.MakeMenuCheckBox(filter.Author == "@me"),
				OnPress: func() error {
					return setFilter(func(filter *github_commands.ListPullRequestsOptions) { toggle(&filter.Author, "@me") })
				},
			},
			{
				Label:  tr.FilterReviewRequested,
				Widget: types.MakeMenuCheckBox(filter.ReviewRequested == "@me"),
				OnPress: func() error {
					return setFilter(func(filter *github_commands.ListPullRequestsOptions) { toggle(&filter.ReviewRequested, "@me") })
				},
			},
			{
				Label:  tr.FilterAssigned,
				Widget: types.MakeMenuCheckBox(filter.Assignee == "@me"),
				OnPress: func() error {
					return setFilter(func(filter *github_commands.ListPullRequestsOptions) { toggle(&filter.Assignee, "@me") })
				},
			},
			{
				Label:  withValue(tr.FilterAuthor, lo.Ternary(filter.Author != "@me", filter.Author, "")),
				Widget: types.MakeMenuCheckBox(filter.Author != "" && filter.Author != "@me"),
				OnPress: prompt(tr.FilterAuthorPrompt, lo.Ternary(filter.Author != "@me", filter.Author, ""), nil,
					func(filter *github_commands.ListPullRequestsOptions, value string) { filter.Author = value }),
			},
			{
				Label:  withValue(tr.FilterLabel, lo.Ternary(len(filter.Labels) > 0, lo.FirstOrEmpty(filter.Labels), "")),
				Widget: types.MakeMenuCheckBox(len(filter.Labels) > 0),
				OnPress: prompt(tr.FilterLabelPrompt, lo.FirstOrEmpty(filter.Labels), nil,
					func(filter *github_commands.ListPullRequestsOptions, value string) {
						filter.Labels = lo.Ternary(value == "", nil, []string{value})
					}),
			},
			{
				Label:  withValue(tr.FilterBase, filter.BaseBranch),
				Widget: types.MakeMenuCheckBox(filter.BaseBranch != ""),
				OnPress: prompt(tr.FilterBasePrompt, filter.BaseBranch, self.c.Helpers().Suggestions.GetRemoteBranchesForRemoteSuggestionsFunc(self.baseRemoteName()),
					func(filter *github_commands.ListPullRequestsOptions, value string) { filter.BaseBranch = value }),
			},
			{
				Label:  withValue(tr.FilterSearch, filter.Search),
				Widget: types.MakeMenuCheckBox(filter.Search != ""),
				OnPress: prompt(tr.FilterSearchPrompt, filter.Search, nil,
					func(filter *github_commands.ListPullRequestsOptions, value string) { filter.Search = value }),
			},
			{
				Label: tr.ClearFilter,
				OnPress: func() error {
					return helper.SetListFilter(github_commands.ListPullRequestsOptions{})
				},
				DisabledReason: lo.Ternary(isDefaultPullRequestsFilter(filter), &types.DisabledReason{Text: tr.NoFilter}, nil),
			},
		},
	})
}

func isDefaultPullRequestsFilter(filter github_commands.ListPullRequestsOptions) bool {
	return (filter.State == "" || filter.State == github_commands.PullRequestStateOpen) &&
		filter.Author == "" && filter.Assignee == "" && filter.ReviewRequested == "" &&
		len(filter.Labels) == 0 && filter.BaseBranch == "" && filter.Search == ""
}

func (self *PullRequestsController) selectState(current string, setFilter func(func(*github_commands.ListPullRequestsOptions)) error) error {
	tr := self.c.Tr.PullRequests
	if current == "" {
		current = github_commands.PullRequestStateOpen
	}
	item := func(state string, label string) *types.MenuItem {
		return &types.MenuItem{
			Label:  label,
			Widget: types.MakeMenuRadioButton(current == state),
			OnPress: func() error {
				return setFilter(func(filter *github_commands.ListPullRequestsOptions) { filter.State = state })
			},
		}
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title: tr.FilterState,
		Items: []*types.MenuItem{
			item(github_commands.PullRequestStateOpen, tr.FilterOpen),
			item(github_commands.PullRequestStateClosed, tr.FilterClosed),
			item(github_commands.PullRequestStateMerged, tr.FilterMerged),
			item(github_commands.PullRequestStateAll, tr.FilterAll),
		},
	})
}

// baseRemoteName returns the name of the remote that is the pull requests'
// repository, if known.
func (self *PullRequestsController) baseRemoteName() string {
	return self.c.Helpers().GithubClient.BaseRemoteName()
}
