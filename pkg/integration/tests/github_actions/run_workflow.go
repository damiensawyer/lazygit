package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RunWorkflow = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Run a workflow that has a workflow_dispatch trigger, choosing the ref and entering its inputs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub:   newFakeGithubWithWorkflowFiles(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.ViewWorkflows)

		t.ExpectPopup().Menu().
			Title(Equals("Workflows")).
			Lines(
				Contains("CI").Contains(".github/workflows/ci.yml").IsSelected(),
				Contains("Release").Contains(".github/workflows/release.yml"),
				Contains("Nightly").Contains("disabled").Contains(".github/workflows/nightly.yml"),
				Contains("Cancel"),
			).
			Select(Contains("Release")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Release")).
			Lines(
				Contains("Run workflow").IsSelected(),
				Contains("Show only its runs"),
				Contains("Disable workflow"),
				Contains("Edit workflow file"),
				Contains("Open in browser"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Run Release on branch or tag")).
			InitialText(Equals("master")).
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("environment: Where to deploy")).
			Lines(
				Equals("(•) staging").IsSelected(),
				Equals("( ) production"),
				Contains("Cancel"),
			).
			Select(Contains("production")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("version: Version to release")).
			Type("1.2.3").
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("dry_run")).
			Lines(
				Equals("( ) true").IsSelected(),
				Equals("(•) false"),
				Contains("Cancel"),
			).
			Select(Contains("true")).
			Confirm()

		t.ExpectToast(Equals("Workflow run requested"))
		t.FakeGithub().ChangeRequests(
			Equals(`POST /repos/owner/repo/actions/workflows/20/dispatches {"inputs":{"dry_run":"true","environment":"production","version":"1.2.3"},"ref":"master"}`),
		)
		t.FakeGithub().ReceivedRequest(Equals("GET /repos/owner/repo/contents/.github/workflows/release.yml?ref=master"))
	},
})
