package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeRangeWithHiddenAnchor = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Enabling overview mode while the anchor of a range selection gets hidden doesn't extend the range to other commits",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CreateLightweightTag("base-tag", "HEAD")
		shell.EmptyCommit("plain-1")
		shell.EmptyCommit("plain-2")
		shell.EmptyCommit("top")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("plain-2")).
			Press(keys.Universal.RangeSelectUp).
			Lines(
				Contains("top").IsSelected(),
				Contains("plain-2").IsSelected(),
				Contains("plain-1"),
				Contains("base"),
			).
			Press(keys.Commits.ToggleOverviewMode).
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("top").IsSelected(),
				Contains("base"),
			)
	},
})
