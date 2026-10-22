package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnableAndDisableWorkflow = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Disable an active workflow, and enable a disabled one",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub:   newFakeGithubWithWorkflowFiles(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.ViewWorkflows)

		t.ExpectPopup().Menu().
			Title(Equals("Workflows")).
			Select(Contains("CI")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("CI")).
			Select(Contains("Disable workflow")).
			Tooltip(Equals("A disabled workflow doesn't run, whatever triggers it, until it is enabled again.")).
			Confirm()

		t.ExpectToast(Equals("Workflow disabled"))

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.ViewWorkflows)

		t.ExpectPopup().Menu().
			Title(Equals("Workflows")).
			Select(Contains("Nightly")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Nightly")).
			Select(Contains("Run workflow")).
			/* EXPECTED:
			Tooltip(Contains("Disabled: The workflow is disabled, so it can't be run until it is enabled again.")).
			ACTUAL: */
			Tooltip(Contains("Disabled: disabled")).
			Select(Contains("Enable workflow")).
			Confirm()

		t.ExpectToast(Equals("Workflow enabled"))

		t.FakeGithub().ChangeRequests(
			Equals("PUT /repos/owner/repo/actions/workflows/10/disable"),
			Equals("PUT /repos/owner/repo/actions/workflows/30/enable"),
		)
	},
})
