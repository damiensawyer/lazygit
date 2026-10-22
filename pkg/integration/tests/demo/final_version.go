package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FinalVersion = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the final version of changed files instead of their diffs",
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
		t.SetCaptionPrefix("Show the final version of changed files")

		t.Views().Files().
			Focus().
			Wait(1000).
			NavigateToLine(MatchesRegexp(`store$`)).
			Wait(1500).
			Press(keys.Universal.ToggleShowFinalVersion).
			Tap(func() {
				t.ExpectToast(Equals("Showing final version of files"))
			}).
			Wait(2500).
			NavigateToLine(Contains("store.go")).
			Tap(func() {
				t.Views().Main().
					Title(Equals("Final version")).
					Content(Contains("Home    string"))
			}).
			Wait(2500).
			NavigateToLine(Contains("server.go")).
			Tap(func() {
				t.Views().Main().
					Title(Equals("Final version")).
					Content(Contains("package server"))
			}).
			Wait(2000).
			SetCaptionPrefix("Back to the diffs").
			Press(keys.Universal.ToggleShowFinalVersion).
			Tap(func() {
				t.ExpectToast(Equals("Showing diffs"))
				t.Views().Main().
					Title(Equals("Unstaged changes"))
			}).
			Wait(2000)
	},
})
