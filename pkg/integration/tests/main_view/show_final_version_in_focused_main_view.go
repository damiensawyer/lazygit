package main_view

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionInFocusedMainView = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Showing a file's final version in the focused main view leaves nothing to stage, until the diff comes back",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.UseHunkModeInDiffView = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "unchanged\nold\n")
		shell.Commit("initial")
		shell.UpdateFile("file.txt", "unchanged\nnew\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			SelectedLines(
				Contains("-old"),
			).
			Press(keys.Universal.ToggleShowFinalVersion).
			Tap(func() {
				t.ExpectToast(Equals("Showing final version of files"))
			}).
			// The file's contents aren't diff lines, so there is nothing to
			// select, and so nothing to stage.
			Title(Equals("Final version")).
			Content(Equals("unchanged\nnew")).
			IsFocused().
			SelectionIsHidden().
			PressPrimaryAction().
			Tap(func() {
				t.ExpectToast(Equals("Disabled: There is nothing to select here"))
			})

		t.Views().Files().Lines(
			Contains(" M file.txt"),
		)

		t.Views().Main().
			Press(keys.Universal.ToggleShowFinalVersion).
			Tap(func() {
				t.ExpectToast(Equals("Showing diffs"))
			}).
			IsFocused().
			Title(Equals("Unstaged changes")).
			Content(Contains("-old\n+new"))

		// With the diff back, staging from it works as before.
		t.Views().Files().
			Focus().
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			SelectedLines(
				Contains("-old"),
			).
			Press(keys.Universal.RangeSelectDown).
			PressPrimaryAction()

		t.Views().Files().Lines(
			Contains("M  file.txt"),
		)
	},
})
