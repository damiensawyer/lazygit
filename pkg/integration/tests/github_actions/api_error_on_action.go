package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorOnAction = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "An action that GitHub refuses, like re-running a run without permission to, is reported",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Errors = map[string]FakeGithubError{
			"POST /repos/owner/repo/actions/runs/101/rerun": {StatusCode: 403, Message: "Must have admin rights to Repository."},
		}
		return fake
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Fix the frobnicator")).
			Press(keys.GithubActions.Rerun)

		t.ExpectPopup().Menu().
			Title(Equals("Re-run CI #12")).
			Select(Equals("Re-run all jobs")).
			Confirm()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("GitHub API request failed: HTTP 403: Must have admin rights to Repository.")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused()
	},
})
