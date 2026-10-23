package ui

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var MenuGoBack = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Go into a submenu with the right arrow and back out of it with the left arrow",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.
			CloneIntoRemote("origin").
			NewBranch("feature").
			EmptyCommit("feature commit").
			PushBranchAndSetUpstream("origin", "feature").
			NewBranch("base").
			EmptyCommit("base commit")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Branches().
			Focus().
			NavigateToLine(Contains("feature")).
			Press(keys.Branches.SetUpstream)

		t.ExpectPopup().Menu().
			Title(Equals("Upstream options"))

		t.Views().Menu().
			// There's nothing to go back to from the first menu
			Press(keys.Universal.PrevBlock).
			IsFocused().
			NavigateToLine(Contains("Reset checked-out branch onto origin/feature...")).
			Press(keys.Universal.NextBlock)

		t.ExpectPopup().Menu().
			Title(Equals("Upstream options › Reset to origin/feature"))

		t.Views().Menu().
			NavigateToLine(Contains("Hard reset")).
			Press(keys.Universal.PrevBlock)

		t.ExpectPopup().Menu().
			Title(Equals("Upstream options"))

		t.Views().Menu().
			// The item the submenu was opened from is still selected
			SelectedLines(Contains("Reset checked-out branch onto origin/feature...")).
			PressEnter()

		t.ExpectPopup().Menu().
			Title(Equals("Upstream options › Reset to origin/feature")).
			// Escape closes all of the menus, not just the submenu
			Cancel()

		t.Views().Branches().
			IsFocused()
	},
})
