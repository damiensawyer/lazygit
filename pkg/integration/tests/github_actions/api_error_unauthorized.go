package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorUnauthorized = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "When GitHub rejects the token, e.g. because it has been revoked, the user is asked to log in again",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		Errors: map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs": {StatusCode: 401, Message: "Bad credentials"},
		},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.ExpectPopup().Menu().
			Title(Equals("Log in to github.com")).
			ContainsLines(
				Contains("Log in with a web browser"),
				Contains("Paste an authentication token"),
				Contains("Cancel"),
			).
			Cancel()

		t.Views().Files().
			IsFocused()
	},
})
