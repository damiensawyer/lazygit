package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeWithNoVisibleCommits = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Overview mode copes with hiding every commit, which filtering by path can make happen",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Git.Log.ShowGraph = "always"
	},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CreateFileAndAdd("filterFile", "1")
		shell.Commit("filterFile 1")
		shell.UpdateFileAndAdd("filterFile", "2")
		shell.Commit("filterFile 2")
		shell.EmptyCommit("top")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.FilteringMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Filtering")).
			Select(Contains("Enter path to filter by")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Enter path:")).
			Type("filterFile").
			Confirm()

		t.Views().Commits().
			IsFocused().
			Lines(
				Contains("filterFile 2").IsSelected(),
				Contains("filterFile 1"),
			).
			NavigateToLine(Contains("filterFile 1")).
			Press(keys.Commits.ToggleOverviewMode).
			Title(Equals("Commits (overview)")).
			IsEmpty().
			Press(keys.Commits.ToggleOverviewMode).
			Title(Equals("Commits")).
			Lines(
				Contains("filterFile 2"),
				Contains("filterFile 1").IsSelected(),
			)
	},
})
