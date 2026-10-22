package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RerunJob = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Re-run a single job of a completed run, optionally with debug logging",
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
			NavigateToLine(Contains("Fix the frobnicator")).
			PressEnter()

		t.Views().GithubJobs().
			IsFocused().
			NavigateToLine(Contains("test")).
			Press(keys.GithubActions.Rerun)

		t.ExpectPopup().Menu().
			Title(Equals("Re-run test")).
			Select(Equals("Re-run this job")).
			Confirm()

		t.ExpectToast(Equals("Re-run requested"))

		t.Views().GithubJobs().
			IsFocused().
			NavigateToLine(Contains("build")).
			Press(keys.GithubActions.Rerun)

		t.ExpectPopup().Menu().
			Title(Equals("Re-run build")).
			Select(Equals("Re-run this job with debug logging")).
			Confirm()

		t.ExpectToast(Equals("Re-run requested"))
		t.FakeGithub().ChangeRequests(
			Equals("POST /repos/owner/repo/actions/jobs/1002/rerun"),
			Equals(`POST /repos/owner/repo/actions/jobs/1001/rerun {"enable_debug_logging":true}`),
		)
	},
})
