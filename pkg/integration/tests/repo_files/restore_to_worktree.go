package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RestoreToWorktree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Restore a directory to its state at an old commit, as unstaged worktree changes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/file1.txt", "file1 at one\n")
		shell.CreateFileAndAdd("dir/file2.txt", "file2 at one\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("dir/file1.txt", "file1 at two\n")
		shell.DeleteFileAndAdd("dir/file2.txt")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("dir")).
			Press(keys.RepoFiles.Restore)

		t.ExpectPopup().Confirmation().
			Title(Equals("Restore to worktree")).
			Content(Contains("Are you sure you want to replace 'dir'")).
			Confirm()

		// the worktree now has the old state of the directory, as unstaged
		// changes
		t.FileSystem().
			FileContent("dir/file1.txt", Equals("file1 at one\n")).
			FileContent("dir/file2.txt", Equals("file2 at one\n"))

		t.Views().RepoFiles().
			PressEscape()

		t.Views().Files().
			Focus().
			Lines(
				Equals("▼ dir").IsSelected(),
				Equals("   M file1.txt"),
				Equals("  ?? file2.txt"),
			)
	},
})
