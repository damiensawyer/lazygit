package controllers

import (
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type GithubStepsController struct {
	baseController
	*ListControllerTrait[*models.GithubStep]
	c *ControllerCommon
}

var _ types.IController = &GithubStepsController{}

func NewGithubStepsController(c *ControllerCommon) *GithubStepsController {
	return &GithubStepsController{
		baseController: baseController{},
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().GithubSteps,
			c.Contexts().GithubSteps.GetSelected,
			c.Contexts().GithubSteps.GetSelectedItems,
		),
		c: c,
	}
}

func (self *GithubStepsController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.GithubActions.OpenInBrowser),
			Handler:           self.withItem(self.openInBrowser),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.GithubActions.OpenJobInBrowser,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Edit),
			Handler:           self.withItem(self.openLogInEditor),
			GetDisabledReason: self.require(self.singleItemSelected(self.jobIsComplete)),
			Description:       self.c.Tr.GithubActions.OpenLogInEditor,
			Tooltip:           self.c.Tr.GithubActions.OpenLogInEditorTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.c.Helpers().GithubActions.RefreshRuns,
			Description: self.c.Tr.Refresh,
		},
	}
}

func (self *GithubStepsController) GetOnRenderToMain() func() {
	return func() {
		job := self.context().GetJob()
		step := self.context().GetSelected()
		if job == nil || step == nil {
			self.renderToMain(self.c.Tr.GithubActions.NoSteps)
			return
		}

		tr := self.c.Tr.GithubActions
		if job.State().IsActive() {
			self.renderToMain(fmt.Sprintf("%s %s\n\n%s\n",
				presentation.GithubActionsStateIcon(step.State()), style.AttrBold.Sprint(step.Name), tr.LogNotAvailableYet))
			return
		}

		jobLog := self.c.Helpers().GithubActions.GetJobLog(job)
		switch {
		case jobLog == nil:
			self.renderToMain(tr.LoadingLog)
		case jobLog.Err != nil:
			self.renderToMain(style.FgRed.Sprint(jobLog.Err.Error()))
		default:
			self.renderToMain(presentation.FormatGithubStepLog(self.c.Tr, job, step, jobLog.Log, time.Now()))
		}
	}
}

func (self *GithubStepsController) renderToMain(content string) {
	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: self.c.Tr.GithubActions.StepMainTitle,
			Task:  types.NewRenderStringTask(content),
		},
	})
}

func (self *GithubStepsController) openInBrowser(step *models.GithubStep) error {
	// GitHub's job page expands and scrolls to the step named in the anchor.
	return self.c.OS().OpenLink(fmt.Sprintf("%s#step:%d:1", self.context().GetJob().Url, step.Number))
}

func (self *GithubStepsController) context() *context.GithubStepsContext {
	return self.c.Contexts().GithubSteps
}

func (self *GithubStepsController) jobIsComplete(*models.GithubStep) *types.DisabledReason {
	if job := self.context().GetJob(); job != nil && job.State().IsActive() {
		return &types.DisabledReason{Text: self.c.Tr.GithubActions.LogNotAvailableYet}
	}
	return nil
}

func (self *GithubStepsController) openLogInEditor(step *models.GithubStep) error {
	return self.c.Helpers().GithubActions.WithJobLogFile(self.context().GetJob(), step.Number, self.c.Helpers().Files.EditFileAtLine)
}
