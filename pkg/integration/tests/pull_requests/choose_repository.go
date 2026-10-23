package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ChooseRepository = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "With several GitHub remotes, none of them called upstream, the pull requests load once the user has chosen whose to show",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("initial commit")
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.com/me/repo.git"})
		shell.RunCommand([]string{"git", "remote", "add", "team", "https://github.com/owner/repo.git"})
	},
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			IsEmpty()

		t.Views().Main().
			Content(Contains("There are several GitHub remotes. Press R to choose the one whose pull requests to show."))

		t.Views().PullRequests().
			Press(keys.Universal.Refresh)

		t.ExpectPopup().Menu().
			Title(Equals("Which GitHub repository is this one's?")).
			Select(Contains("team")).
			Confirm()

		t.Views().PullRequests().
			Lines(
				Contains("#12").IsSelected(),
			)
		t.FakeGithub().ReceivedRequest(Contains(`POST /graphql PullRequestList`).Contains(`"owner":"owner","repo":"repo"`))
	},
})
