package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleShowFinalVersion = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Toggle between showing diffs and showing the final version of each changed file",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "unchanged line\nold line\n")
		shell.CreateFileAndAdd("deleted.txt", "goodbye\n")
		shell.Commit("initial")

		shell.UpdateFile("file.txt", "unchanged line\nnew line\n")
		shell.CreateFile("added.txt", "brand new file\n")
		shell.DeleteFile("deleted.txt")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			NavigateToLine(Contains("file.txt"))

		// By default the main view shows a diff, so only the changed lines
		// appear, marked up as changes.
		t.Views().Main().
			Content(Contains("-old line")).
			Content(Contains("+new line"))

		t.Views().Files().Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		// Now the whole file as it stands on disk is shown, with no diff
		// markers on it.
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Contains("unchanged line")).
			Content(Contains("new line")).
			Content(DoesNotContainAnyOf("-old line", "+new line"))

		// The toggle sticks while moving through the file list, which is the
		// point of it: each file shows its full contents.
		t.Views().Files().NavigateToLine(Contains("added.txt"))
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Contains("brand new file")).
			Content(DoesNotContain("+brand new file"))

		// A file the change set deleted has no final version to show.
		t.Views().Files().NavigateToLine(Contains("deleted.txt"))
		t.Views().Main().
			Content(Contains("This file doesn't exist anymore, so it has no final version to show."))

		t.Views().Files().Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing diffs"))

		// Toggling back restores the diff for the selected file.
		t.Views().Main().Content(Contains("-goodbye"))
	},
})
