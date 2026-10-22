package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RunWorkflowWithoutDispatchTrigger = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Trying to run a workflow that has no workflow_dispatch trigger reports that it can't be run manually",
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
			Select(Contains("Run workflow")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Run CI on branch or tag")).
			Clear().
			Type("v1.0").
			Confirm()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("CI can't be run manually: it has no workflow_dispatch trigger")).
			Confirm()

		t.FakeGithub().
			ReceivedRequest(Equals("GET /repos/owner/repo/contents/.github/workflows/ci.yml?ref=v1.0")).
			ChangeRequests()
	},
})
