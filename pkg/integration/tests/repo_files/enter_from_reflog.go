package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnterFromReflog = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse the files of a reflog entry for a commit that is no longer on any branch",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n")
		shell.Commit("one")
		shell.CreateFileAndAdd("lost.txt", "lost\n")
		shell.Commit("lost")
		shell.HardReset("HEAD^")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().ReflogCommits().
			Focus().
			Lines(
				Contains("reset: moving to HEAD^").IsSelected(),
				Contains("commit: lost"),
				Contains("commit (initial): one"),
			).
			NavigateToLine(Contains("commit: lost")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  lost.txt"),
				Equals("  one.txt"),
			).
			NavigateToLine(Contains("lost.txt"))

		t.Views().Main().
			Content(Contains("lost"))

		t.Views().RepoFiles().
			PressEscape()

		t.Views().ReflogCommits().
			IsFocused().
			SelectedLine(Contains("commit: lost"))
	},
})
