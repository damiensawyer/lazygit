package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidth = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Adjusting side panel width persists the setting",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Enter half screen mode (starts at 0.5 default)
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.Views().Status().Content(Contains("master"))

		// Adjust the side panel width (0.5 -> 0.45)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Contains("Side panel width: 0.45"))

		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Contains("Side panel width: 0.50"))

		// Reset to default
		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.ExpectToast(Contains("Reset side panel width"))
	},
})
