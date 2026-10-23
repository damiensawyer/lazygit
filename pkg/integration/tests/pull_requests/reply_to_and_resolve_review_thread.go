package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ReplyToAndResolveReviewThread = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Reply to a conversation about lines of a pull request, and resolve it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests:       pullRequests(),
		PullRequestDetails: map[int]*models.GithubPullRequestDetails{12: widgetDetails()},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Press(keys.PullRequests.ReviewThreads)

		t.ExpectPopup().Menu().
			Title(Equals("Review threads of #12")).
			Lines(
				Contains("●").Contains("widget.go:7").Contains("bob Why a pointer?").IsSelected(),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Review threads of #12 › widget.go:7")).
			Select(Contains("Reply…")).
			Confirm()

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Reply on widget.go:7")).
			Type("So that frames can be shared").
			Confirm()

		t.ExpectToast(Equals("Replied"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReviewThreadReply {"input":{"body":"So that frames can be shared","pullRequestReviewThreadId":"T_1"}}`))

		t.Views().PullRequests().
			Press(keys.PullRequests.ReviewThreads)

		t.ExpectPopup().Menu().
			Title(Equals("Review threads of #12")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Review threads of #12 › widget.go:7")).
			Select(Contains("Resolve the conversation")).
			Confirm()

		t.ExpectToast(Equals("Resolved the conversation on widget.go:7"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReviewThreadResolve {"threadId":"T_1"}`))
	},
})
