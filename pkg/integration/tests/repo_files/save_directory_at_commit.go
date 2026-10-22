package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SaveDirectoryAtCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Save a copy of a directory as it existed at a commit, both to a new path and over the original",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/a.txt", "a at one\n")
		shell.CreateFileAndAdd("dir/nested/b.txt", "b at one\n")
		shell.CreateFileAndAdd("dir-other/c.txt", "c at one\n")
		shell.Commit("one")
		shell.CloneIntoSubmodule("my_submodule", "dir/module")
		shell.GitAddAll()
		shell.UpdateFileAndAdd("dir/a.txt", "a at two\n")
		shell.DeleteFileAndAdd("dir/nested/b.txt")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Lines(
				Contains("two").IsSelected(),
				Contains("one"),
			).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Equals("  ▶ dir")).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save as...")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Save to path")).
			InitialText(Equals("dir")).
			Clear().
			Type("copy/of/dir").
			Confirm()

		t.ExpectToast(Equals("Saved to 'copy/of/dir'"))

		// the submodule is skipped, and sibling directories sharing the
		// name's prefix aren't included
		t.FileSystem().
			FileContent("copy/of/dir/a.txt", Equals("a at two\n")).
			PathNotPresent("copy/of/dir/module").
			PathNotPresent("copy/of/dir/c.txt").
			PathNotPresent("copy/of/dir-other")

		t.Views().RepoFiles().
			PressEscape()

		t.Views().Commits().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Equals("  ▶ dir")).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save to original location")).
			Confirm()

		t.ExpectPopup().Confirmation().
			Title(Equals("Save file/directory")).
			Content(Equals("'dir' already exists. Overwrite?")).
			Confirm()

		t.ExpectToast(Equals("Saved to 'dir'"))

		// files that were deleted since are recreated, but nothing is
		// removed, unlike with restore
		t.FileSystem().
			FileContent("dir/a.txt", Equals("a at one\n")).
			FileContent("dir/nested/b.txt", Equals("b at one\n")).
			PathPresent("dir/module")
	},
})
