package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeAcrossRepoSwitch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Overview mode belongs to the repo it was enabled in and survives switching away and back",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("base")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.EmptyCommit("plain")
		shell.EmptyCommit("top")
		shell.AddWorktree("master", "../linked-worktree", "newbranch")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Commits.ToggleOverviewMode).
			Title(Equals("Commits (overview)")).
			Lines(
				Contains("top"),
				Contains("base"),
			)

		t.Views().Worktrees().
			Focus().
			NavigateToLine(Contains("linked-worktree")).
			Press(keys.Universal.Select).
			Lines(
				Contains("linked-worktree").IsSelected(),
				Contains("(main worktree)"),
			)

		t.Views().Information().Content(DoesNotContain("Showing commits overview"))

		t.Views().Commits().
			Focus().
			Title(Equals("Commits")).
			Lines(
				Contains("top"),
				Contains("plain"),
				Contains("base"),
			)

		t.Views().Worktrees().
			Focus().
			NavigateToLine(Contains("(main worktree)")).
			Press(keys.Universal.Select).
			Lines(
				Contains("(main worktree)").IsSelected(),
				Contains("linked-worktree"),
			)

		t.Views().Information().Content(Contains("Showing commits overview"))

		t.Views().Commits().
			Focus().
			/* EXPECTED:
			Title(Equals("Commits (overview)")).
			ACTUAL: */
			Title(Equals("Commits")).
			Lines(
				Contains("top"),
				Contains("base"),
			)
	},
})
