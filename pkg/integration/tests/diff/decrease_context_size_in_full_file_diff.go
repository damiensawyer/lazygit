package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DecreaseContextSizeInFullFileDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Decreasing the context size while showing the full file goes back to the context size in use before it",
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

		t.Views().Files().Press(keys.Universal.DecreaseContextInDiffView)
		t.ExpectToast(
			Equals("Restored context size to 2"))

		t.Views().Main().
			Content(Contains("@@ -8,5 +8,5 @@"))

		// Showing the full file again comes back to the same size
		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))
		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(
			Equals("Restored context size to 2"))
	},
})
