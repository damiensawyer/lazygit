package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterInTreeView = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Filter the files of the repo-files panel in tree view",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir1/apple.txt", "apple\n")
		shell.CreateFileAndAdd("dir1/banana.txt", "banana\n")
		shell.CreateFileAndAdd("dir2/apple-pie.txt", "pie\n")
		shell.CreateFileAndAdd("root.txt", "root\n")
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
				Equals("  ▶ dir1"),
				Equals("  ▶ dir2"),
				Equals("  root.txt"),
			).
			FilterOrSearch("apple").
			Lines(
				Equals("▼ /").IsSelected(),
				/* EXPECTED:
				Equals("  ▼ dir1"),
				Equals("    apple.txt"),
				Equals("  ▼ dir2"),
				Equals("    apple-pie.txt"),
				ACTUAL: */
				Equals("  ▶ dir1"),
				Equals("  ▶ dir2"),
			)
	},
})
