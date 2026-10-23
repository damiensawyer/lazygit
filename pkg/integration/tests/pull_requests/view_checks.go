package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewChecks = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "List a pull request's checks, re-run its failed GitHub Actions jobs, and open a failed one in the GitHub Actions view",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests:       pullRequests(),
		PullRequestDetails: map[int]*models.GithubPullRequestDetails{12: widgetDetails()},
		Runs: []*models.GithubRun{{
			RunId: 101, Name: "CI", DisplayTitle: "Add a widget", Number: 7, Status: "completed", Conclusion: "failure",
			HeadBranch: "widget", Event: "pull_request", CreatedAt: at(2), UpdatedAt: at(2),
		}},
		Jobs: map[int64][]*models.GithubJob{101: {
			{JobId: 1001, RunId: 101, Name: "build", Status: "completed", Conclusion: "success"},
			{JobId: 1002, RunId: 101, Name: "test", Status: "completed", Conclusion: "failure"},
		}},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
				Contains("#10"),
			).
			Press(keys.PullRequests.ViewChecks)

		t.ExpectPopup().Menu().
			Title(Equals("Checks of #12")).
			Lines(
				Contains("X").Contains("CI / test (required)").IsSelected(),
				Contains("✓").Contains("CI / build"),
				Contains("Re-run the failed GitHub Actions jobs"),
				Contains("Open the checks in the browser"),
				Contains("Cancel"),
			).
			Select(Contains("Re-run the failed GitHub Actions jobs")).
			Confirm()

		t.ExpectToast(Equals("Requested re-runs of 1 workflow run(s)"))
		t.FakeGithub().ReceivedRequest(Equals("POST /repos/owner/repo/actions/runs/101/rerun-failed-jobs"))

		t.Views().PullRequests().
			Press(keys.PullRequests.ViewChecks)

		t.ExpectPopup().Menu().
			Title(Equals("Checks of #12")).
			Select(Contains("CI / test")).
			Confirm()

		t.Views().GithubJobs().
			IsFocused().
			Lines(
				Contains("build"),
				Contains("test").IsSelected(),
			).
			PressEscape()

		t.Views().GithubRuns().
			IsFocused().
			Lines(
				Contains("Add a widget").IsSelected(),
			)
	},
})
