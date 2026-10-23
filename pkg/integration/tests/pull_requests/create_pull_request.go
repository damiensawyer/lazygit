package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CreatePullRequest = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Create a pull request for the checked out branch, which isn't pushed yet, starting from its commit's message",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("widget").
			EmptyCommit("Add a widget\n\nIt's useful.")
	},
	FakeGithub: &FakeGithub{},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().PullRequests().
			Focus().
			IsEmpty().
			Press(keys.Universal.New)

		t.ExpectPopup().Prompt().
			Title(Equals("Merge widget into")).
			InitialText(Equals("master")).
			Confirm()

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Pull request: widget into master")).
			InitialText(Equals("Add a widget")).
			Clear().
			Type("Add a frobnicating widget").
			SwitchToDescription().
			Content(Equals("It's useful.")).
			SwitchToSummary().
			Confirm()

		t.ExpectPopup().Menu().
			Title(Equals("Add a frobnicating widget")).
			ContainsLines(
				Contains("widget will be pushed to origin first."),
			).
			ContainsLines(
				Contains("Create pull request").IsSelected(),
				Contains("Create draft pull request"),
				Contains("Cancel"),
			).
			Confirm()

		t.ExpectToast(Equals("Created #1"))

		t.FakeGithub().ReceivedRequest(Equals(`POST /repos/owner/repo/pulls {"base":"master","body":"It's useful.","draft":false,"head":"widget","maintainer_can_modify":true,"title":"Add a frobnicating widget"}`))

		t.Views().PullRequests().
			Lines(
				Contains("#1").Contains("*").Contains("Add a frobnicating widget").IsSelected(),
			)

		t.Shell().RunShellCommand(`test "$(git rev-parse --abbrev-ref widget@{upstream})" = origin/widget`)
	},
})
