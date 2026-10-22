package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionOfStashAndBranchFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the final version of the files of a stash entry and of a commit on another branch, as they left them",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "committed\n")
		shell.Commit("initial")

		shell.NewBranch("feature")
		shell.UpdateFileAndAdd("file.txt", "on feature\n")
		shell.Commit("feature commit")
		shell.Checkout("master")

		shell.UpdateFile("file.txt", "stashed\n")
		shell.Stash("my stash")

		// What's on disk now differs from both, so we can tell which version
		// is shown.
		shell.UpdateFile("file.txt", "on disk\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Stash().
			Focus().
			SelectedLine(Contains("my stash")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			SelectedLine(Contains("file.txt")).
			Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("stashed"))

		t.Views().CommitFiles().PressEscape()

		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("feature")).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			SelectedLine(Contains("feature commit")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			SelectedLine(Contains("file.txt"))

		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("on feature"))
	},
})
