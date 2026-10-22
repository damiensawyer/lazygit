package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// The hash of the commit that SetupRepo creates, which is the same every time
// because the commit's author, date and message are.
const filteredCommitHash = "c5547a6e32eccb04ae1f120a44dda2608b494105"

var FilterRunsByCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Filter the runs by the commit selected in the commits panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommitWithDate("initial commit", "2024-03-01T09:00:00Z")
		shell.EmptyCommitWithDate("second commit", "2024-03-01T09:30:00Z")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.com/owner/repo.git"})
	},
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Runs[1].HeadSha = filteredCommitHash
		return fake
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("initial commit")).
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs for the commit selected in the commits panel (" + filteredCommitHash[:8] + ")")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · commit " + filteredCommitHash[:8])).
			Lines(
				Contains("Fix the frobnicator").IsSelected(),
			)
		t.FakeGithub().ReceivedRequest(Equals("GET /repos/owner/repo/actions/runs?head_sha=" + filteredCommitHash + "&per_page=50"))
	},
})
