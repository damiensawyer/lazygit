package config

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var HalfScreenModeSidePanelWidth = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Adjusting half screen mode side panel width in per-repo config persists the setting",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		otherRepo, _ := filepath.Abs("../other")
		cfg.GetAppState().RecentRepos = []string{otherRepo}
	},
	SetupRepo: func(shell *Shell) {
		shell.CloneNonBare("other")
		shell.CreateFile("../other/.git/lazygit.yml", `
halfScreenModeSidePanelWidth: 0.6`)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.GlobalPress(keys.Universal.OpenRecentRepos)
		t.ExpectPopup().Menu().Title(Equals("Recent repositories")).
			Lines(
				Contains("other").IsSelected(),
				Contains("Cancel"),
			).Confirm()
		t.Views().Status().Content(Contains("other → master"))

		// Enter half screen mode
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.Views().Status().Content(Contains("other → master"))

		// Adjust the side panel width
		t.GlobalPress(keys.Universal.DecreaseHalfScreenModeSidePanelWidth)
		t.ExpectToast(Contains("Half screen mode side panel width: 0.40"))

		t.GlobalPress(keys.Universal.IncreaseHalfScreenModeSidePanelWidth)
		t.ExpectToast(Contains("Half screen mode side panel width: 0.50"))

		// Reset to default
		t.GlobalPress(keys.Universal.ResetHalfScreenModeSidePanelWidth)
		t.ExpectToast(Contains("Reset half screen mode side panel width"))

		// Verify the per-repo config is read
		t.ExpectPopup().Menu().Title(Equals("Menu")).
			Confirm()
	},
})
