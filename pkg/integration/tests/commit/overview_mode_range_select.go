package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeRangeSelect = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A range selected in overview mode diffs the hidden commits within it too, and escape cancels the range before the mode",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("base-file", "base")
		shell.Commit("base")
		shell.CreateLightweightTag("base-tag", "HEAD")
		shell.CreateFileAndAdd("hidden-file", "hidden")
		shell.Commit("hidden")
		shell.CreateFileAndAdd("tagged-file", "tagged")
		shell.Commit("tagged")
		shell.CreateLightweightTag("tagged-tag", "HEAD")
		shell.EmptyCommit("top")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("top"),
				Contains("tagged"),
				Contains("base"),
			).
			NavigateToLine(Contains("tagged")).
			Press(keys.Universal.RangeSelectDown).
			Lines(
				Contains("top"),
				Contains("tagged").IsSelected(),
				Contains("base").IsSelected(),
			)

		t.Views().Main().
			Content(
				Contains("base-file").
					Contains("hidden-file").
					Contains("tagged-file"),
			)

		t.Views().Commits().
			PressEscape().
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("top"),
				Contains("tagged"),
				Contains("base").IsSelected(),
			).
			PressEscape().
			Title(Equals("Commits")).
			Lines(
				Contains("top"),
				Contains("tagged"),
				Contains("hidden"),
				Contains("base").IsSelected(),
			)
	},
})
