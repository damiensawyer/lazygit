package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeLoadsWholeHistory = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Enabling overview mode loads commits beyond the commit limit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		// more than the 300 commits that are loaded initially, with the only
		// tagged commit beyond them
		shell.EmptyCommit("tagged root")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.CreateNCommits(310)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("commit-310").IsSelected(),
				Contains("some-tag").Contains("tagged root"),
			)
	},
})
