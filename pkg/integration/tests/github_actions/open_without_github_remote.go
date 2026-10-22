package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OpenWithoutGithubRemote = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Opening the GitHub Actions view in a repo whose remotes aren't on GitHub reports that",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://gitlab.com/owner/repo.git"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("This repository has no GitHub remote")).
			Confirm()

		t.Views().Files().IsFocused()
	},
})
