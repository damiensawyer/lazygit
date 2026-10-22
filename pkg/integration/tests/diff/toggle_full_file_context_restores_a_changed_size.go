package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleFullFileContextRestoresAChangedSize = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Leaving the full-file diff goes back to the context size in use before it, even one changed since startup",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Git.DiffContextSize = 1
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", twentyLines())
		shell.Commit("initial")
		shell.UpdateFile("file.txt", twentyLines(10))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Press(keys.Universal.IncreaseContextInDiffView)
		t.ExpectToast(Equals("Changed diff context size to 2"))

		t.Views().Main().
			Content(Contains("@@ -8,5 +8,5 @@"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 2"))

		t.Views().Main().
			Content(Contains("@@ -8,5 +8,5 @@"))

		// And the same again from the restored size, which is now the one to
		// come back to.
		t.Views().Files().Press(keys.Universal.IncreaseContextInDiffView)
		t.ExpectToast(Equals("Changed diff context size to 3"))
		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))
		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 3"))
	},
})
