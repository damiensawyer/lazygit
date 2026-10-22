package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorRepoNotFound = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "When GitHub answers that the repository doesn't exist, which it also does for a private one the token can't see, that is explained",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		Errors: map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs": {StatusCode: 404, Message: "Not Found"},
		},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("Repository owner/repo not found on GitHub, or your token has no access to it")).
			Confirm()

		t.Views().Files().
			IsFocused()
	},
})
