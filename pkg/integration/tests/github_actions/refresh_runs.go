package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RefreshRuns = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Refreshing the runs shows new runs and runs that have progressed, keeping the selected run selected",
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
				Contains("* ").Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			).
			NavigateToLine(Contains("Fix the frobnicator"))

		t.FakeGithub().Update(func(fake *FakeGithub) {
			newRun := githubRun(105, "CI", ciWorkflowId, 14, "Bump the dependencies", models.GithubStatusQueued, "")
			fake.Runs = append([]*models.GithubRun{newRun}, fake.Runs...)
			fake.Runs[1].Status = models.GithubStatusCompleted
			fake.Runs[1].Conclusion = models.GithubConclusionSuccess
		})

		t.Views().GithubRuns().
			Press(keys.Universal.Refresh).
			Lines(
				Contains("• ").Contains("Bump the dependencies"),
				Contains("✓ ").Contains("Add a widget"),
				Contains("Fix the frobnicator").IsSelected(),
				Contains("Release v1.0"),
				Contains("Try something"),
			)
	},
})
