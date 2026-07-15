package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var ViewFileAtCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "View the content of files as they existed at a commit in the main view",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("dir/sub.txt", "sub content\n")
		shell.CreateFileAndAdd("plain.txt", "plain content at one\n")
		shell.CreateFileAndAdd("bin.dat", "\x00\x01\x02\x03")
		shell.Commit("one")
		shell.UpdateFileAndAdd("plain.txt", "plain content at two\n")
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
				Equals("  bin.dat"),
				Equals("  ▶ dir"),
				Equals("  plain.txt"),
			).
			NavigateToLine(Contains("plain.txt"))

		// the main view shows the file's content at the selected commit, not
		// the current content
		t.Views().Main().
			Title(Equals("plain.txt")).
			Content(Contains("plain content at one"))

		// binary files get a placeholder instead of a dump of raw bytes
		t.Views().RepoFiles().
			NavigateToLine(Contains("bin.dat"))

		t.Views().Main().
			Content(Contains("Binary file (4 B)"))

		// directories show a listing of their immediate entries
		t.Views().RepoFiles().
			NavigateToLine(Contains("dir"))

		t.Views().Main().
			Title(Equals("dir")).
			Content(Contains("sub.txt"))

		// pressing enter on a file focuses the main view for scrolling
		t.Views().RepoFiles().
			NavigateToLine(Contains("plain.txt")).
			PressEnter()

		t.Views().Main().
			IsFocused().
			PressEscape()

		t.Views().RepoFiles().
			IsFocused()
	},
})
