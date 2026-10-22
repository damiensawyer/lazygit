package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SelectRepository = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With several GitHub remotes, none of them called upstream, ask which repository's runs to show, and remember the choice",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.com/me/repo.git"})
		shell.RunCommand([]string{"git", "remote", "add", "team", "https://github.com/owner/repo.git"})
	},
	FakeGithub: newFakeGithub(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.ExpectPopup().Menu().
			Title(Equals("Select the repository whose Actions to show")).
			Lines(
				Contains("origin").Contains("me/repo").IsSelected(),
				Contains("team").Contains("owner/repo"),
				Contains("Cancel"),
			).
			Select(Contains("team")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo"))
		t.FakeGithub().ReceivedRequest(Equals("GET /repos/owner/repo/actions/runs?per_page=50"))

		// remembered the way `gh repo set-default` does it
		t.Shell().RunShellCommand(`test "$(git config remote.team.gh-resolved)" = base`)
	},
})
