package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var IncreaseContextSizeInFullFileDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Increasing the context size while showing the full file leaves it showing the full file",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Git.DiffContextSize = 2
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", twentyLines())
		shell.Commit("initial")
		shell.UpdateFile("file.txt", twentyLines(10))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		t.Views().Files().Press(keys.Universal.IncreaseContextInDiffView)
		t.ExpectToast(
			/* EXPECTED:
			Equals("Already showing the full file diff"))
			ACTUAL: */
			Equals("Changed diff context size to 18446744073709551615"))

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 2"))

		t.Views().Main().
			Content(Contains("@@ -8,5 +8,5 @@"))
	},
})
