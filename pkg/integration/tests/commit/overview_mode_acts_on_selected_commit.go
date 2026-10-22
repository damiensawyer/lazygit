package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeActsOnSelectedCommit = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Commands that are available in overview mode act on the commit that is selected there",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file", "base content")
		shell.Commit("base")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.EmptyCommit("plain-1")
		shell.EmptyCommit("plain-2")
		shell.EmptyCommit("top")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		// base is at a different position in the overview than in the full
		// list, so acting on the wrong index would hit a different commit
		t.Views().Commits().
			Focus().
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("top").IsSelected(),
				Contains("base"),
			).
			NavigateToLine(Contains("base"))

		t.Views().Main().Content(Contains("base content"))

		t.Views().Commits().
			Press(keys.Commits.CreateTag)

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Tag name")).
			Type("new-tag").
			SwitchToDescription().
			SwitchToSummary().
			Confirm()

		t.Views().Commits().
			Lines(
				Contains("top"),
				Contains("new-tag").Contains("base").IsSelected(),
			)

		t.Views().Commits().
			PressPrimaryAction()

		t.ExpectPopup().Menu().
			Title(Contains("Checkout branch or commit")).
			Select(MatchesRegexp("Checkout commit [a-f0-9]+ as detached head")).
			Confirm()

		t.Views().Branches().
			IsFocused().
			Lines(
				Contains("* (HEAD detached at").IsSelected(),
				Contains("master"),
			)

		t.Git().
			CurrentBranchName("HEAD").
			TagNamesAt("HEAD", []string{"new-tag", "some-tag"})

		t.Views().Commits().
			Lines(
				Contains("new-tag").Contains("base"),
			)
	},
})
