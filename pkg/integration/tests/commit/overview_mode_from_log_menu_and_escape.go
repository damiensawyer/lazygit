package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeFromLogMenuAndEscape = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Enter overview mode from the log menu and leave it with escape",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.EmptyCommit("plain")
		shell.EmptyCommit("top")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Lines(
				Contains("top").IsSelected(),
				Contains("plain"),
				Contains("base"),
			).
			Press(keys.Commits.OpenLogMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Commit Log Options")).
			Select(Contains("Toggle overview mode")).
			Confirm()

		t.Views().Commits().
			IsFocused().
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("top").IsSelected(),
				Contains("base"),
			)

		t.Views().Information().Content(Contains("Showing commits overview"))

		t.Views().Commits().
			PressEscape().
			Title(Equals("Commits")).
			Lines(
				Contains("top").IsSelected(),
				Contains("plain"),
				Contains("base"),
			)

		t.Views().Information().Content(DoesNotContain("Showing commits overview"))

		// the log menu toggles the mode off again, too
		t.Views().Commits().
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("top").IsSelected(),
				Contains("base"),
			).
			Press(keys.Commits.OpenLogMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Commit Log Options")).
			Select(Contains("Toggle overview mode")).
			Confirm()

		t.Views().Commits().
			Title(Equals("Commits")).
			Lines(
				Contains("top").IsSelected(),
				Contains("plain"),
				Contains("base"),
			)
	},
})
