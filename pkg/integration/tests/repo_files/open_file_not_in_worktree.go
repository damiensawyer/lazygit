package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OpenFileNotInWorktree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Open/edit of a file at a commit is disabled when the file no longer exists in the worktree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("gone.txt", "gone\n")
		shell.Commit("one")
		shell.DeleteFileAndAdd("gone.txt")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("gone.txt")).
			Press(keys.Universal.OpenFile)

		t.ExpectToast(Equals("Disabled: Does not exist in the working tree"))

		t.Views().RepoFiles().
			Press(keys.Universal.Edit)

		t.ExpectToast(Equals("Disabled: Does not exist in the working tree"))
	},
})
