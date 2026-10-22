package demo

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var BrowseFiles = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse all files of the repo as they were at a commit",
	ExtraCmdArgs: []string{},
	Skip:         false,
	IsDemo:       true,
	SetupConfig: func(config *config.AppConfig) {
		setFleetDemoConfig(config)
	},
	SetupRepo: func(shell *Shell) {
		setupFleetRepo(shell)
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.SetCaptionPrefix("Browse all files at a commit")

		t.Views().Commits().
			Focus().
			Wait(1000).
			NavigateToLine(Contains("Merge branch 'ship-classes'")).
			Wait(1000).
			Press(keys.Universal.BrowseAllFilesAtCommit)

		t.Views().RepoFiles().
			IsFocused().
			Title(Contains("Merge branch 'ship-classes'")).
			Wait(2000).
			SetCaptionPrefix("Expand directories and view files").
			NavigateToLine(Contains("internal")).
			Wait(800).
			PressEnter().
			Wait(800).
			NavigateToLine(Contains("server")).
			Wait(600).
			PressEnter().
			Wait(800).
			NavigateToLine(Contains("server.go")).
			Wait(2500).
			NavigateToLine(Contains("store")).
			Wait(800).
			PressEnter().
			Wait(600).
			NavigateToLine(Contains("store.go")).
			Tap(func() {
				t.Views().Main().Content(Contains("type Ship struct"))
			}).
			Wait(2500).
			SetCaptionPrefix("Filter files by name").
			Press(keys.Universal.StartSearch).
			Tap(func() {
				t.ExpectSearch().Type("server")
				t.Wait(1500)
				t.ExpectSearch().Confirm()
			}).
			Lines(
				Contains("internal/server"),
				Contains("server.go"),
			).
			Wait(800).
			NavigateToLine(Contains("server.go")).
			Wait(1500).
			SetCaptionPrefix("Copy a file's path or content").
			Press(keys.Files.CopyFileInfoToClipboard).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Copy to clipboard")).
					Wait(2000).
					Cancel()
			}).
			Wait(800).
			SetCaptionPrefix("Save a file as it was at the commit").
			Press(keys.RepoFiles.SaveFile).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Save file/directory")).
					Wait(2000).
					Cancel()
			}).
			Wait(800).
			SetCaptionPrefix("").
			PressEscape().
			Wait(800).
			PressEscape()

		t.Views().Commits().
			IsFocused().
			Wait(1500)
	},
})
