package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CloseAndReopenPullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Close an open pull request, mark a draft as ready for review, and reopen a closed pull request",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{
			pullRequest(12, "Add a widget", "octocat", "OPEN", "widget"),
			pullRequest(11, "Document the frobnicator", "me", "DRAFT", "docs"),
		},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
			).
			Press(keys.PullRequests.ChangeState)

		t.ExpectPopup().Menu().
			Title(Equals("#12 Add a widget")).
			Lines(
				Contains("Close the pull request").IsSelected(),
				Contains("Convert to draft"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Close the pull request")).
			Content(Equals("Close #12 Add a widget without merging it?")).
			Confirm()

		t.ExpectToast(Equals("Closed #12"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestClose {"prID":"PR_12"}`))

		t.Views().PullRequests().
			NavigateToLine(Contains("#11")).
			Press(keys.PullRequests.ChangeState)

		t.ExpectPopup().Menu().
			Title(Equals("#11 Document the frobnicator")).
			Select(Contains("Mark as ready for review")).
			Confirm()

		t.ExpectToast(Equals("Marked #11 as ready for review"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReadyForReview {"prID":"PR_11"}`))

		// a closed one can be reopened
		t.FakeGithub().Update(func(fake *FakeGithub) {
			fake.PullRequests[0].State = "CLOSED"
		})
		t.Views().PullRequests().
			Press(keys.PullRequests.Filter)

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests")).
			Select(Contains("State")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("State")).
			Select(Contains("Closed")).
			Confirm()

		t.Views().PullRequests().
			Lines(
				Contains("#12").IsSelected(),
			).
			Press(keys.PullRequests.ChangeState)

		t.ExpectPopup().Menu().
			Title(Equals("#12 Add a widget")).
			Lines(
				Contains("Reopen the pull request").IsSelected(),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectToast(Equals("Reopened #12"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReopen {"prID":"PR_12"}`))
	},
})
