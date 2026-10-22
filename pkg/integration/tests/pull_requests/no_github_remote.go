package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var NoGithubRemote = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "In a repository without a GitHub remote, the pull requests panel says that there are none to show",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://gitlab.com/owner/repo.git"})
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			IsEmpty()

		t.Views().Main().
			Content(Contains("This repository has no GitHub remote, so there are no pull requests to show."))
	},
})
