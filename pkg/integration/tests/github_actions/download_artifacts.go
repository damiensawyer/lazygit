package github_actions

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var DownloadArtifacts = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Download one of a run's artifacts, or all of them, into the working tree",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(config *config.AppConfig) {},
	SetupRepo:    setupGithubRepo,
	FakeGithub: func() *FakeGithub {
		fake := newFakeGithub()
		fake.Artifacts = map[int64][]*models.GithubArtifact{
			failedRunId: {
				{ArtifactId: 1, Name: "dist", SizeInBytes: 2048},
				{ArtifactId: 2, Name: "coverage", SizeInBytes: 100},
				{ArtifactId: 3, Name: "old-logs", SizeInBytes: 100, Expired: true},
			},
		}
		fake.ArtifactFiles = map[int64]map[string]string{
			1: {"bin/app": "the app"},
			2: {"coverage.txt": "100%"},
		}
		return fake
	}(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			Focus().
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Release v1.0")).
			Press(keys.GithubActions.DownloadArtifacts)

		t.ExpectPopup().Alert().
			Title(Equals("Error")).
			Content(Equals("The run has no artifacts")).
			Confirm()

		t.Views().GithubRuns().
			IsFocused().
			NavigateToLine(Contains("Fix the frobnicator")).
			Press(keys.GithubActions.DownloadArtifacts)

		t.ExpectPopup().Menu().
			Title(Equals("Download artifacts")).
			Lines(
				Contains("Download all").IsSelected(),
				Contains("dist").Contains("2.0 KiB"),
				Contains("coverage").Contains("100 B"),
				Contains("old-logs").Contains("expired"),
				Contains("Cancel"),
			).
			Select(Contains("old-logs")).
			Confirm()

		t.ExpectToast(Equals("Disabled: The artifact has expired"))

		t.ExpectPopup().Menu().
			Title(Equals("Download artifacts")).
			Select(Contains("dist")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Download to directory")).
			InitialText(Equals("dist")).
			Confirm()

		t.ExpectToast(Contains("Downloaded to ").Contains("/dist"))
		t.FileSystem().
			FileContent("dist/bin/app", Equals("the app"))

		t.Views().GithubRuns().
			IsFocused().
			Press(keys.GithubActions.DownloadArtifacts)

		t.ExpectPopup().Menu().
			Title(Equals("Download artifacts")).
			Select(Contains("Download all")).
			Confirm()

		t.ExpectPopup().Prompt().
			Title(Equals("Download to directory")).
			InitialText(Equals(".")).
			Clear().
			Type("artifacts").
			Confirm()

		t.ExpectToast(Contains("Downloaded to ").Contains("/artifacts"))
		t.FileSystem().
			FileContent("artifacts/dist/bin/app", Equals("the app")).
			FileContent("artifacts/coverage/coverage.txt", Equals("100%")).
			PathNotPresent("artifacts/old-logs")
	},
})
