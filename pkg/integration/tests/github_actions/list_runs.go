package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ListRuns = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Opening the GitHub Actions view lists the repository's workflow runs, whatever state they are in",
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
			Title(Equals("Actions · owner/repo")).
			Lines(
				Contains("* ").Contains("CI").Contains("Add a widget").IsSelected(),
				Contains("X ").Contains("CI").Contains("Fix the frobnicator"),
				Contains("✓ ").Contains("Release").Contains("Release v1.0"),
				Contains("X ").Contains("CI").Contains("Try something"),
			).
			NavigateToLine(Contains("Fix the frobnicator"))

		t.Views().Main().
			Title(Equals("Run")).
			ContainsLines(
				Equals("X Fix the frobnicator"),
				Equals("CI #12 · main 01234567 · .github/workflows/ci.yml"),
			).
			ContainsLines(
				Equals("Jobs"),
				Contains("✓ build 1m0s"),
				Contains("X test 2m0s"),
				Equals("  ✓ Set up job"),
				Equals("  X Run tests"),
			).
			ContainsLines(
				Equals("Annotations"),
				Equals("test"),
				Equals("  X expected 1, got 2"),
				Equals("    frobnicator_test.go#12"),
			)
	},
})
