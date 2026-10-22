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
