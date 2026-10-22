package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SaveRootDirectory = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Save a copy of the whole repository as it existed at a commit by saving the root item",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("a.txt", "a\n")
		shell.CreateFileAndAdd("dir/b.txt", "b\n")
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Equals("▼ /")).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save as...")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Save to path")).
			InitialText(Equals(".")).
			Clear().
			Type("out").
			Confirm()

		t.ExpectToast(Equals("Saved to 'out'"))

		t.FileSystem().
			/* EXPECTED:
			FileContent("out/a.txt", Equals("a\n")).
			FileContent("out/dir/b.txt", Equals("b\n"))
			ACTUAL: */
			FileContent("outa.txt", Equals("a\n")).
			FileContent("outdir/b.txt", Equals("b\n"))
	},
})
