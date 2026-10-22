package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CancelRun = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Cancel or force cancel a run that is in progress",
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
			NavigateToLine(Contains("Fix the frobnicator")).
			Press(keys.GithubActions.CancelRun)

		t.ExpectToast(Equals("Disabled: The run has already completed"))

		t.Views().GithubRuns().
			NavigateToLine(Contains("Add a widget")).
			Press(keys.GithubActions.CancelRun)

		t.ExpectPopup().Menu().
			Title(Equals("Cancel CI #13")).
			Select(Equals("Cancel run")).
			Confirm()

		t.ExpectToast(Equals("Cancellation requested"))

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.CancelRun)

		t.ExpectPopup().Menu().
			Title(Equals("Cancel CI #13")).
			Select(Equals("Force cancel run")).
			Confirm()

		t.ExpectToast(Equals("Cancellation requested"))
		t.FakeGithub().ChangeRequests(
			Equals("POST /repos/owner/repo/actions/runs/102/cancel"),
			Equals("POST /repos/owner/repo/actions/runs/102/force-cancel"),
		)
	},
})
