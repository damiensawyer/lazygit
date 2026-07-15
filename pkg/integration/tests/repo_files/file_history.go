package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var FileHistory = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "View the history of a file selected in the repo-files panel, via the filtering menu",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file1.txt", "one\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("file1.txt", "two\n")
		shell.Commit("two")
		shell.CreateFileAndAdd("file2.txt", "three\n")
		shell.Commit("three")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("file1.txt")).
			Press(keys.Universal.FilteringMenu)

		t.ExpectPopup().Menu().Title(Equals("Filtering")).
			Select(Contains("Filter by 'file1.txt'")).Confirm()

		// we land in the commits panel showing only the commits that touched
		// the file
		t.Views().Commits().
			IsFocused().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			)
	},
})
