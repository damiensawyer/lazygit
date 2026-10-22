package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var GithubEnterpriseLogin = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Logging in to a GitHub Enterprise Server only offers pasting a token, since the browser login only exists on github.com",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Services = map[string]string{"git.acme.com": "github:git.acme.com"}
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://git.acme.com/owner/repo.git"})
	},
	FakeGithub: &FakeGithub{
		Errors: map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs": {StatusCode: 401, Message: "Bad credentials"},
		},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.ExpectPopup().Menu().
			Title(Equals("Log in to git.acme.com")).
			ContainsLines(
				Contains("Paste an authentication token"),
				Contains("Cancel"),
			)

		t.Views().Menu().
			Content(DoesNotContain("Log in with a web browser"))

		t.ExpectPopup().Menu().
			Title(Equals("Log in to git.acme.com")).
			Select(Contains("Paste an authentication token")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("GitHub token for git.acme.com")).
			Cancel()
	},
})
