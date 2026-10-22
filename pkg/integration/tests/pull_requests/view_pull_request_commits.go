package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewPullRequestCommits = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Enter a pull request to see its commits and the files they change, without checking it out",
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
			RunCommand([]string{"git", "branch", "-D", "contribution"})
	},
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").Contains("Add a widget").IsSelected(),
			).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Title(Contains("#12 Add a widget")).
			Lines(
				Contains("test the widget").IsSelected(),
				Contains("add a widget"),
				Contains("initial commit"),
			).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Contains("widget_test.go"),
			).
			PressEscape()

		t.Views().SubCommits().
			IsFocused().
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
