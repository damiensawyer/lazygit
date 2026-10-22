package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthFromPerRepoConfig = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Side panel widths saved in the per-repo config are picked up on startup and take precedence over the global config",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.SidePanelWidth = 0.2
		cfg.GetUserConfig().Gui.HalfScreenModeSidePanelWidth = 0.3
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFile(".git/lazygit.yml", `# my settings
gui:
  sidePanelWidth: 0.5
  showIcons: false
  halfScreenModeSidePanelWidth: 0.8 # wide
`)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.45"))

		t.GlobalPress(keys.Universal.NextScreenMode)
		t.GlobalPress(keys.Universal.DecreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.75"))

		// The rest of the file is left as it was
		t.FileSystem().FileContent(".git/lazygit.yml", Equals(`# my settings
gui:
  sidePanelWidth: 0.45
  showIcons: false
  halfScreenModeSidePanelWidth: 0.75 # wide
`))
	},
})
