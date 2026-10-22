package controllers

import (
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type GithubStepsController struct {
	baseController
	*ListControllerTrait[*models.GithubStep]
	c *ControllerCommon

	mainView *githubMainViewRenderer
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
		mainView: &githubMainViewRenderer{
			c:     c,
			title: func() string { return c.Tr.GithubActions.StepMainTitle },
		},
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
			ShortDescription:  self.c.Tr.GithubActions.OpenLogInEditorShort,
			Tooltip:           self.c.Tr.GithubActions.OpenLogInEditorTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.GithubActions.CopyToClipboard),
			Handler:           self.withItem(self.copyStepLog),
			GetDisabledReason: self.require(self.singleItemSelected(self.jobIsComplete)),
			Description:       self.c.Tr.GithubActions.CopyStepLogAction,
			DisplayOnScreen:   true,
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
			self.mainView.render("", self.c.Tr.GithubActions.NoSteps)
			return
		}

		tr := self.c.Tr.GithubActions
		stepId := fmt.Sprintf("%d/%d", job.JobId, step.Number)
		if job.State().IsActive() {
			self.mainView.render(stepId, fmt.Sprintf("%s %s\n\n%s\n",
				presentation.GithubActionsStateIcon(step.State()), style.AttrBold.Sprint(step.Name), tr.LogNotAvailableYet))
			return
		}

		jobLog := self.c.Helpers().GithubActions.GetJobLog(job)
		switch {
		case jobLog == nil:
			self.mainView.render(stepId, tr.LoadingLog)
		case jobLog.Err != nil:
			self.mainView.render(stepId, style.FgRed.Sprint(jobLog.Err.Error()))
		default:
			self.mainView.render(stepId, presentation.FormatGithubStepLog(self.c.Tr, job, step, jobLog.Log, time.Now()))
		}
	}
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

func (self *GithubStepsController) copyStepLog(step *models.GithubStep) error {
	tr := self.c.Tr.GithubActions
	return self.c.Helpers().GithubActions.WithJobLog(self.context().GetJob(), func(log *github_commands.JobLog) error {
		stepLog, _ := lo.Find(log.Steps, func(stepLog github_commands.StepLog) bool {
			return stepLog.StepNumber == step.Number
		})
		if err := self.c.OS().CopyToClipboard(utils.Decolorise(joinLogLines(stepLog.Lines))); err != nil {
			return err
		}
		self.c.Toast(utils.ResolvePlaceholderString(tr.CopiedToClipboard, map[string]string{"what": tr.CopyStepLog}))
		return nil
	})
}
