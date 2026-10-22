package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidth = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Resize the side panels from the keyboard",
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
		resize := func(key config.Keybinding, times int, pause int) {
			for range times {
				t.Views().Files().
					Press(key).
					Tap(func() {
						t.ExpectToast(Contains("Side panel width: "))
					}).
					Wait(pause)
			}
		}

		t.SetCaptionPrefix("Resize the side panels")

		t.Views().Files().
			IsFocused().
			Wait(1000)

		resize(keys.Universal.IncreaseSidePanelWidth, 3, 400)
		t.Wait(600)
		resize(keys.Universal.DecreaseSidePanelWidth, 5, 400)
		t.Wait(600)

		t.SetCaptionPrefix("Each screen mode keeps its own width")

		t.Views().Files().
			Press(keys.Universal.NextScreenMode).
			Wait(1500)

		resize(keys.Universal.IncreaseSidePanelWidth, 2, 400)
		t.Wait(600)

		t.Views().Files().
			Press(keys.Universal.PrevScreenMode).
			Wait(2000)
	},
})
