package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionOfCommitFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Show the final version of a file in a commit, as the commit left it rather than as it is now",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "first\n")
		shell.Commit("commit one")

		shell.UpdateFileAndAdd("file.txt", "first\nsecond\n")
		shell.Commit("commit two")

		// A later state that the commit under inspection knows nothing about,
		// so we can tell the commit's version apart from the current one.
		shell.UpdateFileAndAdd("file.txt", "first\nsecond\nthird\n")
		shell.Commit("commit three")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("commit two")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			SelectedLine(Contains("file.txt"))

		t.Views().Main().
			Content(Contains("+second")).
			Content(DoesNotContain("third"))

		t.Views().CommitFiles().Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		// The file as commit two left it: the whole file, with no diff markers,
		// and without the line commit three went on to add.
		t.Views().Main().
			Title(Equals("Final version")).
			Content(Contains("first")).
			Content(Contains("second")).
			Content(DoesNotContain("third")).
			Content(DoesNotContain("+second"))
	},
})
