package main_view

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var StageDiffLinesInAFullFileDiff = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Switch the focused main view to a full-file diff and stage lines from it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.UseHunkModeInDiffView = false
	},
	SetupRepo: func(shell *Shell) {
		lines := make([]string, 20)
		for i := range lines {
			lines[i] = fmt.Sprintf("line%02d", i+1)
		}
		shell.CreateFileAndAdd("file.txt", strings.Join(lines, "\n")+"\n")
		shell.Commit("initial")

		// Two changes, far enough apart to be separate hunks at the default
		// context size.
		lines[2] = "LINE03"
		lines[16] = "LINE17"
		shell.UpdateFile("file.txt", strings.Join(lines, "\n")+"\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			Content(Contains("@@ -1,6 +1,6 @@")).
			Content(Contains("@@ -14,7 +14,7 @@")).
			SelectedLines(
				Contains("-line03"),
			).
			Press(keys.Universal.ToggleContextInDiffView).
			Tap(func() {
				t.ExpectToast(Equals("Showing full file diff"))
			}).
			// One hunk covering the whole file now, and the selection stays on
			// the line it was on.
			Content(Contains("@@ -1,20 +1,20 @@")).
			Content(DoesNotContain("@@ -14,")).
			SelectedLines(
				Contains("-line03"),
			).
			Press(keys.Universal.RangeSelectDown).
			SelectedLines(
				Contains("-line03"),
				Contains("+LINE03"),
			).
			PressPrimaryAction()

		t.Views().Files().Lines(
			Contains("MM file.txt"),
		)

		// Only the selected change went into the index, and both sides keep
		// showing the whole file.
		t.Views().Secondary().
			Content(Contains("@@ -1,20 +1,20 @@")).
			ContainsLines(
				Contains("-line03"),
				Contains("+LINE03"),
			).
			Content(DoesNotContain("LINE17"))

		t.Views().Main().
			IsFocused().
			Content(Contains("@@ -1,20 +1,20 @@")).
			Content(DoesNotContain("-line03")).
			Content(Contains(" LINE03")).
			ContainsLines(
				Contains("-line17"),
				Contains("+LINE17"),
			).
			Press(keys.Universal.ToggleContextInDiffView).
			Tap(func() {
				t.ExpectToast(Equals("Restored context size to 3"))
			}).
			Content(Contains("@@ -14,7 +14,7 @@")).
			Content(DoesNotContain("LINE03"))

		t.Views().Secondary().
			Content(Contains("@@ -1,6 +1,6 @@"))
	},
})
