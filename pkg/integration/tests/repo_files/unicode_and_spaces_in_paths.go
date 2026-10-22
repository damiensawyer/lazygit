package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var UnicodeAndSpacesInPaths = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse, view, copy, save and restore files whose paths contain spaces and non-ASCII characters",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		// note: this is required to simulate the clipboard during CI
		config.GetUserConfig().OS.CopyToClipboardCmd = "printf '%s' {{text}} > clipboard"
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir with spaces/file name.txt", "spaces at one\n")
		shell.CreateFileAndAdd("日本語/ファイル.txt", "unicode at one\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("dir with spaces/file name.txt", "spaces at two\n")
		shell.UpdateFileAndAdd("日本語/ファイル.txt", "unicode at two\n")
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
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir with spaces"),
				Equals("  ▶ 日本語"),
			).
			Press(keys.Files.ExpandAll).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▼ dir with spaces"),
				Equals("    file name.txt"),
				Equals("  ▼ 日本語"),
				Equals("    ファイル.txt"),
			).
			NavigateToLine(Contains("日本語"))

		t.Views().Main().
			Title(Equals("日本語")).
			Content(Equals("ファイル.txt"))

		t.Views().RepoFiles().
			NavigateToLine(Contains("ファイル.txt"))

		t.Views().Main().
			Title(Equals("日本語/ファイル.txt")).
			Content(Contains("unicode at one"))

		t.Views().RepoFiles().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("Relative path")).
			Confirm()

		t.ExpectToast(Equals("File path copied to clipboard"))
		t.FileSystem().FileContent("clipboard", Equals("日本語/ファイル.txt"))
		t.Shell().DeleteFile("clipboard")

		t.Views().RepoFiles().
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save as...")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Save to path")).
			InitialText(Equals("日本語/ファイル.txt")).
			Clear().
			Type("saved copy.txt").
			Confirm()

		t.ExpectToast(Equals("Saved to 'saved copy.txt'"))
		t.FileSystem().FileContent("saved copy.txt", Equals("unicode at one\n"))

		t.Views().RepoFiles().
			NavigateToLine(Contains("file name.txt"))

		t.Views().Main().
			Title(Equals("dir with spaces/file name.txt")).
			Content(Contains("spaces at one"))

		t.Views().RepoFiles().
			Press(keys.RepoFiles.Restore)

		t.ExpectPopup().Confirmation().
			Title(Equals("Restore to worktree")).
			Content(Contains("'dir with spaces/file name.txt'")).
			Confirm()

		t.FileSystem().
			FileContent("dir with spaces/file name.txt", Equals("spaces at one\n")).
			FileContent("日本語/ファイル.txt", Equals("unicode at two\n"))
	},
})
