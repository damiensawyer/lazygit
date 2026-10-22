package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnterFromStash = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse the files of a stash entry, which include the stashed changes",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "committed\n")
		shell.Commit("one")
		shell.UpdateFile("file.txt", "stashed\n")
		shell.CreateFileAndAdd("new.txt", "new\n")
		shell.Stash("my stash")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Stash().
			Focus().
			Lines(
				Contains("my stash").IsSelected(),
			).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  file.txt"),
				Equals("  new.txt"),
			).
			NavigateToLine(Contains("file.txt"))

		t.Views().Main().
			Content(Contains("stashed"))

		t.Views().RepoFiles().
			PressEscape()

		t.Views().Stash().
			IsFocused()
	},
})
