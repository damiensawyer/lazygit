package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleFullFileContextOfACommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Toggle the full-file diff from the commits, commit files and branches panels",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", twentyLines())
		shell.Commit("initial")
		shell.UpdateFileAndAdd("file.txt", twentyLines(10))
		shell.Commit("change line ten")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			SelectedLine(Contains("change line ten"))

		t.Views().Main().
			Content(Contains("@@ -7,7 +7,7 @@"))

		t.Views().Commits().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@")).
			Content(Contains(" line01")).
			Content(Contains(" line20"))

		// The setting is global, so the commit's files show the whole file
		// too, and can switch it back.
		t.Views().Commits().PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			SelectedLine(Contains("file.txt"))

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))

		t.Views().CommitFiles().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 3"))

		t.Views().Main().
			Content(Contains("@@ -7,7 +7,7 @@"))

		t.Views().CommitFiles().PressEscape()

		// The branches panel shows a log rather than a diff, but the toggle
		// still applies from there to every diff shown afterwards.
		t.Views().Branches().
			Focus().
			Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		t.Views().Commits().Focus()

		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@"))
	},
})
