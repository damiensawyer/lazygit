package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseAllFilesAtCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse all files of the repo as they existed at a commit, and navigate the tree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir1/file1.txt", "file1 content\n")
		shell.CreateFileAndAdd("dir2/file2.txt", "file2 content\n")
		shell.CreateFileAndAdd("root.txt", "root content\n")
		shell.Commit("one")
		shell.DeleteFileAndAdd("dir2/file2.txt")
		shell.CreateFileAndAdd("dir3/file3.txt", "file3 content\n")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			).
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		// the panel shows the full state of the repo at the selected commit,
		// with only the top level expanded
		t.Views().RepoFiles().
			IsFocused().
			Title(Contains("Files at")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir1"),
				Equals("  ▶ dir2"),
				Equals("  root.txt"),
			).
			NavigateToLine(Contains("dir2")).
			PressEnter().
			Lines(
				Equals("▼ /"),
				Equals("  ▶ dir1"),
				Equals("  ▼ dir2").IsSelected(),
				Equals("    file2.txt"),
				Equals("  root.txt"),
			).
			PressEscape()

		t.Views().Commits().
			IsFocused()

		// re-entering the same commit keeps the tree state (dir2 expanded)
		t.Views().Commits().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  ▶ dir1"),
				Equals("  ▼ dir2").IsSelected(),
				Equals("    file2.txt"),
				Equals("  root.txt"),
			).
			PressEscape()

		// the most recent commit shows dir3 instead of dir2
		t.Views().Commits().
			NavigateToLine(Contains("two")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir1"),
				Equals("  ▶ dir3"),
				Equals("  root.txt"),
			)
	},
})
