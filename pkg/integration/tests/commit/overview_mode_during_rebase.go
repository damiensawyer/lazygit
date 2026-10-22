package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeDuringRebase = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Overview mode keeps showing rebase todos, and follows the rebase when it continues",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateNCommits(5)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("commit-03")).
			Press(keys.Universal.Edit).
			Lines(
				Contains("─── Pending rebase todos"),
				Contains("pick").Contains("commit-05"),
				Contains("pick").Contains("commit-04"),
				Contains("─── Commits"),
				Contains("commit-03").IsSelected(),
				Contains("commit-02"),
				Contains("commit-01"),
			).
			Press(keys.Commits.ToggleOverviewMode).
			// the todos stay, and so does the commit we stopped at, which HEAD
			// points to
			Lines(
				Contains("─── Pending rebase todos"),
				Contains("pick").Contains("commit-05"),
				Contains("pick").Contains("commit-04"),
				Contains("─── Commits"),
				Contains("commit-03").IsSelected(),
			).
			NavigateToLine(Contains("commit-04")).
			Press(keys.Commits.MoveUpCommit)

		t.ExpectPopup().Confirmation().
			Title(Equals("Command not available")).
			Content(Equals("Command not available in overview mode. Exit overview mode?")).
			Cancel()

		t.Views().Commits().
			Lines(
				Contains("─── Pending rebase todos"),
				Contains("pick").Contains("commit-05"),
				Contains("pick").Contains("commit-04").IsSelected(),
				Contains("─── Commits"),
				Contains("commit-03"),
			)

		t.Common().ContinueRebase()

		t.Views().Commits().
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("commit-05"),
			).
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("commit-05"),
				Contains("commit-04"),
				Contains("commit-03"),
				Contains("commit-02"),
				Contains("commit-01"),
			)
	},
})
