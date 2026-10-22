package controllers

import (
	goContext "context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// GithubWorkflowsAction offers what `gh workflow` does: running a workflow,
// enabling and disabling it, and showing its runs.
type GithubWorkflowsAction struct {
	c *ControllerCommon
}

func (self *GithubWorkflowsAction) Call() error {
	return self.c.Helpers().GithubActions.WithWorkflows(func(workflows []*models.GithubWorkflow) error {
		return self.c.Menu(types.CreateMenuOptions{
			Title: self.c.Tr.GithubActions.Workflows,
			Items: lo.Map(workflows, func(workflow *models.GithubWorkflow, _ int) *types.MenuItem {
				state := ""
				if !workflow.IsActive() {
					state = style.FgYellow.Sprint(self.c.Tr.GithubActions.WorkflowDisabled)
				}
				return &types.MenuItem{
					LabelColumns: []string{workflow.Name, state, style.FgBlue.Sprint(workflow.Path)},
					OpensMenu:    true,
					OnPress:      func() error { return self.workflowMenu(workflow) },
				}
			}),
		})
	})
}

func (self *GithubWorkflowsAction) workflowMenu(workflow *models.GithubWorkflow) error {
	tr := self.c.Tr.GithubActions
	helper := self.c.Helpers().GithubActions

	toggleItem := &types.MenuItem{
		Label:   tr.DisableWorkflow,
		Tooltip: tr.DisableWorkflowTooltip,
		OnPress: func() error {
			return helper.RunAction(tr.DisablingWorkflow, tr.WorkflowDisabledToast, func(ctx goContext.Context, client *github_commands.Client) error {
				return client.DisableWorkflow(ctx, workflow.WorkflowId)
			})
		},
	}
	if !workflow.IsActive() {
		toggleItem = &types.MenuItem{
			Label: tr.EnableWorkflow,
			OnPress: func() error {
				return helper.RunAction(tr.EnablingWorkflow, tr.WorkflowEnabled, func(ctx goContext.Context, client *github_commands.Client) error {
					return client.EnableWorkflow(ctx, workflow.WorkflowId)
				})
			},
		}
	}

	runItem := &types.MenuItem{
		Label:   tr.RunWorkflow,
		Tooltip: tr.RunWorkflowTooltip,
		OnPress: func() error { return self.promptForRef(workflow) },
	}
	if !workflow.IsActive() {
		runItem.DisabledReason = &types.DisabledReason{Text: tr.CantRunDisabledWorkflow}
	}

	editItem := &types.MenuItem{
		Label: tr.EditWorkflowFile,
		OnPress: func() error {
			return self.c.Helpers().Files.EditFiles([]string{self.localPath(workflow)})
		},
	}
	if _, err := os.Stat(self.localPath(workflow)); err != nil {
		editItem.DisabledReason = &types.DisabledReason{
			Text: utils.ResolvePlaceholderString(tr.WorkflowFileNotInWorktree, map[string]string{"path": workflow.Path}),
		}
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: workflow.Name,
		Items: []*types.MenuItem{
			runItem,
			{
				Label: tr.ShowWorkflowRuns,
				OnPress: func() error {
					filter := self.c.Contexts().GithubRuns.GetRunsFilter()
					filter.Workflow = workflow
					return helper.SetRunsFilter(filter)
				},
			},
			toggleItem,
			editItem,
			{
				Label: tr.OpenWorkflowInBrowser,
				OnPress: func() error {
					repo := self.c.Helpers().GithubClient.Client().Repo()
					return self.c.OS().OpenLink(fmt.Sprintf("https://%s/%s/actions/workflows/%s",
						repo.Host, repo.FullName(), path.Base(workflow.Path)))
				},
			},
		},
	})
}

func (self *GithubWorkflowsAction) localPath(workflow *models.GithubWorkflow) string {
	return filepath.Join(self.c.Git().RepoPaths.WorktreePath(), filepath.FromSlash(workflow.Path))
}

