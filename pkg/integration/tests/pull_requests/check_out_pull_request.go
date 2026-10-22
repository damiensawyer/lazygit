package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CheckOutPullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Check out a pull request from a branch of the repository, on a new local branch tracking it, and then again once it has a new commit, fast-forwarding that branch",
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
			Lines(
				Contains("#12").DoesNotContain("*").Contains("Add a widget").IsSelected(),
			).
			Press(keys.PullRequests.CheckOut)

		t.ExpectToast(Equals("Checked out #12 as widget"))

		t.Views().PullRequests().
			Lines(
				Contains("#12").Contains("*").Contains("Add a widget").IsSelected(),
			)

		t.Views().Branches().
			Lines(
				Contains("widget").Contains("✓"),
				Contains("master"),
			)

		t.Views().Commits().
			Lines(
				Contains("add a widget"),
				Contains("initial commit"),
			)

		t.Shell().RunShellCommand(`test "$(git rev-parse --abbrev-ref widget@{upstream})" = origin/widget`)

		// a new commit is pushed to the pull request
		t.Shell().
			Checkout("master").
			NewBranchFrom("other", "widget").
			EmptyCommit("polish the widget").
			PushBranch("origin", "other:widget").
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "other"}).
			RunCommand([]string{"git", "update-ref", "refs/remotes/origin/widget", "widget"})

		t.Views().PullRequests().
			Focus().
			Press(keys.PullRequests.CheckOut)

		t.ExpectToast(Equals("Checked out #12 as widget"))

		t.Views().Commits().
			Lines(
				Contains("polish the widget"),
				Contains("add a widget"),
				Contains("initial commit"),
			)
	},
})
