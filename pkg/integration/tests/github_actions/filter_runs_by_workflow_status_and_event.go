package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterRunsByWorkflowStatusAndEvent = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Filter the runs by workflow, status and event, combining the filters, and then show all runs again",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub:   newFakeGithub(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Show all runs")).
			Confirm()

		t.ExpectToast(Equals("Disabled: The runs aren't filtered"))

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs of workflow")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Runs of workflow")).
			Lines(
				Contains("Any workflow").IsSelected(),
				Contains("CI"),
				Contains("Release"),
				Contains("Cancel"),
			).
			Select(Contains("CI")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · CI")).
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Try something"),
			).
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs with status")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Runs with status")).
			Lines(
				Contains("Any status").IsSelected(),
				Contains("Failed"),
				Contains("In progress"),
				Contains("Queued"),
				Contains("Waiting for approval"),
				Contains("Succeeded"),
				Contains("Cancelled"),
				Contains("Cancel"),
			).
			Select(Contains("Cancelled")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · CI, Cancelled")).
			Lines(
				Contains("Try something").IsSelected(),
			)
		t.FakeGithub().ReceivedRequest(Equals("GET /repos/owner/repo/actions/workflows/10/runs?per_page=50&status=cancelled"))

		t.Views().GithubRuns().
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Runs triggered by")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Runs triggered by")).
			Select(Equals("( ) pull_request")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo · CI, Cancelled, on pull_request")).
			IsEmpty()

		t.Views().Main().
			Content(Equals("No workflow runs"))

		t.Views().GithubRuns().
			Press(keys.GithubActions.FilterRuns)

		t.ExpectPopup().Menu().
			Title(Equals("Filter runs")).
			Select(Contains("Show all runs")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · owner/repo")).
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			)
	},
})
