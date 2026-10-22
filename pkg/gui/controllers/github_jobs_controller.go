package controllers

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type GithubJobsController struct {
	baseController
	*ListControllerTrait[*models.GithubJob]
	c *ControllerCommon
}

var _ types.IController = &GithubJobsController{}

func NewGithubJobsController(c *ControllerCommon) *GithubJobsController {
	return &GithubJobsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().GithubJobs,
			c.Contexts().GithubJobs.GetSelected,
			c.Contexts().GithubJobs.GetSelectedItems,
		),
		c: c,
	}
}

func (self *GithubJobsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	tr := self.c.Tr.GithubActions

	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Handler:           self.withItem(self.c.Helpers().GithubActions.OpenSteps),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       tr.ViewSteps,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.GithubActions.OpenInBrowser),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       tr.OpenJobInBrowser,
			DisplayOnScreen:   true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.c.Helpers().GithubActions.RefreshRuns,
			Description: self.c.Tr.Refresh,
		},
	}
}

func (self *GithubJobsController) GetOnRenderToMain() func() {
	return func() {
		job := self.context().GetSelected()
		if job == nil {
			self.renderToMain(self.c.Tr.GithubActions.NoJobs)
			return
		}

		opts := presentation.GithubJobDetailsOpts{Job: job, Now: time.Now()}
		if !job.State().IsActive() {
			if jobLog := self.c.Helpers().GithubActions.GetJobLog(job); jobLog != nil {
				opts.LogLoaded = true
				opts.Log = jobLog.Log
				opts.LogError = jobLog.Err
				opts.Annotations = jobLog.Annotations
			}
		}
		self.renderToMain(presentation.FormatGithubJobDetails(self.c.Tr, opts))
	}
}

func (self *GithubJobsController) renderToMain(content string) {
	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: self.c.Tr.GithubActions.JobMainTitle,
			Task:  types.NewRenderStringTask(content),
		},
	})
}

func (self *GithubJobsController) openInBrowser(job *models.GithubJob) error {
	return self.c.OS().OpenLink(job.Url)
}

func (self *GithubJobsController) context() *context.GithubJobsContext {
	return self.c.Contexts().GithubJobs
}
