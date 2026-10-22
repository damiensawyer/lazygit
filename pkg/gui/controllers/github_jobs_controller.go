package controllers

import (
	goContext "context"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
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
			Keys:              opts.GetKeys(opts.Config.GithubActions.Rerun),
			Handler:           self.withItem(self.rerun),
			GetDisabledReason: self.require(self.singleItemSelected(self.runIsComplete)),
			Description:       tr.RerunJob,
			Tooltip:           tr.RerunJobTooltip,
			OpensMenu:         true,
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

// GitHub only re-runs jobs of a run that has completed as a whole.
func (self *GithubJobsController) runIsComplete(*models.GithubJob) *types.DisabledReason {
	if run := self.context().GetRun(); run != nil && run.State().IsActive() {
		return &types.DisabledReason{Text: self.c.Tr.GithubActions.RunStillInProgress}
	}
	return nil
}

func (self *GithubJobsController) rerun(job *models.GithubJob) error {
	tr := self.c.Tr.GithubActions

	rerunItem := func(label string, debug bool) *types.MenuItem {
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return self.c.Helpers().GithubActions.RunAction(tr.RequestingRerun, tr.RerunRequested, func(ctx goContext.Context, client *github_commands.Client) error {
					return client.RerunJob(ctx, job.JobId, debug)
				})
			},
		}
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: utils.ResolvePlaceholderString(tr.RerunJobMenuTitle, map[string]string{"job": job.Name}),
		Items: []*types.MenuItem{
			rerunItem(tr.RerunThisJob, false),
			rerunItem(tr.RerunThisJobWithDebug, true),
		},
	})
}
