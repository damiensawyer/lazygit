package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthResetAfterReload = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Resetting the side panel width goes back to the configured width even after the saved width has been reloaded",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.28"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.23"))

		// Reload the config, which now has the saved width in it
		t.FocusIn()

		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.ExpectToast(Equals("Reset side panel width"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(
			/* EXPECTED:
			Equals("Side panel width: 0.28"))
			ACTUAL: */
			Equals("Side panel width: 0.18"))
	},
})
