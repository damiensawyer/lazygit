package controllers

import (
	goContext "context"
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
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
			Keys:              opts.GetKeys(opts.Config.GithubActions.Rerun),
			Handler:           self.withItem(self.rerun),
			GetDisabledReason: self.require(self.singleItemSelected(self.runIsComplete)),
			Description:       tr.Rerun,
			Tooltip:           tr.RerunTooltip,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.GithubActions.CancelRun),
			Handler:           self.withItem(self.cancel),
			GetDisabledReason: self.require(self.singleItemSelected(self.runIsActive)),
			Description:       tr.CancelRun,
			Tooltip:           tr.CancelRunTooltip,
			OpensMenu:         true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Remove),
			Handler:           self.withItem(self.delete),
			GetDisabledReason: self.require(self.singleItemSelected(self.runIsComplete)),
			Description:       tr.DeleteRun,
			Tooltip:           tr.DeleteRunTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.c.Helpers().GithubActions.RefreshRuns,
			Description: self.c.Tr.Refresh,
		},
	}
}

func (self *GithubRunsController) runIsComplete(run *models.GithubRun) *types.DisabledReason {
	if run.State().IsActive() {
		return &types.DisabledReason{Text: self.c.Tr.GithubActions.RunStillInProgress}
	}
	return nil
}

func (self *GithubRunsController) runIsActive(run *models.GithubRun) *types.DisabledReason {
	if !run.State().IsActive() {
		return &types.DisabledReason{Text: self.c.Tr.GithubActions.RunNotInProgress}
	}
	return nil
}

func runPlaceholders(run *models.GithubRun) map[string]string {
	return map[string]string{
		"workflow": run.Name,
		"number":   fmt.Sprint(run.Number),
		"title":    run.Title(),
	}
}

func (self *GithubRunsController) rerun(run *models.GithubRun) error {
	tr := self.c.Tr.GithubActions
	helper := self.c.Helpers().GithubActions

	rerunItem := func(label string, failedOnly bool, debug bool) *types.MenuItem {
		item := &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return helper.RunAction(tr.RequestingRerun, tr.RerunRequested, func(ctx goContext.Context, client *github_commands.Client) error {
					if failedOnly {
						return client.RerunFailedJobs(ctx, run.RunId, debug)
					}
					return client.RerunRun(ctx, run.RunId, debug)
				})
			},
		}
		if failedOnly && !run.State().IsFailed() {
			item.DisabledReason = &types.DisabledReason{Text: tr.NoFailedJobs}
		}
		return item
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: utils.ResolvePlaceholderString(tr.RerunMenuTitle, runPlaceholders(run)),
		Items: []*types.MenuItem{
			rerunItem(tr.RerunFailedJobs, true, false),
			rerunItem(tr.RerunAllJobs, false, false),
			rerunItem(tr.RerunFailedJobsWithDebug, true, true),
			rerunItem(tr.RerunAllJobsWithDebug, false, true),
		},
	})
}

func (self *GithubRunsController) cancel(run *models.GithubRun) error {
	tr := self.c.Tr.GithubActions
	helper := self.c.Helpers().GithubActions

	cancelItem := func(label string, force bool) *types.MenuItem {
		return &types.MenuItem{
			Label: label,
			OnPress: func() error {
				return helper.RunAction(tr.CancellingRun, tr.CancelRequested, func(ctx goContext.Context, client *github_commands.Client) error {
					if force {
						return client.ForceCancelRun(ctx, run.RunId)
					}
					return client.CancelRun(ctx, run.RunId)
				})
			},
		}
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: utils.ResolvePlaceholderString(tr.CancelRunMenuTitle, runPlaceholders(run)),
		Items: []*types.MenuItem{
			cancelItem(tr.CancelRun, false),
			cancelItem(tr.ForceCancelRun, true),
		},
	})
}

func (self *GithubRunsController) delete(run *models.GithubRun) error {
	tr := self.c.Tr.GithubActions

	self.c.Confirm(types.ConfirmOpts{
		Title:  tr.DeleteRunTitle,
		Prompt: utils.ResolvePlaceholderString(tr.DeleteRunPrompt, runPlaceholders(run)),
		HandleConfirm: func() error {
			return self.c.Helpers().GithubActions.RunAction(tr.DeletingRun, tr.RunDeleted, func(ctx goContext.Context, client *github_commands.Client) error {
				return client.DeleteRun(ctx, run.RunId)
			})
		},
	})
	return nil
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
