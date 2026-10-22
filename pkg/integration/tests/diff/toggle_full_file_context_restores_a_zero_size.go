package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleFullFileContextRestoresAZeroSize = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Leaving the full-file diff goes back to a context size of zero when that was the size in use before it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Git.DiffContextSize = 0
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", twentyLines())
		shell.Commit("initial")
		shell.UpdateFile("file.txt", twentyLines(10))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Main().
			Content(Contains("@@ -10 +10 @@"))

		t.Views().Files().
			IsFocused().
			Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 0"))

		t.Views().Main().
			Content(Contains("@@ -10 +10 @@"))
	},
})
