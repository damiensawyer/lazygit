package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ListNoRuns = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Opening the GitHub Actions view of a repository that has no workflow runs says so",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub:   &FakeGithub{},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo")).
			IsEmpty()

		t.Views().Main().
			Content(Equals("No workflow runs"))
	},
})
