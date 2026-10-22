package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RestoreWithModifiedFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Restoring from the repo-files panel is refused when it would overwrite local modifications",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/modified.txt", "at one\n")
		shell.CreateFileAndAdd("other.txt", "other at one\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("dir/modified.txt", "at two\n")
		shell.UpdateFileAndAdd("other.txt", "other at two\n")
		shell.Commit("two")
		shell.UpdateFile("dir/modified.txt", "local modification\n")
		shell.CreateFile("untracked.txt", "untracked\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		// the directory containing the modified file can't be restored
		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("dir")).
			Press(keys.RepoFiles.Restore)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("You have local modifications for the file(s) you are trying to check out. You need to stash or discard these first.")).
			Confirm()

		// and neither can the root, which contains everything
		t.Views().RepoFiles().
			NavigateToLine(Equals("▼ /")).
			Press(keys.RepoFiles.Restore)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Contains("You have local modifications")).
			Confirm()

		t.FileSystem().
			FileContent("dir/modified.txt", Equals("local modification\n"))

		// an unmodified file can still be restored, and untracked files don't
		// get in the way
		t.Views().RepoFiles().
			NavigateToLine(Contains("other.txt")).
			Press(keys.RepoFiles.Restore)

		t.ExpectPopup().Confirmation().
			Title(Equals("Restore to worktree")).
			Content(Contains("'other.txt'")).
			Confirm()

		t.FileSystem().
			FileContent("other.txt", Equals("other at one\n")).
			FileContent("dir/modified.txt", Equals("local modification\n")).
			FileContent("untracked.txt", Equals("untracked\n"))
	},
})