func (self *GithubWorkflowsAction) promptForRef(workflow *models.GithubWorkflow) error {
	ref := ""
	if branch := self.c.Helpers().Refs.GetCheckedOutRef(); branch != nil && !branch.DetachedHead {
		ref = branch.Name
	}

	self.c.Prompt(types.PromptOpts{
		Title: utils.ResolvePlaceholderString(self.c.Tr.GithubActions.RunWorkflowRefPrompt, map[string]string{
			"workflow": workflow.Name,
		}),
		InitialContent:      ref,
		FindSuggestionsFunc: self.c.Helpers().Suggestions.GetRefsSuggestionsFunc(),
		HandleConfirm: func(ref string) error {
			return self.loadInputs(workflow, ref)
		},
	})
	return nil
}

// loadInputs finds out which inputs the workflow takes at the given ref, from
// its definition there.
func (self *GithubWorkflowsAction) loadInputs(workflow *models.GithubWorkflow, ref string) error {
	tr := self.c.Tr.GithubActions
	client := self.c.Helpers().GithubClient.Client()

	return self.c.WithWaitingStatus(tr.LoadingWorkflow, func(gocui.Task) error {
		content, err := client.GetFileContent(goContext.Background(), workflow.Path, ref)
		var inputs []github_commands.WorkflowInput
		if err == nil {
			inputs, err = github_commands.ParseWorkflowDispatchInputs(content)
		}

		self.c.OnUIThread(func() error {
			if errors.Is(err, github_commands.ErrNotDispatchable) {
				return errors.New(utils.ResolvePlaceholderString(tr.NotDispatchable, map[string]string{
					"workflow": workflow.Name,
				}))
			}
			if err != nil {
				return err
			}
			return self.askForInputs(workflow, ref, inputs, map[string]string{})
		})
		return nil
	})
}

// askForInputs asks for the value of each input in turn, and then runs the
// workflow with them.
func (self *GithubWorkflowsAction) askForInputs(workflow *models.GithubWorkflow, ref string, inputs []github_commands.WorkflowInput, values map[string]string) error {
	if len(inputs) == 0 {
		return self.dispatch(workflow, ref, values)
	}

	input := inputs[0]
	next := func(value string) error {
		values[input.Name] = value
		return self.askForInputs(workflow, ref, inputs[1:], values)
	}

	title := utils.ResolvePlaceholderString(self.c.Tr.GithubActions.WorkflowInputPrompt, map[string]string{"name": input.Name})
	if input.Description != "" {
		title = utils.ResolvePlaceholderString(self.c.Tr.GithubActions.WorkflowInputPromptWithDescription, map[string]string{
			"name":        input.Name,
			"description": input.Description,
		})
	}

	options := input.Options
	if input.Type == "boolean" {
		options = []string{"true", "false"}
	}
	if len(options) > 0 {
		return self.c.Menu(types.CreateMenuOptions{
			Title: title,
			Items: lo.Map(options, func(option string, _ int) *types.MenuItem {
				return &types.MenuItem{
					Label:   option,
					Widget:  types.MakeMenuRadioButton(option == input.Default),
					OnPress: func() error { return next(option) },
				}
			}),
		})
	}

	self.c.Prompt(types.PromptOpts{
		Title:           title,
		InitialContent:  input.Default,
		AllowEmptyInput: !input.Required,
		HandleConfirm:   next,
	})
	return nil
}

func (self *GithubWorkflowsAction) dispatch(workflow *models.GithubWorkflow, ref string, inputs map[string]string) error {
	tr := self.c.Tr.GithubActions
	helper := self.c.Helpers().GithubActions

	return helper.RunAction(tr.StartingWorkflow, tr.WorkflowStarted, func(ctx goContext.Context, client *github_commands.Client) error {
		if err := client.DispatchWorkflow(ctx, workflow.WorkflowId, ref, inputs); err != nil {
			return err
		}
		self.c.OnUIThread(func() error {
			helper.ExpectNewRun()
			return nil
		})
		return nil
	})
}
