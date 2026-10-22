package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CopyAbsolutePathAndBlobHash = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Copy the absolute path and blob hash of a file, and the name/path of a directory, from the repo-files panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		// note: this is required to simulate the clipboard during CI
		config.GetUserConfig().OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/sub/file.txt", "old content\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("dir/sub/file.txt", "new content\n")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▶ dir/sub").IsSelected(),
			).
			PressEnter().
			Lines(
				Equals("▼ dir/sub").IsSelected(),
				Equals("  file.txt"),
			).
			NavigateToLine(Contains("file.txt")).
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Absolute path")).
			Confirm()

		t.ExpectToast(Equals("File path copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Contains("/repo/dir/sub/file.txt"))

		// the blob hash is that of the content at the viewed commit
		t.Views().RepoFiles().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Blob hash")).
			Confirm()

		t.ExpectToast(Equals("Blob hash copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("33194a0a6f3f99e366d606c24d9b1ab0e0086e69"))

		// for a (compressed) directory, the name and path are those of its
		// deepest part, but there's no blob hash
		t.Views().RepoFiles().
			NavigateToLine(Contains("dir/sub")).
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("File name")).
			Confirm()

		t.ExpectToast(Equals("File name copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("sub"))

		t.Views().RepoFiles().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Relative path")).
			Confirm()

		t.ExpectToast(Equals("File path copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("dir/sub"))

		t.Views().RepoFiles().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Blob hash")).
			Tooltip(Contains("Disabled: Cannot copy content of directories")).
			Cancel()

		t.Shell().DeleteFile("clipboard")
	},
})
