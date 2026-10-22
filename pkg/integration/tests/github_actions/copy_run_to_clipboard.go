package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// We're emulating the clipboard by writing to a file called clipboard

var CopyRunToClipboard = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Copy a run's URL, ID, commit hash and branch name to the clipboard",
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

		copy := func(item string, expectedToast string, expectedClipboard string) {
			t.Views().GithubRuns().
				IsFocused().
				Press(keys.GithubActions.CopyToClipboard)

			t.ExpectPopup().Menu().
				Title(Equals("Copy to clipboard")).
				Select(Equals(item)).
				Confirm()

			t.ExpectToast(Equals(expectedToast))
			t.FileSystem().FileContent("clipboard", Equals(expectedClipboard))
		}

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Fix the frobnicator")).
			Press(keys.GithubActions.CopyToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Lines(
				Equals("Run URL").IsSelected(),
				Equals("Run ID"),
				Equals("Commit hash"),
				Equals("Branch name"),
				Contains("Cancel"),
			).
			Cancel()

		copy("Run URL", "Run URL copied to clipboard", "https://github.com/owner/repo/actions/runs/101")
		copy("Run ID", "Run ID copied to clipboard", "101")
		copy("Commit hash", "Commit hash copied to clipboard", "0123456789abcdef0123456789abcdef01234567")
		copy("Branch name", "Branch name copied to clipboard", "main")
	},
})
