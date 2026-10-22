package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthInPopup = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The side panel width keybindings do nothing while a popup is open",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFile("file", "content")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Press(keys.Files.ViewStashOptions)

		t.ExpectPopup().Menu().Title(Equals("Stash options"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.ExpectPopup().Menu().Title(Equals("Stash options")).Cancel()

		t.FileSystem().PathNotPresent(".git/lazygit.yml")

		t.Views().Files().IsFocused()
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.28"))
	},
})
