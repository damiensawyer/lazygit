package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var KeybindingsMenu = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "The keybindings menu of the repo-files panel lists its actions, and invoking one from there works",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/file.txt", "content\n")
		shell.CreateFileAndAdd("other.txt", "other\n")
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("dir")).
			Press(keys.Universal.OptionMenu)

		t.ExpectPopup().Menu().
			Title(Equals("Keybindings")).
			ContainsLines(
				Contains("Focus file content / Toggle directory collapsed"),
			).
			ContainsLines(
				Contains("Restore to worktree"),
			).
			ContainsLines(
				Contains("Save file/directory"),
			).
			ContainsLines(
				Contains("Copy to clipboard"),
			).
			ContainsLines(
				Contains("Toggle file tree view"),
			).
			ContainsLines(
				Contains("Collapse all files"),
			).
			ContainsLines(
				Contains("Expand all files"),
			).
			Select(Contains("Focus file content / Toggle directory collapsed")).
			Confirm()

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir").IsSelected(),
				Equals("    file.txt"),
				Equals("  other.txt"),
			)
	},
})
