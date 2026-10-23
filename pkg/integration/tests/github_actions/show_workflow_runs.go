package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowWorkflowRuns = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show only the runs of a workflow chosen from the workflows menu",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub:   newFakeGithub(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.ViewWorkflows)

		t.ExpectPopup().Menu().
			Title(Equals("Workflows")).
			Select(Contains("Release")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Workflows › Release")).
			Select(Contains("Show only its runs")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · Release")).
			Lines(
				Contains("Release v1.0").IsSelected(),
			)
		t.FakeGithub().ReceivedRequest(Equals("GET /repos/owner/repo/actions/workflows/20/runs?per_page=50"))
	},
})
