package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CheckOutPullRequestInWorktree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Check out a pull request detached in a new worktree, going back from where to put the worktree to how to check it out",
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
			Press(keys.PullRequests.CheckOut)

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12")).
			Select(Contains("Check out in a new worktree...")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12 › In a new worktree")).
			Select(Contains("Check out as a branch named...")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Local branch name")).
			Clear().
			Type("review-widget").
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12 › In a new worktree › Worktree location"))

		// Back past the prompt, to the menu it was asked from
		t.Views().Menu().
			Press(keys.Universal.PrevBlock)

		t.Views().Menu().
			SelectedLines(Contains("Check out as a branch named..."))

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12 › In a new worktree")).
			Select(Contains("Check out detached")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Check out #12 › In a new worktree › Worktree location")).
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

		t.Shell().RunShellCommand(`! git rev-parse --verify --quiet review-widget`)
	},
})
