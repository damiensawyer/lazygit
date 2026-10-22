package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var NotLoggedIn = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Without a GitHub token, showing the pull requests asks for nothing until refreshing them, which offers to log in",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			IsEmpty()

		t.Views().Main().
			Content(Contains("Not logged in to GitHub. Press R to log in and load the pull requests."))

		t.Views().PullRequests().
			Press(keys.Universal.Refresh)

		t.ExpectPopup().Menu().
			Title(Equals("Log in to github.com")).
			Cancel()
	},
})
