package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionOfRenamedAndDeletedCommitFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the final version of files that a commit renamed or deleted, then switch back to their diffs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.ShowFileTree = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("old-name.txt", "moved content\n")
		shell.CreateFileAndAdd("doomed.txt", "goodbye\n")
		shell.Commit("initial")

		shell.RenameFileInGit("old-name.txt", "new-name.txt")
		shell.DeleteFileAndAdd("doomed.txt")
		shell.Commit("rename and delete")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			SelectedLine(Contains("rename and delete")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Contains("D doomed.txt").IsSelected(),
				Contains("R old-name.txt → new-name.txt"),
			).
			Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		t.Views().Main().
			Content(Equals("This file doesn't exist anymore, so it has no final version to show."))

		// A rename shows the file under the name the commit gave it.
		t.Views().CommitFiles().NavigateToLine(Contains("new-name.txt"))
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("moved content"))

		t.Views().CommitFiles().Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing diffs"))

		t.Views().Main().
			Title(Equals("Patch")).
			Content(Contains("rename from old-name.txt")).
			Content(Contains("rename to new-name.txt"))

		t.Views().CommitFiles().NavigateToLine(Contains("doomed.txt"))
		t.Views().Main().
			Title(Equals("Patch")).
			Content(Contains("-goodbye"))
	},
})
