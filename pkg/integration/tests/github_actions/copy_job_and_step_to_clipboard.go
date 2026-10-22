package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// We're emulating the clipboard by writing to a file called clipboard

var CopyJobAndStepToClipboard = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Copy a job's URL, ID and log, and a step's log, to the clipboard",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
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

		copy := func(item string, expectedToast string, expectedClipboard string) {
			t.Views().GithubJobs().
				IsFocused().
				Press(keys.GithubActions.CopyToClipboard)

			t.ExpectPopup().Menu().
				Title(Equals("Copy to clipboard")).
				Select(Equals(item)).
				Confirm()

			t.ExpectToast(Equals(expectedToast))
			t.FileSystem().FileContent("clipboard", Equals(expectedClipboard))
		}

		t.Views().GithubJobs().
			IsFocused().
			NavigateToLine(Contains("test")).
			Press(keys.GithubActions.CopyToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Lines(
				Equals("Job URL").IsSelected(),
				Equals("Job ID"),
				Equals("Log"),
				Contains("Cancel"),
			).
			Cancel()

		copy("Job URL", "Job URL copied to clipboard", "https://github.com/owner/repo/actions/runs/101/job/1002")
		copy("Job ID", "Job ID copied to clipboard", "1002")
		copy("Log", "Log copied to clipboard",
			"Preparing the test runner\n"+
				"##[group]Run make test\n"+
				"frobnicator_test.go:12: expected 1, got 2\n"+
				"##[error]Process completed with exit code 1.\n")

		t.Views().GithubJobs().
			PressEnter()

		t.Views().GithubSteps().
			IsFocused().
			Lines(
				Contains("Set up job"),
				Contains("Run tests").IsSelected(),
				Contains("Upload coverage"),
			).
			Press(keys.GithubActions.CopyToClipboard)

		t.ExpectToast(Equals("Log of this step copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals(
			"##[group]Run make test\n"+
				"frobnicator_test.go:12: expected 1, got 2\n"+
				"##[error]Process completed with exit code 1.\n"))
	},
})
