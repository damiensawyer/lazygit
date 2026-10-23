package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthPersistence = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Normal and half screen mode side panel widths are adjusted independently and each is saved to the per-repo config",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.FileSystem().PathNotPresent(".git/lazygit.yml")

		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.28"))
		t.FileSystem().FileContent(".git/lazygit.yml",
			Equals("gui:\n  sidePanelWidth: 0.2833\n"))

		// Half screen mode starts from its own default, not from the normal width
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.55"))
		t.FileSystem().FileContent(".git/lazygit.yml",
			Equals("gui:\n  sidePanelWidth: 0.2833\n  halfScreenModeSidePanelWidth: 0.55\n"))

		// Back in normal mode, the normal width is unaffected by the half screen
		// mode adjustment
		t.GlobalPress(keys.Universal.PrevScreenMode)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.33"))
		t.FileSystem().FileContent(".git/lazygit.yml",
			Equals("gui:\n  sidePanelWidth: 0.3333\n  halfScreenModeSidePanelWidth: 0.55\n"))

		// And the half screen mode width was remembered too
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.50"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.45"))

		// Resetting in half screen mode removes the saved half screen mode width
		// and leaves the normal width alone
		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.ExpectToast(Equals("Reset side panel width"))
		t.FileSystem().FileContent(".git/lazygit.yml",
			/* EXPECTED:
			Equals("gui:\n  sidePanelWidth: 0.3333\n"))
			ACTUAL: */
			Equals("gui:\n  sidePanelWidth: 0.3333\n  halfScreenModeSidePanelWidth: 0.5\n"))

		// Resetting in normal mode likewise leaves the half screen mode width alone
		t.GlobalPress(keys.Universal.PrevScreenMode)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.38"))
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.55"))
		t.GlobalPress(keys.Universal.PrevScreenMode)
		t.GlobalPress(keys.Universal.ResetSidePanelWidth)
		t.ExpectToast(Equals("Reset side panel width"))
		t.FileSystem().FileContent(".git/lazygit.yml",
			/* EXPECTED:
			Equals("gui:\n  halfScreenModeSidePanelWidth: 0.55\n"))
			ACTUAL: */
			Equals("gui:\n  sidePanelWidth: 0.3333\n  halfScreenModeSidePanelWidth: 0.55\n"))
	},
})
