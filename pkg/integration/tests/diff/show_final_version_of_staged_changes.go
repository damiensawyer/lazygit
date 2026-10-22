package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionOfStagedChanges = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The final version of a file with staged changes is the one on disk, whatever is in the index",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.ShowFileTree = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("partly-staged.txt", "committed\n")
		shell.CreateFileAndAdd("staged.txt", "committed\n")
		shell.CreateFileAndAdd("old-name.txt", "moved content\n")
		shell.CreateFileAndAdd("staged-deletion.txt", "doomed\n")
		shell.Commit("initial")

		shell.UpdateFileAndAdd("partly-staged.txt", "in the index\n")
		shell.UpdateFile("partly-staged.txt", "on disk\n")
		shell.UpdateFileAndAdd("staged.txt", "staged\n")
		shell.RenameFileInGit("old-name.txt", "new-name.txt")
		shell.DeleteFileAndAdd("staged-deletion.txt")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Lines(
				Contains("R  old-name.txt → new-name.txt").IsSelected(),
				Contains("MM partly-staged.txt"),
				Contains("D  staged-deletion.txt"),
				Contains("M  staged.txt"),
			).
			Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		// A rename shows the file under its new name.
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("moved content"))

		t.Views().Files().NavigateToLine(Contains("partly-staged.txt"))
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("on disk"))

		t.Views().Files().NavigateToLine(Contains("staged-deletion.txt"))
		t.Views().Main().
			Content(Equals("This file doesn't exist anymore, so it has no final version to show."))

		t.Views().Files().NavigateToLine(Contains("M  staged.txt"))
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("staged"))
	},
})
