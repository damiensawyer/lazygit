package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CheckOutPullRequestInWorktreeDirectly = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Check out a pull request in a new worktree with the new worktree key, without going through the check out menu",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("widget").
			EmptyCommit("add a widget").
			PushBranch("origin", "widget").
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "widget"})
	},
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Press(keys.Universal.NewWorktree)

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12 in a new worktree")).
			Select(Contains("Check out detached")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12 in a new worktree › Worktree location")).
			Confirm()

		t.Views().Branches().
			Lines(
				Contains("(no branch)"),
				Contains("master (worktree repo)"),
			)

		t.Views().Commits().
			Lines(
				Contains("add a widget"),
				Contains("initial commit"),
			)
	},
})
