package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthClamping = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Adjusting the side panel width stops at the minimum and maximum, in both normal and half screen mode",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Normal mode: 0.3333 decreases in steps of 0.05 down to 0.1
		for _, expected := range []string{"0.28", "0.23", "0.18", "0.13", "0.10", "0.10"} {
			t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
			t.ExpectToast(Equals("Side panel width: " + expected))
		}
		t.FileSystem().FileContent(".git/lazygit.yml", Equals("gui:\n  sidePanelWidth: 0.1\n"))

		// Going back up from the minimum uses the same step size
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.15"))

		for _, expected := range []string{
			"0.20", "0.25", "0.30", "0.35", "0.40", "0.45", "0.50", "0.55",
			"0.60", "0.65", "0.70", "0.75", "0.80", "0.85", "0.90", "0.90",
		} {
			t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
			t.ExpectToast(Equals("Side panel width: " + expected))
		}
		t.FileSystem().FileContent(".git/lazygit.yml", Equals("gui:\n  sidePanelWidth: 0.9\n"))

		// Half screen mode: 0.5 increases up to 0.9
		t.GlobalPress(keys.Universal.NextScreenMode)
		for _, expected := range []string{"0.55", "0.60", "0.65", "0.70", "0.75", "0.80", "0.85", "0.90", "0.90"} {
			t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
			t.ExpectToast(Equals("Side panel width: " + expected))
		}
		t.FileSystem().FileContent(".git/lazygit.yml",
			Contains("halfScreenModeSidePanelWidth: 0.9\n"))

		// ...and decreases down to 0.1
		for range 16 {
			t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
			t.ExpectToast(Contains("Side panel width: "))
		}
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.10"))
		t.FileSystem().FileContent(".git/lazygit.yml",
			Equals("gui:\n  sidePanelWidth: 0.9\n  halfScreenModeSidePanelWidth: 0.1\n"))
	},
})
