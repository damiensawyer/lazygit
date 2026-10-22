package repo_files

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var EnterFromTags = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse the files of lightweight and annotated tags",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("one.txt", "one\n")
		shell.Commit("one")
		shell.CreateLightweightTag("lightweight", "HEAD")
		shell.CreateFileAndAdd("two.txt", "two\n")
		shell.Commit("two")
		// an annotated tag points at a tag object rather than a commit, so it
		// must be peeled to get at the commit's tree
		shell.CreateAnnotatedTag("annotated", "annotated message", "HEAD")
		shell.CreateFileAndAdd("three.txt", "three\n")
		shell.Commit("three")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Tags().
			Focus().
			Lines(
				Contains("annotated").IsSelected(),
				Contains("lightweight"),
			).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("▼ /").IsSelected(),
				Equals("  one.txt"),
				Equals("  two.txt"),
			).
			PressEscape()

		t.Views().Tags().
			IsFocused().
			NavigateToLine(Contains("lightweight")).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Lines(
				Equals("one.txt").IsSelected(),
			).
			PressEscape()

		t.Views().Tags().
			IsFocused()
	},
})
