package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var GithubEnterprise = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the runs of a repository on a GitHub Enterprise Server configured as a hosting service, where there is no browser login",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Services = map[string]string{"git.acme.com": "github:git.acme.com"}
		config.GetUserConfig().OS.OpenLink = "echo {{link}} > opened-link"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://git.acme.com/owner/repo.git"})
	},
	FakeGithub: newFakeGithub(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo")).
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			).
			Press(keys.GithubActions.ViewWorkflows)

		t.ExpectPopup().Menu().
			Title(Equals("Workflows")).
			Select(Contains("CI")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Workflows › CI")).
			Select(Contains("Open in browser")).
			Confirm()

		t.FileSystem().
			FileContent("opened-link", Equals("https://git.acme.com/owner/repo/actions/workflows/ci.yml\n"))
	},
})
