package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DeleteRun = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Delete a completed run, after confirming it",
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
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			).
			Press(keys.Universal.Remove)

		t.ExpectToast(Equals("Disabled: The run is still in progress"))

		t.Views().GithubRuns().
			NavigateToLine(Contains("Try something")).
			Press(keys.Universal.Remove)

		t.ExpectPopup().Confirmation().
			Title(Equals("Delete run")).
			Content(Equals("Delete CI #11 (Try something) along with its logs and artifacts? This cannot be undone.")).
			Cancel()

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.Universal.Remove)

		t.ExpectPopup().Confirmation().
			Title(Equals("Delete run")).
			Content(Contains("Delete CI #11 (Try something)")).
			Confirm()

		t.ExpectToast(Equals("Run deleted"))
		t.FakeGithub().ChangeRequests(
			Equals("DELETE /repos/owner/repo/actions/runs/104"),
		)
	},
})
