package config

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthSwitchingRepos = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Each repo saves and restores its own side panel widths when switching between repos",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		otherRepo, _ := filepath.Abs("../other")
		cfg.GetAppState().RecentRepos = []string{otherRepo}
	},
	SetupRepo: func(shell *Shell) {
		shell.CloneNonBare("other")
		shell.CreateFile("../other/.git/lazygit.yml", `
gui:
  sidePanelWidth: 0.6
  halfScreenModeSidePanelWidth: 0.4
`)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// Switching repos changes the working directory, so resolve the paths
		// up front
		repoConfigPath, _ := filepath.Abs(".git/lazygit.yml")
		otherConfigPath, _ := filepath.Abs("../other/.git/lazygit.yml")

		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.28"))
		t.FileSystem().FileContent(repoConfigPath,
			Equals("gui:\n  sidePanelWidth: 0.2833\n"))

		switchToRecentRepo(t, keys)
		t.Views().Status().Content(Contains("other → master"))

		// The other repo uses its own saved widths
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.65"))
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.45"))
		t.GlobalPress(keys.Universal.PrevScreenMode)
		t.FileSystem().FileContent(otherConfigPath,
			Equals("gui:\n  sidePanelWidth: 0.65\n  halfScreenModeSidePanelWidth: 0.45\n"))

		// The first repo's saved width is untouched
		t.FileSystem().FileContent(repoConfigPath,
			Equals("gui:\n  sidePanelWidth: 0.2833\n"))

		// Switching back restores the first repo's widths
		switchToRecentRepo(t, keys)
		t.Views().Status().Content(Contains("repo → master"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.23"))
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.45"))
		t.GlobalPress(keys.Universal.PrevScreenMode)

		// And switching to the other repo again restores its widths
		switchToRecentRepo(t, keys)
		t.Views().Status().Content(Contains("other → master"))
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.60"))
		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.40"))
	},
})

func switchToRecentRepo(t *TestDriver, keys config.KeybindingConfig) {
	t.GlobalPress(keys.Universal.OpenRecentRepos)
	t.ExpectPopup().Menu().Title(Equals("Recent repositories")).Confirm()
}
