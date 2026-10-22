package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FullFileDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Toggle a diff between its usual context and the whole file",
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
		t.SetCaptionPrefix("Show the whole file in a diff")

		t.Views().Files().
			Focus().
			Wait(1000).
			NavigateToLine(MatchesRegexp(`store$`)).
			Wait(1500).
			Press(keys.Universal.ToggleContextInDiffView).
			Tap(func() {
				t.ExpectToast(Equals("Showing full file diff"))
				t.Views().Main().
					Content(Contains("@@ -1,")).
					Content(Contains("Home    string"))
			}).
			Wait(2500).
			NavigateToLine(Contains("store.go")).
			Wait(2000).
			SetCaptionPrefix("Back to the usual context").
			Press(keys.Universal.ToggleContextInDiffView).
			Tap(func() {
				t.ExpectToast(Equals("Restored context size to 3"))
				t.Views().Main().
					Content(DoesNotContain("package store"))
			}).
			Wait(2000)
	},
})
