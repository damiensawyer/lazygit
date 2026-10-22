package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EditPullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Edit a pull request's title and description, add a label to it, and request a review",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
		PullRequestDetails: map[int]*models.GithubPullRequestDetails{12: {
			Body:          "It's useful.",
			ViewerCanEdit: true,
		}},
		Labels:          []models.GithubLabel{{Name: "bug"}, {Name: "enhancement"}},
		AssignableUsers: []string{"alice", "bob"},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Press(keys.PullRequests.Edit)

		t.ExpectPopup().Menu().
			Title(Equals("Edit #12")).
			Select(Contains("Edit the title and description")).
			Confirm()

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Edit #12")).
			InitialText(Equals("Add a widget")).
			Type(" to the frame").
			SwitchToDescription().
			Content(Equals("It's useful.")).
			SwitchToSummary().
			Confirm()

		t.ExpectToast(Equals("Updated #12"))
		t.FakeGithub().ReceivedRequest(Equals(`PATCH /repos/owner/repo/pulls/12 {"body":"It's useful.","title":"Add a widget to the frame"}`))

		t.Views().PullRequests().
			Press(keys.PullRequests.Edit)

		t.ExpectPopup().Menu().
			Title(Equals("Edit #12")).
			Select(Contains("Labels")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Labels")).
			Lines(
				Contains("bug").IsSelected(),
				Contains("enhancement"),
				Contains("Cancel"),
			).
			Select(Contains("enhancement")).
			Confirm()

		t.FakeGithub().ReceivedRequest(Equals(`POST /repos/owner/repo/issues/12/labels {"labels":["enhancement"]}`))

		t.Views().PullRequests().
			Press(keys.PullRequests.Edit)

		t.ExpectPopup().Menu().
			Title(Equals("Edit #12")).
			Select(Contains("Reviewers")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Reviewers")).
			Select(Contains("Request a review from…")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Request a review from (a login, or org/team)")).
			Type("ali").
			SuggestionLines(Contains("alice")).
			ConfirmFirstSuggestion()

		t.ExpectToast(Equals("Requested a review from alice"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /repos/owner/repo/pulls/12/requested_reviewers {"reviewers":["alice"],"team_reviewers":[]}`))
	},
})
