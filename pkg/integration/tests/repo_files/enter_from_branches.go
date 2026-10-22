package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnterFromBranches = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse the files of local and remote branches, returning to the originating panel on escape",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("base.txt", "base\n")
		shell.Commit("base")
		shell.NewBranch("feature")
		shell.CreateFileAndAdd("pushed.txt", "pushed\n")
		shell.Commit("pushed")
		shell.CloneIntoRemote("origin")
		// the local branch is now ahead of its remote counterpart
		shell.CreateFileAndAdd("local-only.txt", "local only\n")
		shell.Commit("local only")
		shell.Checkout("master")
		shell.CreateFileAndAdd("master.txt", "master\n")
		shell.Commit("master")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus().
			Lines(
				Contains("master").IsSelected(),
				Contains("feature"),
			).
			NavigateToLine(Contains("feature")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Title(Equals("Files at feature")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  base.txt"),
				Equals("  local-only.txt"),
				Equals("  pushed.txt"),
			).
			PressEscape()

		t.Views().Branches().
			IsFocused().
			SelectedLine(Contains("feature"))

		t.Views().Remotes().
			Focus().
			Lines(
				Contains("origin").IsSelected(),
			).
			PressEnter()

		t.Views().RemoteBranches().
			IsFocused().
			NavigateToLine(Contains("feature")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Title(Equals("Files at origin/feature")).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  base.txt"),
				Equals("  pushed.txt"),
			).
			PressEscape()

		t.Views().RemoteBranches().
			IsFocused().
			SelectedLine(Contains("feature"))
	},
})
