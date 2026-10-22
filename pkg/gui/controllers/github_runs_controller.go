package controllers

import (
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type GithubRunsController struct {
	baseController
	*ListControllerTrait[*models.GithubRun]
	c *ControllerCommon
}

var _ types.IController = &GithubRunsController{}

func NewGithubRunsController(c *ControllerCommon) *GithubRunsController {
	return &GithubRunsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().GithubRuns,
			c.Contexts().GithubRuns.GetSelected,
			c.Contexts().GithubRuns.GetSelectedItems,
		),
		c: c,
	}
}

func (self *GithubRunsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	tr := self.c.Tr.GithubActions

	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Handler:           self.withItem(self.c.Helpers().GithubActions.OpenJobs),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       tr.ViewJobs,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.GithubActions.OpenInBrowser),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       tr.OpenRunInBrowser,
			DisplayOnScreen:   true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.c.Helpers().GithubActions.RefreshRuns,
			Description: self.c.Tr.Refresh,
		},
	}
}

func (self *GithubRunsController) GetOnRenderToMain() func() {
	return func() {
		run := self.context().GetSelected()
		if run == nil {
			self.renderToMain(self.c.Tr.GithubActions.NoRuns)
			return
		}

		opts := presentation.GithubRunSummaryOpts{Run: run, Now: time.Now()}
		if details := self.c.Helpers().GithubActions.GetRunDetails(run); details != nil {
			opts.JobsLoaded = true
			opts.Jobs = details.Jobs
			opts.JobsError = details.Err
			opts.Annotations = details.Annotations
		}
		self.renderToMain(presentation.FormatGithubRunSummary(self.c.Tr, opts))
	}
}

func (self *GithubRunsController) renderToMain(content string) {
	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: self.c.Tr.GithubActions.RunMainTitle,
			Task:  types.NewRenderStringTask(content),
		},
	})
}

func (self *GithubRunsController) openInBrowser(run *models.GithubRun) error {
	return self.c.OS().OpenLink(run.Url)
}

func (self *GithubRunsController) context() *context.GithubRunsContext {
	return self.c.Contexts().GithubRuns
}
