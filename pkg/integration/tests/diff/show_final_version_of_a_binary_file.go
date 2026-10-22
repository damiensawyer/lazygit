package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionOfABinaryFile = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Showing the final version of a binary file explains that it can't be shown rather than writing its bytes to the view",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("committed.bin", "abc\x00def")
		shell.Commit("add binary")

		shell.CreateFile("untracked.bin", "ghi\x00jkl")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			SelectedLine(Contains("untracked.bin")).
			Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("This is a binary file, so its contents can't be shown."))

		t.Views().Commits().
			Focus().
			SelectedLine(Contains("add binary")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			SelectedLine(Contains("committed.bin"))

		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("This is a binary file, so its contents can't be shown."))
	},
})
