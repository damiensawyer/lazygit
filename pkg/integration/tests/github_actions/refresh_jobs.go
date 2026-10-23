package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RefreshJobs = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refreshing the jobs of a run in progress shows how they have progressed",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Jobs[inProgressRunId] = []*models.GithubJob{
			{JobId: 2001, RunId: inProgressRunId, Name: "lint", Status: "in_progress", StartedAt: at(0, 0)},
		}
		return fake
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Add a widget")).
			PressEnter()

		t.Views().GithubJobs().
			IsFocused().
			Lines(
				Contains("* lint").IsSelected(),
			)

		t.FakeGithub().Update(func(fake *FakeGithub) {
			fake.Jobs[inProgressRunId][0].Status = models.GithubStatusCompleted
			fake.Jobs[inProgressRunId][0].Conclusion = models.GithubConclusionSuccess
			fake.Jobs[inProgressRunId][0].CompletedAt = at(1, 0)
		})

		t.Views().GithubJobs().
			Press(keys.Universal.Refresh).
			Lines(
				Contains("✓ lint").IsSelected(),
			)
	},
})
