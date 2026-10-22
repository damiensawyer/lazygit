package diff

import (
	"math"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleFullFileContextFromAConfiguredFullFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Leaving a full-file diff that the config started in, with no earlier size to go back to, falls back to three lines of context",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Git.DiffContextSize = math.MaxUint64
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", twentyLines())
		shell.Commit("initial")
		shell.UpdateFile("file.txt", twentyLines(10))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))

		t.Views().Files().
			IsFocused().
			Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 3"))

		t.Views().Main().
			Content(Contains("@@ -7,7 +7,7 @@"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))
	},
})
