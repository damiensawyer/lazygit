package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterRunsByBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Filter the runs by the current branch, or by a branch typed in",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupGithubRepo(shell)
		shell.NewBranch("main")
	},
	FakeGithub: newFakeGithub(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			).
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs on the current branch (main)")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · branch main")).
			Lines(
				Contains("Fix the frobnicator").IsSelected(),
				Contains("Release v1.0"),
			)
		t.FakeGithub().ReceivedRequest(Equals("GET /repos/owner/repo/actions/runs?branch=main&per_page=50"))

		// Choosing the current branch again turns the filter off
		t.Views().GithubRuns().
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs on the current branch (main)")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo")).
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			).
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs on branch")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Branch")).
			Type("experiment").
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · branch experiment")).
			Lines(
				Contains("Try something").IsSelected(),
			)
	},
})
