package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CustomCommandOnPullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A custom command in the pull requests panel can use the selected pull request",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().CustomCommands = []config.CustomCommand{
			{
				Key:     config.Keybinding{"X"},
				Context: "pullRequests",
				Command: `printf "%s %s %s" {{.SelectedPullRequest.Number}} {{.SelectedPullRequest.HeadRefName | quote}} {{.SelectedPullRequest.Url | quote}} > selected-pr`,
			},
		}
	},
	SetupRepo: setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
			).
			Press(config.Keybinding{"X"})

		t.FileSystem().
			FileContent("selected-pr", Equals("12 widget https://github.com/owner/repo/pull/12"))
	},
})
