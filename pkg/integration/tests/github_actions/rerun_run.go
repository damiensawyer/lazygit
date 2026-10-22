package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RerunRun = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Re-run the failed jobs or all jobs of a completed run, optionally with debug logging",
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
			Lines(
				Contains("Add a widget").IsSelected(),
				Contains("Fix the frobnicator"),
				Contains("Release v1.0"),
				Contains("Try something"),
			).
			Press(keys.GithubActions.Rerun)

		t.ExpectToast(Equals("Disabled: The run is still in progress"))

		t.Views().GithubRuns().
			NavigateToLine(Contains("Fix the frobnicator")).
			Press(keys.GithubActions.Rerun)

		t.ExpectPopup().Menu().
			Title(Equals("Re-run CI #12")).
			Lines(
				Contains("Re-run failed jobs").IsSelected(),
				Contains("Re-run all jobs"),
				Contains("Re-run failed jobs with debug logging"),
				Contains("Re-run all jobs with debug logging"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectToast(Equals("Re-run requested"))
		t.FakeGithub().ChangeRequests(
			Equals("POST /repos/owner/repo/actions/runs/101/rerun-failed-jobs"),
		)

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.Rerun)

		t.ExpectPopup().Menu().
			Title(Equals("Re-run CI #12")).
			Select(Equals("Re-run all jobs with debug logging")).
			Confirm()

		t.ExpectToast(Equals("Re-run requested"))
		t.FakeGithub().ChangeRequests(
			Equals("POST /repos/owner/repo/actions/runs/101/rerun-failed-jobs"),
			Equals(`POST /repos/owner/repo/actions/runs/101/rerun {"enable_debug_logging":true}`),
		)

		// A run that succeeded has no failed jobs to re-run
		t.Views().GithubRuns().
			NavigateToLine(Contains("Release v1.0")).
			Press(keys.GithubActions.Rerun)

		t.ExpectPopup().Menu().
			Title(Equals("Re-run Release #3")).
			Select(Equals("Re-run failed jobs")).
			Confirm()

		t.ExpectToast(Equals("Disabled: The run has no failed jobs"))

		t.ExpectPopup().Menu().
			Title(Equals("Re-run Release #3")).
			Select(Equals("Re-run all jobs")).
			Confirm()

		t.ExpectToast(Equals("Re-run requested"))
		t.FakeGithub().ChangeRequests(
			Equals("POST /repos/owner/repo/actions/runs/101/rerun-failed-jobs"),
			Equals(`POST /repos/owner/repo/actions/runs/101/rerun {"enable_debug_logging":true}`),
			Equals("POST /repos/owner/repo/actions/runs/103/rerun"),
		)
	},
})
