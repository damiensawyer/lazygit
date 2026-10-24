package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewMergedPullRequestChangedFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the files a pull request that was merged with a merge commit changes, though its base branch has them all by now",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("contribution").
			CreateFileAndAdd("widget.go", "package widget\n").
			Commit("add a widget").
			RunCommand([]string{"git", "push", "origin", "contribution:refs/pull/12/head"}).
			Checkout("master").
			CreateFileAndAdd("changelog.md", "# Changelog\n").
			Commit("start a changelog").
			Merge("contribution").
			RunCommand([]string{"git", "branch", "-D", "contribution"}).
			PushBranch("origin", "master")
	},
	FakeGithub: &FakeGithub{
		PullRequests:       []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "MERGED", "widget")},
		PullRequestDetails: map[int]*models.GithubPullRequestDetails{12: {}},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// where the base branch was when the pull request was merged
		baseWhenMerged := t.Git().GetCommitHash("master^1")
		t.FakeGithub().Update(func(fake *FakeGithub) {
			fake.PullRequestDetails[12].BaseRefOid = baseWhenMerged
		})

		t.Views().PullRequests().
			Focus().
			Press(keys.PullRequests.Filter)

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests")).
			Select(Contains("State").Contains("Open")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests › State")).
			Select(Contains("Merged")).
			Confirm()

		t.Views().PullRequests().
			Lines(
				Contains("#12").Contains("Add a widget").IsSelected(),
			).
			Press(keys.PullRequests.ViewChangedFiles)

		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Equals("A widget.go").IsSelected(),
			)
	},
})
