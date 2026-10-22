package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorUnauthorizedLoadingWorkflow = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "When GitHub rejects the token while loading a workflow to run it, the user is asked to log in again",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithubWithWorkflowFiles()
		fake.Errors = map[string]FakeGithubError{
			"GET /repos/owner/repo/contents/.github/workflows/ci.yml": {StatusCode: 401, Message: "Bad credentials"},
		}
		return fake
	}(),
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
			Type("master").
			Confirm()

		t.ExpectPopup().
			Menu().
			Title(Equals("Log in to github.com")).
			Cancel()
	},
})
