package filter_and_search

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterMenuInlineFilterEscape = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Filter the keybindings menu by typing directly, then clear the filter with escape",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().IsFocused().
			Press(keys.Universal.OptionMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings"))

		// Typing directly filters the menu, without needing the search prompt
		t.GlobalPress(config.Keybinding{"I"})
		t.GlobalPress(config.Keybinding{"g"})

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			Lines(
				// menu has filtered down to the one item that matches the filter
				Contains(`─── Local`),
				Contains(`Ignore or exclude file...`).IsSelected(),
			)

		// Escape should clear the filter, not close the menu
		t.GlobalPress(keys.Universal.Return)
		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			ContainsLines(
				Contains(`Ignore or exclude file...`),
				// an item that did not match the filter is visible again
				Contains("Refresh"),
			)

		// Another escape closes the menu
		t.GlobalPress(keys.Universal.Return)
		t.Views().Files().IsFocused()
	},
})
