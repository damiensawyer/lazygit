package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OpenDiffLineInBrowser = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Open the selected line of the diff of a pull request entered from the pull requests panel in the pull request on GitHub",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Gui.UseHunkModeInDiffView = false
		config.GetUserConfig().OS.OpenLink = "echo {{link}} > opened-link"
	},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("contribution").
			CreateFileAndAdd("widget.go", "package widget\n\nfunc New() {}\n").
			Commit("add a widget").
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
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("add a widget").IsSelected(),
				Contains("initial commit"),
			).
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			NavigateToLine(Contains("+func New() {}")).
			Press(keys.Commits.OpenPullRequestInBrowser)

		t.FileSystem().
			FileContent("opened-link", Contains("https://github.com/owner/repo/pull/12/changes/").Contains("R3\n"))
	},
})
