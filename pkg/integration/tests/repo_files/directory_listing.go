package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DirectoryListing = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Selecting a directory in the repo-files panel lists its immediate entries in the main view",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/a.txt", "a\n")
		shell.CreateFileAndAdd("dir/nested/deep/b.txt", "b\n")
		shell.CreateFileAndAdd("dir/z.txt", "z\n")
		shell.CreateFileAndAdd("top.txt", "top\n")
		shell.Commit("one")
		// later changes must not leak into the listing of the older commit
		shell.CreateFileAndAdd("dir/later.txt", "later\n")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		// the root item lists the top level of the repo
		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir"),
				Equals("  top.txt"),
			)

		t.Views().Main().
			Content(Equals("dir/\ntop.txt"))

		// subdirectories get a trailing slash, and their contents aren't
		// listed
		t.Views().RepoFiles().
			NavigateToLine(Contains("dir"))

		t.Views().Main().
			Title(Equals("dir")).
			Content(Equals("a.txt\nnested/\nz.txt"))

		// a compressed directory lists the contents of its deepest part
		t.Views().RepoFiles().
			PressEnter().
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir").IsSelected(),
				Equals("    a.txt"),
				Equals("    ▶ nested/deep"),
				Equals("    z.txt"),
				Equals("  top.txt"),
			).
			NavigateToLine(Contains("nested/deep"))

		t.Views().Main().
			Title(Equals("dir/nested/deep")).
			Content(Equals("b.txt"))
	},
})
