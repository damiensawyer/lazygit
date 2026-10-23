package config

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SidePanelWidthRounding = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Repeatedly adjusting the side panel width saves it without floating point noise",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo:    func(shell *Shell) {},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.38"))
		t.GlobalPress(keys.Universal.IncreaseSidePanelWidth)
		t.ExpectToast(Equals("Side panel width: 0.43"))

		t.FileSystem().FileContent(".git/lazygit.yml",
			Equals("gui:\n  sidePanelWidth: 0.4333\n"))
	},
})
