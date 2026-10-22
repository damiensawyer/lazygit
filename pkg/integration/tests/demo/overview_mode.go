package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Collapse the commits to the merges and the commits that refs point to",
	ExtraCmdArgs: []string{},
	Skip:         false,
	IsDemo:       true,
	SetupConfig: func(config *config.AppConfig) {
		setFleetDemoConfig(config)
	},
	SetupRepo: func(shell *Shell) {
		setupFleetRepo(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.SetCaptionPrefix("Show an overview of the commits")

		t.Views().Commits().
			Focus().
			Wait(1500).
			NavigateToLine(Contains("Add CI badge")).
			Wait(1000).
			Press(keys.Commits.ToggleOverviewMode).
			Title(Equals("Commits (overview)")).
			SelectedLine(Contains("Merge branch 'readme-badges'")).
			Wait(2500).
			SelectNextItem().
			Wait(700).
			SelectNextItem().
			Wait(700).
			SelectNextItem().
			SelectedLine(Contains("Merge branch 'crew-validation'")).
			Wait(1500).
			SetCaptionPrefix("Back to all commits").
			Press(keys.Commits.ToggleOverviewMode).
			Title(Equals("Commits")).
			Wait(2500)
	},
})
