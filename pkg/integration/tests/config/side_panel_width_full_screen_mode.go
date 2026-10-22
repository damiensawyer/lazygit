package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthFullScreenMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The side panel width keybindings do nothing in full screen mode",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.GlobalPress(keys.Universal.PrevScreenMode)
		t.Views().Files().IsFocused()

		// No toast is shown (the test would fail on an unacknowledged toast) and
		// nothing is saved
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.FileSystem().PathNotPresent(".git/lazygit.yml")

		// Neither width was changed
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.28"))

		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.45"))
	},
})
