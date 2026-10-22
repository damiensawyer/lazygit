package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorLoadingRuns = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A server error while loading the runs is reported, and the runs aren't shown",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Errors = map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs": {StatusCode: 500, Message: "Server Error"},
		}
		return fake
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("GitHub API request failed: HTTP 500: Server Error")).
			Confirm()

		t.Views().Files().
			IsFocused()
	},
})
