package context

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// GithubRunsContext lists the GitHub Actions workflow runs of the current
// repository. It takes over a side window when opened, like the sub-commits
// view does.
type GithubRunsContext struct {
	*FilteredListViewModel[*models.GithubRun]
	*ListContextTrait

	runsFilter GithubRunsFilter
}

// GithubRunsFilter narrows down which runs are loaded. Zero values mean
// "don't filter by this".
type GithubRunsFilter struct {
	Branch   string
	Status   string
	Event    string
	HeadSha  string
	Workflow *models.GithubWorkflow
}

func (self GithubRunsFilter) IsActive() bool {
	return self != GithubRunsFilter{}
}

var _ types.IListContext = (*GithubRunsContext)(nil)

func NewGithubRunsContext(c *ContextCommon) *GithubRunsContext {
	viewModel := NewFilteredListViewModel(
		func() []*models.GithubRun { return c.Model().GithubRuns },
		func(run *models.GithubRun) []string {
			return []string{run.Title(), run.Name, run.HeadBranch, run.Event, run.Actor.Login}
		},
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetGithubRunListDisplayStrings(
			c.Tr,
			viewModel.GetItems(),
			c.State().GetRepoState().GetScreenMode() != types.SCREEN_NORMAL,
			time.Now(),
		)
	}

	return &GithubRunsContext{
		FilteredListViewModel: viewModel,
		ListContextTrait: &ListContextTrait{
			Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
				View:                       c.Views().GithubRuns,
				WindowName:                 "commits",
				Key:                        GITHUB_RUNS_CONTEXT_KEY,
				Kind:                       types.SIDE_CONTEXT,
				Focusable:                  true,
				Transient:                  true,
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

func (self *GithubRunsContext) GetRunsFilter() GithubRunsFilter {
	return self.runsFilter
}

func (self *GithubRunsContext) SetRunsFilter(filter GithubRunsFilter) {
	self.runsFilter = filter
}
