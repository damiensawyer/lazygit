package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SaveSymlinkAndExecutable = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Saving a symlink or an executable file from the repo-files panel recreates it as it is in the commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("target.txt", "target content\n")
		shell.RunCommand([]string{"ln", "-s", "target.txt", "link"})
		shell.CreateFile("script.sh", "echo hi\n")
		shell.MakeExecutable("script.sh")
		shell.GitAddAll()
		shell.Commit("one")
		shell.DeleteFile("link")
		shell.DeleteFile("script.sh")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Lines(
				Equals("▼ /"),
				Equals("   D link"),
				Equals("   D script.sh"),
			)

		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			NavigateToLine(Equals("  link")).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save to original location")).
			Confirm()

		t.ExpectToast(Equals("Saved to 'link'"))

		t.Views().RepoFiles().
			NavigateToLine(Equals("  script.sh")).
			Press(keys.RepoFiles.SaveFile)

		t.ExpectPopup().Menu().
			Title(Equals("Save file/directory")).
			Select(Contains("Save to original location")).
			Confirm()

		t.ExpectToast(Equals("Saved to 'script.sh'"))

		// the saved entries match the commit again, so there's nothing left
		// to restore
		t.Views().Files().
			/* EXPECTED:
			IsEmpty()
			ACTUAL: */
			Lines(
				Equals("▼ /"),
				Equals("   T link"),
				Equals("   M script.sh"),
			)
	},
})
