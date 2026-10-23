package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var MergePullRequestDeletingBranch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Merge the checked out pull request and delete its branch, on GitHub and locally, checking out its base branch first",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("widget").
			EmptyCommit("add a widget").
			PushBranchAndSetUpstream("origin", "widget")
	},
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").Contains("*").IsSelected(),
			).
			Press(keys.PullRequests.Merge)

		t.ExpectPopup().Menu().
			Title(Equals("Merge #12 into master")).
			Select(Contains("Create a merge commit")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Merge #12 into master › Merge #12 Add a widget")).
			Lines(
				Contains("Merge, and delete the branch along with the local branch widget").IsSelected(),
				Contains("Merge, keeping the branch"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectToast(Equals("Merged #12"))

		t.FakeGithub().ReceivedRequest(Contains(`POST /graphql PullRequestMerge`).Contains(`"mergeMethod":"MERGE"`))
		t.FakeGithub().ReceivedRequest(Equals("DELETE /repos/owner/repo/git/refs/heads/widget"))

		t.Views().Branches().
			Lines(
				Contains("master"),
			)
	},
})
