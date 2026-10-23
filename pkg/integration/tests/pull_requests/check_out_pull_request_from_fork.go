package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CheckOutPullRequestFromFork = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Check out a pull request from a fork that has no remote, fetching it from the base repository and setting the branch up to push to the fork, since its maintainers may",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("contribution").
			EmptyCommit("document the frobnicator").
			RunCommand([]string{"git", "push", "origin", "contribution:refs/pull/11/head"}).
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "contribution"})
	},
	FakeGithub: func() *FakeGithub {
		pr := pullRequest(11, "Document the frobnicator", "contributor", "OPEN", "docs")
		pr.IsCrossRepository = true
		pr.MaintainerCanModify = true
		pr.HeadRepositoryOwner.Login = "contributor"
		pr.HeadRepositoryName = "repo-fork"
		return &FakeGithub{PullRequests: []*models.GithubPullRequest{pr}}
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#11").Contains("Document the frobnicator").IsSelected(),
			).
			Press(keys.PullRequests.CheckOut)

		t.ExpectPopup().Menu().
			Title(Equals("Check out #11")).
			TopLines(
				Contains("Check out").IsSelected(),
			).
			Confirm()

		t.ExpectToast(Equals("Checked out #11 as docs"))

		t.Views().Commits().
			Lines(
				Contains("document the frobnicator"),
				Contains("initial commit"),
			)

		t.Shell().
			RunShellCommand(`test "$(git branch --show-current)" = docs`).
			RunShellCommand(`test "$(git config branch.docs.remote)" = https://github.com/contributor/repo-fork.git`).
			RunShellCommand(`test "$(git config branch.docs.pushRemote)" = https://github.com/contributor/repo-fork.git`).
			RunShellCommand(`test "$(git config branch.docs.merge)" = refs/heads/docs`)
	},
})
