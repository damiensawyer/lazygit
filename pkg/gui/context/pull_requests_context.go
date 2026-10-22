package context

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// PullRequestsContext lists the pull requests of the current repository's
// GitHub repository, as a tab of the branches panel.
type PullRequestsContext struct {
	*FilteredListViewModel[*models.GithubPullRequest]
	*ListContextTrait

	listFilter github_commands.ListPullRequestsOptions
}

var _ types.IListContext = (*PullRequestsContext)(nil)

func NewPullRequestsContext(c *ContextCommon) *PullRequestsContext {
	viewModel := NewFilteredListViewModel(
		func() []*models.GithubPullRequest { return c.Model().ListedPullRequests },
		func(pr *models.GithubPullRequest) []string {
			return []string{pr.Title, pr.ID(), pr.Author, pr.HeadRefName}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetPullRequestListDisplayStrings(presentation.PullRequestListOpts{
			PullRequests:     viewModel.GetItems(),
			LocalBranches:    LocalBranchesOfPullRequests(c.Model()),
			CheckedOutBranch: c.Model().CheckedOutBranch,
			FullDescription:  c.State().GetRepoState().GetScreenMode() != types.SCREEN_NORMAL,
			Now:              time.Now(),
			Tr:               c.Tr,
		})
	}

	return &PullRequestsContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
				View:                       c.Views().PullRequests,
				WindowName:                 "branches",
				Key:                        PULL_REQUESTS_CONTEXT_KEY,
				Kind:                       types.SIDE_CONTEXT,
				Focusable:                  true,
				NeedsRerenderOnWidthChange: types.NEEDS_RERENDER_ON_WIDTH_CHANGE_WHEN_SCREEN_MODE_CHANGES,
			})),
			ListRenderer: ListRenderer{
				list:              viewModel,
				getDisplayStrings: getDisplayStrings,
			},
			c: c,
		},
	}
}

// ListFilter returns which pull requests are loaded.
func (self *PullRequestsContext) ListFilter() github_commands.ListPullRequestsOptions {
	return self.listFilter
}

func (self *PullRequestsContext) SetListFilter(filter github_commands.ListPullRequestsOptions) {
	self.listFilter = filter
}

// LocalBranchesOfPullRequests maps the number of each listed pull request
// whose head branch a local branch tracks to the name of that local branch.
func LocalBranchesOfPullRequests(model *types.Model) map[int]string {
	branchToPr := git_commands.GenerateGithubPullRequestMap(model.ListedPullRequests, model.Branches, model.Remotes)
	result := make(map[int]string, len(branchToPr))
	for branchName, pr := range branchToPr {
		// Two local branches tracking the same remote branch are rare; prefer
		// the checked out one, then the one named like the head branch.
		if existing, ok := result[pr.Number]; ok && (existing == model.CheckedOutBranch || existing == pr.HeadRefName) {
			continue
		}
		result[pr.Number] = branchName
	}
	return result
}
