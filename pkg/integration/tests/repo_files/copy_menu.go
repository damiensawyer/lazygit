package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CopyMenu = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Copy file info from the repo-files panel: name, path, and content at the commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		// note: this is required to simulate the clipboard during CI
		config.GetUserConfig().OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/file.txt", "old content\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("dir/file.txt", "new content\n")
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
			NavigateToLine(Contains("file.txt")).
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("File name")).
			Confirm()

		t.ExpectToast(Equals("File name copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("file.txt"))

		t.Views().RepoFiles().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Relative path")).
			Confirm()

		t.ExpectToast(Equals("File path copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("dir/file.txt"))

		// the copied content is the content at the viewed commit, not the
		// current content
		t.Views().RepoFiles().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Content of selected file")).
			Confirm()

		t.ExpectToast(Equals("File content copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("old content\n"))

		// directories can't have their content copied
		t.Views().RepoFiles().
			NavigateToLine(Contains("dir")).
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Content of selected file")).
			Tooltip(Contains("Disabled: Cannot copy content of directories")).
			Cancel()

		t.Shell().DeleteFile("clipboard")
	},
})
