package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RunWorkflowOnRefWithoutIt = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Trying to run a workflow on a branch or tag that doesn't have it reports that",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithubWithWorkflowFiles()
		fake.Errors = map[string]FakeGithubError{
			"GET /repos/owner/repo/contents/.github/workflows/ci.yml": {StatusCode: 404, Message: "Not Found"},
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
			Type("no-such-branch").
			Confirm()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals(".github/workflows/ci.yml doesn't exist at no-such-branch")).
			Confirm()
	},
})
