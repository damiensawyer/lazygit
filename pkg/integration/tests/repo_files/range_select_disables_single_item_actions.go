package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var RangeSelectDisablesSingleItemActions = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Actions of the repo-files panel that work on a single item are disabled for a range selection",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("a.txt", "a\n")
		shell.CreateFileAndAdd("b.txt", "b\n")
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Contains("a.txt")).
			Press(keys.Universal.ToggleRangeSelect).
			NavigateToLine(Contains("b.txt")).
			SelectedLines(
				Equals("  a.txt"),
				Equals("  b.txt"),
			).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectToast(Equals("Disabled: Action does not support range selection, please select a single item"))

		t.Views().RepoFiles().
			Press(keys.Universal.OpenFile)

		t.ExpectToast(Equals("Disabled: Action does not support range selection, please select a single item"))

		t.Views().RepoFiles().
			PressEnter()

		t.ExpectToast(Equals("Disabled: Action does not support range selection, please select a single item"))

		t.Views().RepoFiles().
			IsFocused().
			Press(keys.Files.CopyFileInfoToClipboard)

		t.ExpectPopup().Menu().
			Title(Equals("Copy to clipboard")).
			Select(Contains("File name")).
			Tooltip(Contains("Disabled: Action does not support range selection")).
			Select(Contains("Relative path")).
			Tooltip(Contains("Disabled: Action does not support range selection")).
			Select(Contains("Absolute path")).
			Tooltip(Contains("Disabled: Action does not support range selection")).
			Select(Contains("Content of selected file")).
			Tooltip(Contains("Disabled: Action does not support range selection")).
			Select(Contains("Blob hash")).
			Tooltip(Contains("Disabled: Action does not support range selection")).
			Cancel()
	},
})
