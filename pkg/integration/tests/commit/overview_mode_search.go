package commit

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var OverviewModeSearch = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Searching in overview mode only finds the commits it shows",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.EmptyCommit("match-tagged")
		shell.CreateLightweightTag("some-tag", "HEAD")
		shell.EmptyCommit("match-plain")
		shell.EmptyCommit("top")
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Commits().
			Focus().
			Press(keys.Commits.ToggleOverviewMode).
			Lines(
				Contains("top").IsSelected(),
				Contains("match-tagged"),
			).
			Press(keys.Universal.StartSearch).
			Tap(func() {
				t.ExpectSearch().
					Type("match").
					Confirm()

				t.Views().Search().IsVisible().Content(Contains("matches for 'match' (1 of 1)"))
			}).
			Lines(
				Contains("top"),
				Contains("match-tagged").IsSelected(),
			).
			Press(keys.Universal.StartSearch).
			Tap(func() {
				t.ExpectSearch().
					Clear().
					Type("plain").
					Confirm()

				t.Views().Search().IsVisible().Content(Contains("No matches for 'plain'"))
			}).
			Lines(
				Contains("top"),
				Contains("match-tagged").IsSelected(),
			)
	},
})
