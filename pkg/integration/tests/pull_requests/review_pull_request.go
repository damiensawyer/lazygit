package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ReviewPullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Approve a pull request, request changes to it with a comment, and comment on it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Press(keys.PullRequests.Review)

		t.ExpectPopup().Menu().
			Title(Equals("Review #12")).
			Lines(
				Contains("Approve").IsSelected(),
				Contains("Approve with a comment"),
				Contains("Request changes"),
				Contains("Comment without approving"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectToast(Equals("Reviewed #12"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReviewAdd {"input":{"event":"APPROVE","pullRequestId":"PR_12"}}`))

		t.Views().PullRequests().
			Press(keys.PullRequests.Review)

		t.ExpectPopup().Menu().
			Title(Equals("Review #12")).
			Select(Contains("Request changes")).
			Confirm()

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Request changes")).
			Type("Please add tests").
			SwitchToDescription().
			Type("They should cover the empty widget.").
			SwitchToSummary().
			Confirm()

		t.ExpectToast(Equals("Reviewed #12"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReviewAdd {"input":{"body":"Please add tests\n\nThey should cover the empty widget.","event":"REQUEST_CHANGES","pullRequestId":"PR_12"}}`))

		t.Views().PullRequests().
			Press(keys.PullRequests.Comment)

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Comment on #12")).
			Type("Nice work").
			Confirm()

		t.ExpectToast(Equals("Commented on #12"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql CommentCreate {"input":{"body":"Nice work","subjectId":"PR_12"}}`))
	},
})
