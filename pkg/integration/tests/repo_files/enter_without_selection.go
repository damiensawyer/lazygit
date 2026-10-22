package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnterWithoutSelection = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browsing all files is disabled when the ref panel has nothing selected",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Tags().
			Focus().
			IsEmpty().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.ExpectToast(Equals("Disabled: No item selected"))

		t.Views().Tags().
			IsFocused()
	},
})
