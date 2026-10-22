package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ToggleTreeViewAndCollapse = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Expand/collapse all directories and switch between tree and flat view in the repo-files panel",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir1/sub/a.txt", "a\n")
		shell.CreateFileAndAdd("dir1/b.txt", "b\n")
		shell.CreateFileAndAdd("dir2/c.txt", "c\n")
		shell.CreateFileAndAdd("root.txt", "root\n")
		shell.Commit("one")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Universal.BrowseAllFilesAtCommit)

		// every directory below the top level starts out collapsed too, not
		// just the top-level ones
		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▶ dir1"),
				Equals("  ▶ dir2"),
				Equals("  root.txt"),
			).
			NavigateToLine(Contains("dir1")).
			PressEnter().
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir1").IsSelected(),
				Equals("    b.txt"),
				Equals("    ▶ sub"),
				Equals("  ▶ dir2"),
				Equals("  root.txt"),
			).
			Press(keys.Files.ExpandAll).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir1").IsSelected(),
				Equals("    b.txt"),
				Equals("    ▼ sub"),
				Equals("      a.txt"),
				Equals("  ▼ dir2"),
				Equals("    c.txt"),
				Equals("  root.txt"),
			).
			NavigateToLine(Contains("a.txt")).
			Press(keys.Files.ToggleTreeView).
			// flat view keeps the selected file selected
			Lines(
				Equals("dir1/b.txt"),
				Equals("dir1/sub/a.txt").IsSelected(),
				Equals("dir2/c.txt"),
				Equals("root.txt"),
			).
			Press(keys.Files.CollapseAll)

		t.ExpectToast(Equals("Disabled: Not available in flat view"))

		t.Views().RepoFiles().
			Press(keys.Files.ExpandAll)

		t.ExpectToast(Equals("Disabled: Not available in flat view"))

		t.Views().RepoFiles().
			Press(keys.Files.ToggleTreeView).
			Lines(
				Equals("▼ /"),
				Equals("  ▼ dir1"),
				Equals("    b.txt"),
				Equals("    ▼ sub"),
				Equals("      a.txt").IsSelected(),
				Equals("  ▼ dir2"),
				Equals("    c.txt"),
				Equals("  root.txt"),
			).
			// collapsing everything selects the top-level ancestor of the
			// selected file, which is the root item
			Press(keys.Files.CollapseAll).
			Lines(
				Equals("▶ /").IsSelected(),
			).
			Press(keys.Files.ExpandAll).
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  ▼ dir1"),
				Equals("    b.txt"),
				Equals("    ▼ sub"),
				Equals("      a.txt"),
				Equals("  ▼ dir2"),
				Equals("    c.txt"),
				Equals("  root.txt"),
			)
	},
})
