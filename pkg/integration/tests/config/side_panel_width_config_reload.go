package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthConfigReload = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Editing the side panel widths in the config and refocusing the window applies them live",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Shell().CreateFile(".git/lazygit.yml", `
gui:
  sidePanelWidth: 0.5
  halfScreenModeSidePanelWidth: 0.6
`)
		t.FocusIn()

		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		/* EXPECTED:
		t.ExpectToast(Equals("Side panel width: 0.45"))
		ACTUAL: */
		t.ExpectToast(Equals("Side panel width: 0.28"))

		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		/* EXPECTED:
		t.ExpectToast(Equals("Side panel width: 0.55"))
		ACTUAL: */
		t.ExpectToast(Equals("Side panel width: 0.45"))
	},
})
