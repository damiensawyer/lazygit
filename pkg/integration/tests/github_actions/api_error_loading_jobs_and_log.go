package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ApiErrorLoadingJobsAndLog = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Failing to load a run's jobs or a job's log is shown in the main view, and reported when drilling into the jobs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Errors = map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs/104/jobs":  {StatusCode: 500, Message: "Server Error"},
			"GET /repos/owner/repo/actions/jobs/1001/logs": {StatusCode: 410, Message: "Gone"},
		}
		return fake
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Try something"))

		t.Views().Main().
			ContainsLines(
				Equals("Jobs"),
				Equals("GitHub API request failed: HTTP 500: Server Error"),
			)

		t.Views().GithubRuns().
			PressEnter()

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("GitHub API request failed: HTTP 500: Server Error")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Fix the frobnicator")).
			PressEnter()

		t.Views().GithubJobs().
			IsFocused().
			Lines(
				Contains("build").IsSelected(),
				Contains("test"),
			)

		t.Views().Main().
			Content(Contains("GitHub API request failed: HTTP 410: Gone"))
	},
})
