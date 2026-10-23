package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ListPullRequests = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the open pull requests in a tab of the branches panel, and the selected one's details in the main view",
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
			Lines(
				Contains("#12").Contains("✗").Contains("±").Contains("octocat").Contains("Add a widget").IsSelected(),
				Contains("#11").Contains("●").Contains("me").Contains("Document the frobnicator"),
				Contains("#10").Contains("✓").Contains("✔").Contains("hubot").Contains("Fix the frobnicator"),
			)

		t.FakeGithub().ReceivedRequest(Contains(`POST /graphql PullRequestList`).Contains(`"owner":"owner","repo":"repo","states":["OPEN"]`))

		t.Views().Main().
			Title(Equals("Pull request")).
			ContainsLines(
				Contains("Add a widget  #12"),
			).
			ContainsLines(
				Contains("octocat wants to merge widget into master"),
			).
			ContainsLines(
				Contains("Assignees: me"),
			).
			ContainsLines(
				Contains("Reviewers:").Contains("bob").Contains("alice"),
			).
			ContainsLines(
				Contains("Checks: 1 failing, 1 passing"),
			).
			ContainsLines(
				Contains("Merge: blocked"),
			).
			ContainsLines(
				Equals("This adds a widget."),
				Equals(""),
				Equals("Fixes #3."),
			).
			ContainsLines(
				Contains("Failing checks"),
				Contains("CI / test (required)"),
			).
			ContainsLines(
				Contains("2 file(s) changed in 2 commit(s)"),
				Contains("A widget.go +30 -0"),
				Contains("M frame.go +10 -3"),
			).
			ContainsLines(
				Contains("carol commented"),
				Equals("  Looks useful"),
			).
			ContainsLines(
				Contains("bob requested changes"),
				Equals("  Please add tests"),
			).
			ContainsLines(
				Contains("Review threads (1 of 1 unresolved)"),
			).
			ContainsLines(
				Equals("widget.go:7"),
				Equals("   package widget"),
				Equals("  +"),
				Equals("  +type Widget struct {"),
				Contains("  + frame *Frame"),
				Contains("  bob"),
				Equals("    Why a pointer?"),
			)

		t.Views().PullRequests().
			NavigateToLine(Contains("#10"))

		t.Views().Main().
			ContainsLines(
				Contains("hubot wants to merge fix into master"),
			).
			ContainsLines(
				Equals("No description provided."),
			)
	},
})
