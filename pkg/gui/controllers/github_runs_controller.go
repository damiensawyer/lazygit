package controllers

import (
	goContext "context"
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
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
			Keys:        opts.GetKeys(opts.Config.GithubActions.FilterRuns),
			Handler:     self.filterMenu,
			Description: tr.FilterRuns,
			Tooltip:     tr.FilterRunsTooltip,
			OpensMenu:   true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     self.c.Helpers().GithubActions.RefreshRuns,
			Description: self.c.Tr.Refresh,
		},
	}
}

func (self *GithubRunsController) filterMenu() error {
	tr := self.c.Tr.GithubActions
	helper := self.c.Helpers().GithubActions
	filter := self.context().GetRunsFilter()

	setFilter := func(change func(filter *context.GithubRunsFilter)) error {
		newFilter := filter
		change(&newFilter)
		return helper.SetRunsFilter(newFilter)
	}

	currentBranchName := ""
	if branch := self.c.Helpers().Refs.GetCheckedOutRef(); branch != nil && !branch.DetachedHead {
		currentBranchName = branch.Name
	}
	currentBranchItem := &types.MenuItem{
		Label:  utils.ResolvePlaceholderString(tr.FilterByCurrentBranch, map[string]string{"branch": currentBranchName}),
		Widget: types.MakeMenuCheckBox(filter.Branch != "" && filter.Branch == currentBranchName),
		OnPress: func() error {
			return setFilter(func(filter *context.GithubRunsFilter) {
				if filter.Branch == currentBranchName {
					filter.Branch = ""
				} else {
					filter.Branch = currentBranchName
				}
			})
		},
	}
	if currentBranchName == "" {
		currentBranchItem.DisabledReason = &types.DisabledReason{Text: tr.NotOnABranch}
	}

	commit := self.c.Contexts().LocalCommits.GetSelected()
	commitItem := &types.MenuItem{
		Widget: types.MakeMenuCheckBox(filter.HeadSha != "" && commit != nil && filter.HeadSha == commit.Hash()),
		Label:  tr.FilterBySelectedCommit,
	}
	if commit == nil {
		commitItem.Label = utils.ResolvePlaceholderString(commitItem.Label, map[string]string{"hash": ""})
		commitItem.DisabledReason = &types.DisabledReason{Text: tr.NoCommitSelected}
	} else {
		commitItem.Label = utils.ResolvePlaceholderString(commitItem.Label, map[string]string{"hash": commit.ShortHash()})
		commitItem.OnPress = func() error {
			return setFilter(func(filter *context.GithubRunsFilter) {
				if filter.HeadSha == commit.Hash() {
					filter.HeadSha = ""
				} else {
					filter.HeadSha = commit.Hash()
				}
			})
		}
	}

	withValue := func(label string, value string) string {
		if value == "" {
			return label
		}
		return label + " " + style.FgCyan.Sprint(value)
	}
	statusLabel := ""
	if status, found := lo.Find(helper.RunStatusFilters(), func(status helpers.GithubRunStatusFilter) bool {
		return status.Value == filter.Status
	}); found {
		statusLabel = status.Label
	}
	workflowName := ""
	if filter.Workflow != nil {
		workflowName = filter.Workflow.Name
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: tr.FilterRuns,
		Items: []*types.MenuItem{
			currentBranchItem,
			{
				Label:  withValue(tr.FilterByBranch, lo.Ternary(filter.Branch != currentBranchName, filter.Branch, "")),
				Widget: types.MakeMenuCheckBox(filter.Branch != "" && filter.Branch != currentBranchName),
				OnPress: func() error {
					self.c.Prompt(types.PromptOpts{
						Title:               tr.FilterBranchPrompt,
						InitialContent:      filter.Branch,
						FindSuggestionsFunc: self.c.Helpers().Suggestions.GetBranchNameSuggestionsFunc(),
						AllowEmptyInput:     true,
						HandleConfirm: func(branch string) error {
							return setFilter(func(filter *context.GithubRunsFilter) { filter.Branch = branch })
						},
					})
					return nil
				},
			},
			{
				Label:     withValue(tr.FilterByWorkflow, workflowName),
				Widget:    types.MakeMenuCheckBox(filter.Workflow != nil),
				OpensMenu: true,
				OnPress: func() error {
					return helper.WithWorkflows(func(workflows []*models.GithubWorkflow) error {
						return self.selectFilterValue(tr.FilterByWorkflow, tr.AnyWorkflow, workflowName,
							lo.Map(workflows, func(workflow *models.GithubWorkflow, _ int) [2]string {
								return [2]string{workflow.Name, workflow.Name}
							}),
							func(name string) error {
								return setFilter(func(filter *context.GithubRunsFilter) {
									workflow, _ := lo.Find(workflows, func(workflow *models.GithubWorkflow) bool {
										return workflow.Name == name
									})
									filter.Workflow = workflow
								})
							})
					})
				},
			},
			{
				Label:     withValue(tr.FilterByStatus, statusLabel),
				Widget:    types.MakeMenuCheckBox(filter.Status != ""),
				OpensMenu: true,
				OnPress: func() error {
					return self.selectFilterValue(tr.FilterByStatus, tr.AnyStatus, filter.Status,
						lo.Map(helper.RunStatusFilters(), func(status helpers.GithubRunStatusFilter, _ int) [2]string {
							return [2]string{status.Value, status.Label}
						}),
						func(status string) error {
							return setFilter(func(filter *context.GithubRunsFilter) { filter.Status = status })
						})
				},
			},
			{
				Label:     withValue(tr.FilterByEvent, filter.Event),
				Widget:    types.MakeMenuCheckBox(filter.Event != ""),
				OpensMenu: true,
				OnPress: func() error {
					events := []string{"push", "pull_request", "workflow_dispatch", "schedule", "release", "merge_group", "pull_request_target"}
					return self.selectFilterValue(tr.FilterByEvent, tr.AnyEvent, filter.Event,
						lo.Map(events, func(event string, _ int) [2]string { return [2]string{event, event} }),
						func(event string) error {
							return setFilter(func(filter *context.GithubRunsFilter) { filter.Event = event })
						})
				},
			},
			commitItem,
			{
				Label: tr.ClearRunsFilter,
				OnPress: func() error {
					return helper.SetRunsFilter(context.GithubRunsFilter{})
				},
				DisabledReason: lo.Ternary(filter.IsActive(), nil, &types.DisabledReason{Text: tr.NoRunsFilter}),
			},
		},
	})
}

// selectFilterValue offers a choice between "any" and each of the given
// value/label pairs, and calls onSelect with the chosen value, which is empty
// for "any".
func (self *GithubRunsController) selectFilterValue(title string, anyLabel string, current string, values [][2]string, onSelect func(string) error) error {
	items := make([]*types.MenuItem, 0, 1+len(values))
	items = append(items, &types.MenuItem{
		Label:   anyLabel,
		Widget:  types.MakeMenuRadioButton(current == ""),
		OnPress: func() error { return onSelect("") },
	})
	for _, value := range values {
		items = append(items, &types.MenuItem{
			Label:   value[1],
			Widget:  types.MakeMenuRadioButton(current == value[0]),
			OnPress: func() error { return onSelect(value[0]) },
		})
	}
	return self.c.Menu(types.CreateMenuOptions{Title: title, Items: items})
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
