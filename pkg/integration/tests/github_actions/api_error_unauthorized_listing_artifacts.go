package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorUnauthorizedListingArtifacts = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "When GitHub rejects the token while listing a run's artifacts, the user is asked to log in again, and asked again if they try again without having",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Artifacts = map[int64][]*models.GithubArtifact{
			failedRunId: {{ArtifactId: 1, Name: "dist", SizeInBytes: 2048}},
		}
		fake.ArtifactFiles = map[int64]map[string]string{1: {"bin/app": "the app"}}
		fake.Errors = map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs/101/artifacts": {StatusCode: 401, Message: "Bad credentials"},
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
			Press(keys.GithubActions.DownloadArtifacts)

		t.ExpectPopup().
			Menu().
			Title(Equals("Log in to github.com")).
			Cancel()

		// Trying again without having logged in asks again.
		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.DownloadArtifacts)

		t.ExpectPopup().
			Menu().
			Title(Equals("Log in to github.com")).
			Cancel()
	},
})
