package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthResetHalfScreenMode = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Resetting the side panel width in half screen mode goes back to the configured half screen mode width",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.HalfScreenModeSidePanelWidth = 0.4
	},
	SetupRepo: func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.35"))

		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.ExpectToast(Equals("Reset side panel width"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(
			/* EXPECTED:
			Equals("Side panel width: 0.35"))
			ACTUAL: */
			Equals("Side panel width: 0.45"))
	},
})
