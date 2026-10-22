package main_view

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleFullFileContextWhileBuildingPatch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Switch a commit's diff to the full file and back while building a custom patch from it",
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

		lines[2] = "LINE03"
		lines[16] = "LINE17"
		shell.UpdateFileAndAdd("file.txt", strings.Join(lines, "\n")+"\n")
		shell.Commit("change two lines")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			SelectedLine(Contains("change two lines")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			SelectedLine(Contains("file.txt")).
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			SelectedLines(Contains("-line03")).
			Press(keys.Universal.RangeSelectDown).
			PressPrimaryAction()

		t.Views().Information().Content(Contains("Building patch"))

		t.Views().Main().
			Press(keys.Universal.ToggleContextInDiffView).
			Tap(func() {
				t.ExpectToast(Equals("Showing full file diff"))
			}).
			Content(Contains("@@ -1,20 +1,20 @@")).
			// The lines already in the patch are still marked as such.
			MarkedLines(
				Contains("-line03"),
				Contains("+LINE03"),
			).
			NavigateToLine(Contains("-line17")).
			Press(keys.Universal.RangeSelectDown).
			PressPrimaryAction().
			MarkedLines(
				Contains("-line03"),
				Contains("+LINE03"),
				Contains("-line17"),
				Contains("+LINE17"),
			).
			Press(keys.Universal.ToggleContextInDiffView).
			Tap(func() {
				t.ExpectToast(Equals("Restored context size to 3"))
			}).
			Content(Contains("@@ -14,7 +14,7 @@")).
			MarkedLines(
				Contains("-line03"),
				Contains("+LINE03"),
				Contains("-line17"),
				Contains("+LINE17"),
			)

		t.Views().Secondary().ContainsLines(
			Contains("-line03"),
			Contains("+LINE03"),
		)
		t.Views().Secondary().ContainsLines(
			Contains("-line17"),
			Contains("+LINE17"),
		)

		// A patch put together from diffs of different context sizes still
		// applies.
		t.Common().SelectPatchOption(Contains("Move patch out into index"))

		t.Views().Files().
			Focus().
			Lines(
				Contains("M  file.txt"),
			)
		t.Views().Secondary().
			Content(Contains("-line03\n+LINE03")).
			Content(Contains("-line17\n+LINE17"))
	},
})
