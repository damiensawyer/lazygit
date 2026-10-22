package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var SymlinksAndSubmodules = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Symlinks and submodules in the repo-files panel are shown as such rather than as file contents",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("target.txt", "target content\n")
		shell.Commit("one")
		shell.CloneIntoSubmodule("my_submodule", "lib/sub")
		shell.RunCommand([]string{"ln", "-s", "target.txt", "link"})
		shell.GitAddAll()
		shell.Commit("two")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  .gitmodules"),
				Equals("  ▶ lib"),
				Equals("  link"),
				Equals("  target.txt"),
			).
			NavigateToLine(Contains("link"))

		t.Views().Main().
			Title(Equals("link")).
			Content(Equals("Symlink to target.txt"))

		t.Views().RepoFiles().
			NavigateToLine(Contains("lib")).
			PressEnter().
			Lines(
				Equals("▼ /"),
				Equals("  .gitmodules"),
				Equals("  ▼ lib").IsSelected(),
				Equals("    sub (submodule)"),
				Equals("  link"),
				Equals("  target.txt"),
			)

		// the directory listing doesn't mark the submodule as a directory
		t.Views().Main().
			Title(Equals("lib")).
			Content(Equals("sub"))

		t.Views().RepoFiles().
			NavigateToLine(Contains("sub (submodule)"))

		t.Views().Main().
			Title(Equals("lib/sub")).
			Content(Contains("Submodule at "))

		t.Views().RepoFiles().
			Press(keys.RepoFiles.SaveFile)

		t.ExpectToast(Equals("Disabled: Cannot save a submodule"))
	},
})
