package context

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// GithubStepsContext lists the steps of the job it was opened for.
type GithubStepsContext struct {
	*FilteredListViewModel[*models.GithubStep]
	*ListContextTrait

	job   *models.GithubJob
	steps []*models.GithubStep
}

var _ types.IListContext = (*GithubStepsContext)(nil)

func NewGithubStepsContext(c *ContextCommon) *GithubStepsContext {
	self := &GithubStepsContext{}

	viewModel := NewFilteredListViewModel(
		func() []*models.GithubStep { return self.steps },
		func(step *models.GithubStep) []string { return []string{step.Name} },
	)

	getDisplayStrings := func(_ int, _ int) [][]string {
		return presentation.GetGithubStepListDisplayStrings(self.c.Tr, viewModel.GetItems(), time.Now())
	}

	self.FilteredListViewModel = viewModel
	self.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
			View:       c.Views().GithubSteps,
			WindowName: "commits",
			Key:        GITHUB_STEPS_CONTEXT_KEY,
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

func (self *GithubStepsContext) GetJob() *models.GithubJob {
	return self.job
}

func (self *GithubStepsContext) SetJob(job *models.GithubJob) {
	self.job = job
	self.steps = lo.Map(job.Steps, func(_ models.GithubStep, i int) *models.GithubStep {
		return &job.Steps[i]
	})
	self.GetView().Title = utils.ResolvePlaceholderString(self.c.Tr.GithubActions.StepsTitleForJob, map[string]string{
		"job": utils.TruncateWithEllipsis(job.Name, 50),
	})
}
