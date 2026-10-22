package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RestoreRangeSelection = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Restore a range selection of files and directories from the repo-files panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("a.txt", "a at one\n")
		shell.CreateFileAndAdd("dir/b.txt", "b at one\n")
		shell.CreateFileAndAdd("z.txt", "z at one\n")
		shell.Commit("one")
		shell.UpdateFileAndAdd("a.txt", "a at two\n")
		shell.UpdateFileAndAdd("dir/b.txt", "b at two\n")
		shell.CreateFileAndAdd("dir/added.txt", "added at two\n")
		shell.UpdateFileAndAdd("z.txt", "z at two\n")
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			NavigateToLine(Contains("one")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  a.txt"),
				Equals("  ▶ dir"),
				Equals("  z.txt"),
			).
			NavigateToLine(Contains("dir")).
			PressEnter().
			NavigateToLine(Contains("a.txt")).
			Press(keys.Universal.ToggleRangeSelect).
			NavigateToLine(Contains("b.txt")).
			SelectedLines(
				Equals("  a.txt"),
				Equals("  ▼ dir"),
				Equals("    b.txt"),
			).
			Press(keys.RepoFiles.Restore)

		// dir/b.txt is covered by dir, so it isn't listed separately
		t.ExpectPopup().Confirmation().
			Title(Equals("Restore to worktree")).
			Content(Contains("Are you sure you want to replace 'a.txt, dir' in your working tree with the version at")).
			Confirm()

		// files that didn't exist at the commit are removed from a restored
		// directory, and files outside the selection are left alone
		t.FileSystem().
			FileContent("a.txt", Equals("a at one\n")).
			FileContent("dir/b.txt", Equals("b at one\n")).
			PathNotPresent("dir/added.txt").
			FileContent("z.txt", Equals("z at two\n"))

		t.Views().RepoFiles().
			PressEscape()

		t.Views().Files().
			Focus().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("   M a.txt"),
				Equals("  ▼ dir"),
				Equals("     D added.txt"),
				Equals("     M b.txt"),
			)
	},
})
