package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ClickArrowToCollapse = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Click the arrow on a directory in the repo-files panel to collapse/expand it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/file-one", "one\n")
		shell.CreateFileAndAdd("dir2/file-two", "two\n")
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir"),
				Equals("  ▶ dir2"),
			).
			// the arrow on "dir" is at row 1, column 2
			Click(2, 1).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir").IsSelected(),
				Equals("    file-one"),
				Equals("  ▶ dir2"),
			).
			// the space after the arrow counts as part of it
			Click(3, 3).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir"),
				Equals("    file-one"),
				Equals("  ▼ dir2").IsSelected(),
				Equals("    file-two"),
			).
			// clicking the name only selects the directory
			Click(5, 1).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir").IsSelected(),
				Equals("    file-one"),
				Equals("  ▼ dir2"),
				Equals("    file-two"),
			).
			// and so does clicking left of the arrow
			Click(1, 3).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir"),
				Equals("    file-one"),
				Equals("  ▼ dir2").IsSelected(),
				Equals("    file-two"),
			).
			// clicking a file does nothing beyond selecting it
			Click(4, 2).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir"),
				Equals("    file-one").IsSelected(),
				Equals("  ▼ dir2"),
				Equals("    file-two"),
			).
			Click(0, 0).
			Lines(
				Equals("▶ /").IsSelected(),
			)
	},
})
