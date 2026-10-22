package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FilterFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Filter the files of the repo-files panel by path in flat view, and have re-entering clear the filter",
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
			Press(keys.Files.ToggleTreeView).
			Lines(
				Equals("dir1/apple.txt").IsSelected(),
				Equals("dir1/banana.txt"),
				Equals("dir2/apple-pie.txt"),
				Equals("root.txt"),
			).
			FilterOrSearch("apple").
			Lines(
				Equals("dir1/apple.txt").IsSelected(),
				Equals("dir2/apple-pie.txt"),
			).
			NavigateToLine(Contains("apple-pie"))

		t.Views().Main().
			Content(Contains("pie"))

		t.Views().RepoFiles().
			Press(keys.Universal.Return).
			// the selection is kept when the filter is cancelled
			Lines(
				Equals("dir1/apple.txt"),
				Equals("dir1/banana.txt"),
				Equals("dir2/apple-pie.txt").IsSelected(),
				Equals("root.txt"),
			).
			Tap(func() {
				t.Views().Search().IsInvisible()
			}).
			FilterOrSearch("banana").
			Lines(
				Equals("dir1/banana.txt").IsSelected(),
			).
			PressEscape()

		t.Views().RepoFiles().
			IsFocused().
			PressEscape()

		t.Views().Commits().
			IsFocused().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		// re-entering the panel doesn't keep the filter
		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("dir1/apple.txt"),
				Equals("dir1/banana.txt").IsSelected(),
				Equals("dir2/apple-pie.txt"),
				Equals("root.txt"),
			).
			Tap(func() {
				t.Views().Search().IsInvisible()
			})
	},
})
