package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeRefresh = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Overview mode stays on across a refresh, re-deciding which commits to show",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.EmptyCommit("plain")
		shell.EmptyCommit("top")
		shell.CreateFile("file", "content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("top"),
				Contains("base"),
			)

		t.Views().Files().
			Focus().
			PressPrimaryAction().
			Press(keys.Files.CommitChanges)

		t.ExpectPopup().CommitMessagePanel().
			Type("new top").
			Confirm()

		// the new commit is HEAD now, so the previous one is hidden
		t.Views().Commits().
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("new top"),
				Contains("base"),
			)

		t.Views().Information().Content(Contains("Showing commits overview"))

		// a tag created outside lazygit shows up after a refresh, too
		t.Shell().CreateLightweightTag("other-tag", "HEAD~2")
		t.GlobalPress(keys.Universal.Refresh)

		t.Views().Commits().
			Lines(
				Contains("new top"),
				Contains("other-tag").Contains("plain"),
				Contains("base"),
			)
	},
})
