package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeInFilteringMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Combine overview mode with filtering by path, and leave both with escape",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("filterFile", "1")
		shell.Commit("filterFile 1")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.CreateFileAndAdd("otherFile", "1")
		shell.Commit("otherFile 1")
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

		// the HEAD commit doesn't touch the file, so no commit above the
		// selected one stays visible and the selection moves to the top
		t.Views().Commits().
			IsFocused().
			Lines(
				Contains("filterFile 2").IsSelected(),
				Contains("filterFile 1"),
			).
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("filterFile 1").IsSelected(),
			)

		t.Views().Information().Content(Contains("Filtering by 'filterFile'"))

		// escape leaves filtering mode first, keeping overview mode on
		t.Views().Commits().
			PressEscape().
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("top"),
				Contains("filterFile 1").IsSelected(),
			)

		t.Views().Information().Content(Contains("Showing commits overview"))

		// the selection is still where entering overview mode moved it, so
		// leaving the mode jumps back to the commit that was selected then
		t.Views().Commits().
			PressEscape().
			Title(Equals("Commits")).
			Lines(
				Contains("top"),
				Contains("filterFile 2").IsSelected(),
				Contains("otherFile 1"),
				Contains("filterFile 1"),
			)

		t.Views().Information().Content(DoesNotContain("Showing commits overview"))
	},
})
