package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewJobsAndSteps = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Drill into a run's jobs and a job's steps, reading their logs, and escape back out",
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
			PressEnter()

		t.Views().GithubJobs().
			IsFocused().
			Title(Equals("Jobs · Fix the frobnicator")).
			Lines(
				Contains("✓ build").IsSelected(),
				Contains("X test"),
			)

		t.Views().Main().
			Title(Equals("Job")).
			ContainsLines(
				Equals("Log"),
				Equals("━━ Set up job"),
				Equals("Preparing the runner"),
				Equals("━━ Build"),
				Equals("▾ Run make build"),
				Equals("built it"),
			)

		t.Views().GithubJobs().
			NavigateToLine(Contains("X test"))

		t.Views().Main().
			ContainsLines(
				Equals("Log of failed steps"),
			).
			Content(DoesNotContain("Preparing the test runner"))

		t.Views().GithubJobs().
			PressEnter()

		t.Views().GithubSteps().
			IsFocused().
			Title(Equals("Steps · test")).
			Lines(
				Contains("✓ Set up job"),
				Contains("X Run tests").IsSelected(),
				Contains("- Upload coverage"),
			)

		t.Views().Main().
			Title(Equals("Step")).
			Content(Contains("frobnicator_test.go:12: expected 1, got 2")).
			Content(Contains("Error: Process completed with exit code 1."))

		t.Views().GithubSteps().
			NavigateToLine(Contains("Set up job"))

		t.Views().Main().
			Content(Contains("Preparing the test runner"))

		t.Views().GithubSteps().
			NavigateToLine(Contains("Upload coverage"))

		t.Views().Main().
			Content(Contains("This step logged nothing"))

		t.Views().GithubSteps().
			PressEscape()

		t.Views().GithubJobs().
			IsFocused().
			PressEscape()

		t.Views().GithubRuns().
			IsFocused().
			PressEscape()

		t.Views().Commits().
			IsFocused()
	},
})
