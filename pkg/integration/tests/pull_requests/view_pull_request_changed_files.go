package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewPullRequestChangedFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the files a pull request changes, without checking it out, leaving out what its base branch got since it branched off",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("contribution").
			CreateFileAndAdd("widget.go", "package widget\n").
			Commit("add a widget").
			CreateFileAndAdd("widget_test.go", "package widget\n").
			Commit("test the widget").
			RunCommand([]string{"git", "push", "origin", "contribution:refs/pull/12/head"}).
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "contribution"}).
			CreateFileAndAdd("changelog.md", "# Changelog\n").
			Commit("start a changelog").
			PushBranch("origin", "master").
			HardReset("HEAD^")
	},
	FakeGithub: &FakeGithub{
		PullRequests:       []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
		PullRequestDetails: map[int]*models.GithubPullRequestDetails{12: {}},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		baseHead := t.Git().GetCommitHash("origin/master")
		t.FakeGithub().Update(func(fake *FakeGithub) {
			fake.PullRequestDetails[12].BaseRefOid = baseHead
		})

		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").Contains("Add a widget").IsSelected(),
			).
			Press(keys.PullRequests.ViewChangedFiles)

		t.Views().CommitFiles().
			IsFocused().
			Title(Contains("#12 Add a widget")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  A widget.go"),
				Equals("  A widget_test.go"),
			).
			PressEscape()

		t.Views().PullRequests().
			IsFocused()

		// the pull request wasn't checked out
		t.Views().Branches().
			Lines(
				Contains("master"),
			)
	},
})
