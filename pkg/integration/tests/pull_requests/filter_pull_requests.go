package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterPullRequests = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Choose which pull requests to load: the viewer's own, then merged ones, then all open ones again",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: &FakeGithub{
		PullRequests: append(pullRequests(), pullRequest(9, "Remove the old frobnicator", "me", "MERGED", "remove")),
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
				Contains("#10"),
			).
			Press(keys.PullRequests.Filter)

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests")).
			Select(Contains("Created by me")).
			Confirm()

		t.FakeGithub().ReceivedRequest(Contains(`POST /graphql PullRequestSearch`).Contains(`"q":"repo:owner/repo is:pr sort:updated-desc is:open author:@me"`))

		t.Views().PullRequests().
			Lines(
				Contains("#11").Contains("Document the frobnicator").IsSelected(),
			).
			Press(keys.PullRequests.Filter)

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests")).
			Select(Contains("State").Contains("Open")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests › State")).
			Select(Contains("Merged")).
			Confirm()

		t.Views().PullRequests().
			Lines(
				Contains("#9").Contains("Remove the old frobnicator").IsSelected(),
			).
			Press(keys.PullRequests.Filter)

		t.ExpectPopup().Menu().
			Title(Equals("Filter pull requests")).
			Select(Contains("Show all open pull requests")).
			Confirm()

		t.Views().PullRequests().
			Lines(
				Contains("#12").IsSelected(),
				Contains("#11"),
				Contains("#10"),
			)
	},
})
