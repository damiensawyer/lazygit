package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SaveFileAtCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Save a copy of a file as it existed at a commit, both to a new path and over the original",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", "old content\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("file.txt", "new content\n")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("file.txt")).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save as...")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Save to path")).
			InitialText(Equals("file.txt")).
			Clear().
			Type("copy.txt").
			Confirm()

		t.ExpectToast(Equals("Saved to 'copy.txt'"))

		t.FileSystem().
			FileContent("copy.txt", Equals("old content\n"))

		// saving over the original location asks for confirmation because the
		// file already exists
		t.Views().RepoFiles().
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save to original location")).
			Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Save file/directory")).
			Content(Equals("'file.txt' already exists. Overwrite?")).
			Confirm()

		t.ExpectToast(Equals("Saved to 'file.txt'"))

		t.FileSystem().
			FileContent("file.txt", Equals("old content\n"))
	},
})
