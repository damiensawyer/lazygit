package diff

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ShowFinalVersionSkipsDirectories = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "A directory in the file tree has no single final version, so it keeps showing the diff of what's in it",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(cfg *config.AppConfig) {
		cfg.GetUserConfig().Gui.ShowFileTree = true
	},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/a.txt", "old a\n")
		shell.CreateFileAndAdd("dir/b.txt", "old b\n")
		shell.Commit("initial")
		shell.UpdateFileAndAdd("dir/a.txt", "new a\n")
		shell.Commit("change a")

		shell.UpdateFile("dir/a.txt", "newer a\n")
		shell.UpdateFile("dir/b.txt", "new b\n")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.ToggleShowFinalVersion)
		t.ExpectToast(Equals("Showing final version of files"))

		t.Views().Files().
			NavigateToLine(Contains("dir"))

		t.Views().Main().
			Title(Equals("Unstaged changes")).
			Content(Contains("+newer a")).
			Content(Contains("+new b"))

		// A file in it does show its final version.
		t.Views().Files().
			NavigateToLine(Contains("b.txt"))

		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("new b"))

		// And the same for a commit's files.
		t.Views().Commits().
			Focus().
			SelectedLine(Contains("change a")).
			PressEnter()

		t.Views().CommitFiles().
			IsFocused().
			Lines(
				Contains("dir").IsSelected(),
				Contains("a.txt"),
			)

		t.Views().Main().
			Title(Equals("Patch")).
			Content(Contains("+new a"))

		t.Views().CommitFiles().
			NavigateToLine(Contains("a.txt"))

		t.Views().Main().
			Title(Equals("Final version")).
			Content(Equals("new a"))
	},
})
