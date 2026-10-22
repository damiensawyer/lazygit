package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var MergePullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Squash and merge a pull request, keeping its branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
		PullRequestDetails: map[int]*models.GithubPullRequestDetails{12: {
			Repository: models.GithubRepositorySettings{
				DefaultBranch:       "master",
				MergeCommitAllowed:  true,
				SquashMergeAllowed:  true,
				RebaseMergeAllowed:  false,
				AutoMergeAllowed:    true,
				ViewerDefaultMethod: "SQUASH",
			},
		}},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
			).
			Press(keys.PullRequests.Merge)

		t.ExpectPopup().Menu().
			Title(Equals("Merge #12 into master")).
			Lines(
				Contains("Create a merge commit").IsSelected(),
				Contains("Squash and merge (your default)"),
				Contains("Rebase and merge"),
				Contains("Enable auto-merge..."),
				Contains("Cancel"),
			).
			Select(Contains("Rebase and merge")).
			Confirm()

		t.ExpectToast(Equals("Disabled: The repository doesn't allow merging this way."))

		t.ExpectPopup().Menu().
			Title(Equals("Merge #12 into master")).
			Select(Contains("Squash and merge")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Merge #12 Add a widget")).
			Lines(
				Contains("Merge, keeping the branch").IsSelected(),
				Contains("Merge, and delete the branch"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectToast(Equals("Merged #12"))

		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestMerge {"input":{"expectedHeadOid":"0123456789abcdef0123456789abcdef01234567","mergeMethod":"SQUASH","pullRequestId":"PR_12"}}`))
	},
})
