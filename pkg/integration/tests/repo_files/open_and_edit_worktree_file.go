package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OpenAndEditWorktreeFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Open/edit the worktree version of files selected in the repo-files panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().OS.Edit = "echo {{filename}} > edit-command"
		config.GetUserConfig().OS.Open = "echo {{filename}} > open-command"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/kept.txt", "kept\n")
		shell.CreateFileAndAdd("dir/gone.txt", "gone\n")
		shell.CreateFileAndAdd("other.txt", "other\n")
		shell.Commit("one")
		shell.DeleteFileAndAdd("dir/gone.txt")
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
			PressEnter().
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir").IsSelected(),
				Equals("    gone.txt"),
				Equals("    kept.txt"),
				Equals("  other.txt"),
			).
			// a directory on its own can't be edited
			Press(keys.Universal.Edit).
			Tap(func() {
				t.ExpectPopup().Alert().
					Title(Equals("Error")).
					Content(Equals("Cannot edit directories: you can only edit individual files")).
					Confirm()
			}).
			NavigateToLine(Contains("kept.txt")).
			Press(keys.Universal.OpenFile)

		t.FileSystem().FileContent("open-command", Contains("dir/kept.txt"))

		// editing a range only edits the files that still exist in the
		// worktree, skipping the directory and the deleted file
		t.Views().RepoFiles().
			NavigateToLine(Contains("dir")).
			Press(keys.Universal.ToggleRangeSelect).
			NavigateToLine(Contains("other.txt")).
			Press(keys.Universal.Edit)

		t.FileSystem().FileContent("edit-command",
			Contains("dir/kept.txt").Contains("other.txt").DoesNotContain("gone.txt"))
	},
})
