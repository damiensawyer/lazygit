package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnterFromSubCommits = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse the files of a commit of another branch, from the sub-commits panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("base.txt", "base\n")
		shell.Commit("base")
		shell.NewBranch("feature")
		shell.CreateFileAndAdd("feature1.txt", "feature1\n")
		shell.Commit("feature one")
		shell.CreateFileAndAdd("feature2.txt", "feature2\n")
		shell.Commit("feature two")
		shell.Checkout("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("feature")).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("feature two").IsSelected(),
				Contains("feature one"),
				Contains("base"),
			).
			NavigateToLine(Contains("feature one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  base.txt"),
				Equals("  feature1.txt"),
			).
			PressEscape()

		// escape goes back to the sub-commits panel rather than all the way
		// back to the branches panel
		t.Views().SubCommits().
			IsFocused().
			SelectedLine(Contains("feature one")).
			PressEscape()

		t.Views().Branches().
			IsFocused()
	},
})
