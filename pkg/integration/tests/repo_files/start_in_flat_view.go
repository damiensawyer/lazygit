package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var StartInFlatView = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The repo-files panel starts in flat view when the file tree is turned off in the config",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Gui.ShowFileTree = false
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/sub/a.txt", "a\n")
		shell.CreateFileAndAdd("b.txt", "b\n")
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("b.txt").IsSelected(),
				Equals("dir/sub/a.txt"),
			).
			Press(keys.Files.ToggleTreeView).
			TopLines(
				Equals("▼ /"),
				Equals("  b.txt").IsSelected(),
			)
	},
})
