package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OpenLogInEditor = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Open a job's log in the editor, from the jobs view at its start and from the steps view at the selected step",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().OS.EditAtLine = "cat {{filename}} > edited-log; echo {{line}} > edited-line"
	},
	SetupRepo:  setupGithubRepo,
	FakeGithub: newFakeGithub(),
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
			NavigateToLine(Contains("test")).
			Press(keys.Universal.Edit)

		t.FileSystem().
			FileContent("edited-log", Equals(
				"Preparing the test runner\n"+
					"##[group]Run make test\n"+
					"frobnicator_test.go:12: expected 1, got 2\n"+
					"##[error]Process completed with exit code 1.\n")).
			FileContent("edited-line", Equals("1\n"))

		t.Views().GithubJobs().
			PressEnter()

		t.Views().GithubSteps().
			IsFocused().
			Lines(
				Contains("Set up job"),
				Contains("Run tests").IsSelected(),
				Contains("Upload coverage"),
			).
			Press(keys.Universal.Edit)

		t.FileSystem().
			FileContent("edited-line", Equals("2\n"))
	},
})
