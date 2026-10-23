package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var KeybindingsSearch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Find a keybinding by typing in the keybindings menu",
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
		t.SetCaptionPrefix("Find a keybinding by typing")

		t.Views().Files().
			IsFocused().
			Wait(1000).
			Press(keys.Universal.OptionMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Wait(1500).
			Filter("stash").
			ContainsLines(
				Contains("Stash"),
			).
			Wait(2000)

		for range len("stash") {
			t.GlobalPress(config.Keybinding{"<backspace>"})
		}

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Wait(600).
			Filter("rebase").
			ContainsLines(
				Contains("rebase"),
			).
			Wait(2000)

		t.GlobalPress(keys.Universal.Return)

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Wait(1000)

		t.GlobalPress(keys.Universal.Return)

		t.Views().Files().
			IsFocused().
			Wait(1000)
	},
})
