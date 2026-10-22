package context

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// GithubJobsContext lists the jobs of the workflow run it was opened for.
type GithubJobsContext struct {
	*FilteredListViewModel[*models.GithubJob]
	*ListContextTrait

	run  *models.GithubRun
	jobs []*models.GithubJob
}

var _ types.IListContext = (*GithubJobsContext)(nil)

func NewGithubJobsContext(c *ContextCommon) *GithubJobsContext {
	self := &GithubJobsContext{}

	viewModel := NewFilteredListViewModel(
		func() []*models.GithubJob { return self.jobs },
		func(job *models.GithubJob) []string { return []string{job.Name} },
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetGithubJobListDisplayStrings(viewModel.GetItems(), time.Now())
	}

	self.FilteredListViewModel = viewModel
	self.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
			View:       c.Views().GithubJobs,
			WindowName: "commits",
			Key:        GITHUB_JOBS_CONTEXT_KEY,
			Kind:       types.SIDE_CONTEXT,
			Focusable:  true,
			Transient:  true,
		})),
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		c: c,
	}

	return self
}

func (self *GithubJobsContext) GetRun() *models.GithubRun {
	return self.run
}

func (self *GithubJobsContext) SetJobs(run *models.GithubRun, jobs []*models.GithubJob) {
	self.run = run
	self.jobs = jobs
	self.GetView().Title = utils.ResolvePlaceholderString(self.c.Tr.GithubActions.JobsTitleForRun, map[string]string{
		"run": utils.TruncateWithEllipsis(run.Title(), 50),
	})
}
