package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var LargeNestedTree = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Navigate a tree with more entries than fit on screen in the repo-files panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.RunShellCommand(`for i in $(seq -w 1 40); do mkdir -p "dir$i/a/b"; echo "content $i" > "dir$i/a/b/file$i.txt"; done`)
		shell.GitAddAll()
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			LineCount(EqualsInt(41)).
			TopLines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir01/a/b"),
				Equals("  ▶ dir02/a/b"),
			).
			Press(keys.Files.ExpandAll).
			LineCount(EqualsInt(81)).
			// only the visible rows are rendered, so moving far down must
			// render the rows that come into view
			NavigateToLine(Contains("file40.txt")).
			SelectedLine(Equals("    file40.txt")).
			SelectedLineIsVisible()

		t.Views().Main().
			Title(Equals("dir40/a/b/file40.txt")).
			Content(Contains("content 40"))

		t.Views().RepoFiles().
			NavigateToLine(Contains("dir01/a/b")).
			SelectedLine(Equals("  ▼ dir01/a/b")).
			SelectedLineIsVisible().
			TopLines(
				Equals("▼ /"),
				Equals("  ▼ dir01/a/b").IsSelected(),
				Equals("    file01.txt"),
			)
	},
})
