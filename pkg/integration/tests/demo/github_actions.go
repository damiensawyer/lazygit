package demo

import (
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var GithubActions = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Browse a repository's GitHub Actions runs, their jobs and their logs",
	ExtraCmdArgs: []string{},
	Skip:         false,
	IsDemo:       true,
	SetupConfig: func(config *config.AppConfig) {
		setFleetDemoConfig(config)
	},
	SetupRepo: func(shell *Shell) {
		setupFleetRepo(shell)
		shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.com/fleet-org/fleet.git"})
	},
	FakeGithub: fleetGithub(),
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.SetCaptionPrefix("View GitHub Actions")

		t.Views().Files().
			IsFocused().
			Wait(1000).
			Press(keys.Universal.OpenGithubActions)

		t.Views().GithubRuns().
			IsFocused().
			Title(Equals("Actions · fleet-org/fleet")).
			Wait(2000).
			NavigateToLine(Contains("Include latency in request logs")).
			Wait(1000).
			NavigateToLine(Contains("Release v0.4.0")).
			Wait(1500).
			SetCaptionPrefix("Filter the runs").
			Press(keys.GithubActions.FilterRuns).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Filter runs")).
					Wait(800).
					Select(Contains("Runs with status")).
					Wait(600).
					Confirm()

				t.ExpectPopup().Menu().
					Title(Equals("Runs with status")).
					Wait(600).
					Select(Contains("Failed")).
					Wait(600).
					Confirm()
			}).
			Title(Equals("Actions · fleet-org/fleet · Failed")).
			Wait(1500).
			SetCaptionPrefix("Drill into a run's jobs and steps").
			NavigateToLine(Contains("Count requests per route")).
			Wait(800).
			PressEnter()

		t.Views().GithubJobs().
			IsFocused().
			Title(Equals("Jobs · Count requests per route")).
			Wait(2000).
			NavigateToLine(Contains("test")).
			Wait(2000).
			PressEnter()

		t.Views().GithubSteps().
			IsFocused().
			Title(Equals("Steps · test")).
			Tap(func() {
				t.Views().Main().Content(Contains("expected 3 routes, got 2"))
			}).
			Wait(2500).
			NavigateToLine(Contains("Set up Go")).
			Wait(1500).
			NavigateToLine(Contains("Upload coverage")).
			Wait(1500).
			PressEscape()

		t.Views().GithubJobs().
			IsFocused().
			Wait(800).
			PressEscape()

		t.Views().GithubRuns().
			IsFocused().
			Wait(800).
			SetCaptionPrefix("Run, enable and disable workflows").
			Press(keys.GithubActions.ViewWorkflows).
			Tap(func() {
				t.ExpectPopup().Menu().
					Title(Equals("Workflows")).
					Wait(1000).
					Select(Contains("Release")).
					Wait(600).
					Confirm()

				t.ExpectPopup().Menu().
					Title(Equals("Release")).
					Wait(2500).
					Cancel()
			}).
			Wait(800).
			SetCaptionPrefix("").
			PressEscape()

		t.Views().Commits().
			IsFocused().
			Wait(1500)
	},
})

const (
	fleetCIWorkflowId      = 1
	fleetReleaseWorkflowId = 2
	fleetNightlyWorkflowId = 3

	fleetFailedRunId = 207

	fleetLintJobId  = 3001
	fleetBuildJobId = 3002
	fleetTestJobId  = 3003
)

// a GitHub repository for the fleet repo, whose latest run on the
// fleet-metrics branch failed its tests
func fleetGithub() *FakeGithub {
	now := time.Now().Truncate(time.Second)
	ago := func(d time.Duration) time.Time { return now.Add(-d) }

	run := func(id int64, workflow string, workflowId int64, number int64, title string, branch string, event string, conclusion string, started time.Duration, took time.Duration) *models.GithubRun {
		status := models.GithubStatusCompleted
		if conclusion == "" {
			status = "in_progress"
		}
		path := ".github/workflows/ci.yml"
		if workflowId == fleetReleaseWorkflowId {
			path = ".github/workflows/release.yml"
		}
		return &models.GithubRun{
			RunId:        id,
			Name:         workflow,
			DisplayTitle: title,
			Number:       number,
			Attempt:      1,
			Event:        event,
			Status:       status,
			Conclusion:   conclusion,
			WorkflowId:   workflowId,
			HeadBranch:   branch,
			HeadSha:      fmt.Sprintf("%040d", id),
			Path:         path,
			Url:          fmt.Sprintf("https://github.com/fleet-org/fleet/actions/runs/%d", id),
			CreatedAt:    ago(started),
			UpdatedAt:    ago(started - took),
			RunStartedAt: ago(started),
			Actor:        models.GithubUser{Login: "yang-wenli"},
		}
	}

	success := models.GithubConclusionSuccess
	failure := models.GithubConclusionFailure

	runs := []*models.GithubRun{
		run(209, "CI", fleetCIWorkflowId, 58, "Explain how to run the server", "main", "push", "", 2*time.Minute, 0),
		run(208, "CI", fleetCIWorkflowId, 57, "Include latency in request logs", "main", "push", success, 3*time.Hour, 4*time.Minute),
		run(fleetFailedRunId, "CI", fleetCIWorkflowId, 56, "Count requests per route", "fleet-metrics", "pull_request", failure, 5*time.Hour, 3*time.Minute),
		run(206, "Release", fleetReleaseWorkflowId, 9, "Release v0.4.0", "main", "workflow_dispatch", success, 26*time.Hour, 7*time.Minute),
		run(205, "CI", fleetCIWorkflowId, 55, "Default unknown classes to scout", "main", "push", success, 28*time.Hour, 4*time.Minute),
		run(204, "CI", fleetCIWorkflowId, 54, "Expose Prometheus metrics", "fleet-metrics", "pull_request", failure, 30*time.Hour, 2*time.Minute),
		run(203, "CI", fleetCIWorkflowId, 53, "Reject negative crew", "main", "push", models.GithubConclusionCancelled, 50*time.Hour, time.Minute),
		run(202, "CI", fleetCIWorkflowId, 52, "Add shutdown timeout flag", "main", "push", success, 75*time.Hour, 4*time.Minute),
	}

	runStart := ago(5 * time.Hour)
	at := func(seconds int) time.Time { return runStart.Add(time.Duration(seconds) * time.Second) }
	step := func(number int, name string, conclusion string, start int, end int) models.GithubStep {
		return models.GithubStep{
			Number:      number,
			Name:        name,
			Status:      models.GithubStatusCompleted,
			Conclusion:  conclusion,
			StartedAt:   at(start),
			CompletedAt: at(end),
		}
	}
	job := func(id int64, name string, conclusion string, start int, end int, steps ...models.GithubStep) *models.GithubJob {
		return &models.GithubJob{
			JobId:       id,
			RunId:       fleetFailedRunId,
			Name:        name,
			Status:      models.GithubStatusCompleted,
			Conclusion:  conclusion,
			StartedAt:   at(start),
			CompletedAt: at(end),
			Url:         fmt.Sprintf("https://github.com/fleet-org/fleet/actions/runs/%d/job/%d", fleetFailedRunId, id),
			Steps:       steps,
		}
	}

	jobs := []*models.GithubJob{
		job(fleetLintJobId, "lint", success, 0, 41,
			step(1, "Set up job", success, 0, 3),
			step(2, "Set up Go", success, 3, 12),
			step(3, "golangci-lint", success, 12, 41),
		),
		job(fleetBuildJobId, "build", success, 0, 58,
			step(1, "Set up job", success, 0, 2),
			step(2, "Set up Go", success, 2, 11),
			step(3, "Build", success, 11, 58),
		),
		job(fleetTestJobId, "test", failure, 0, 103,
			step(1, "Set up job", success, 0, 3),
			step(2, "Set up Go", success, 3, 13),
			step(3, "Run tests", failure, 13, 103),
			models.GithubStep{Number: 4, Name: "Upload coverage", Status: models.GithubStatusCompleted, Conclusion: models.GithubConclusionSkipped},
		),
	}

	logLine := func(seconds int, text string) string {
		return at(seconds).UTC().Format("2006-01-02T15:04:05.0000000Z") + " " + text + "\n"
	}

	logs := map[int64]string{
		fleetLintJobId: logLine(0, "Preparing the runner") +
			logLine(3, "##[group]Run actions/setup-go@v5") +
			logLine(11, "Installed go 1.23.4") +
			logLine(12, "##[group]Run golangci-lint run") +
			logLine(40, "0 issues."),
		fleetBuildJobId: logLine(0, "Preparing the runner") +
			logLine(2, "##[group]Run actions/setup-go@v5") +
			logLine(10, "Installed go 1.23.4") +
			logLine(11, "##[group]Run go build ./...") +
			logLine(57, "Built example.com/fleet/cmd"),
		fleetTestJobId: logLine(0, "Preparing the runner") +
			logLine(3, "##[group]Run actions/setup-go@v5") +
			logLine(12, "Installed go 1.23.4") +
			logLine(13, "##[group]Run go test ./...") +
			logLine(60, "ok  \texample.com/fleet/internal/store\t0.012s") +
			logLine(101, "--- FAIL: TestMetrics (0.01s)") +
			logLine(101, "    server_test.go:42: expected 3 routes, got 2") +
			logLine(102, "FAIL\texample.com/fleet/internal/server\t0.015s") +
			logLine(102, "##[error]Process completed with exit code 1."),
	}

	return &FakeGithub{
		Runs:    runs,
		Jobs:    map[int64][]*models.GithubJob{fleetFailedRunId: jobs},
		JobLogs: logs,
		Annotations: map[int64][]*models.GithubAnnotation{
			fleetTestJobId: {{Path: "internal/server/server_test.go", StartLine: 42, Level: "failure", Message: "expected 3 routes, got 2"}},
		},
		Workflows: []*models.GithubWorkflow{
			{WorkflowId: fleetCIWorkflowId, Name: "CI", Path: ".github/workflows/ci.yml", State: "active"},
			{WorkflowId: fleetReleaseWorkflowId, Name: "Release", Path: ".github/workflows/release.yml", State: "active"},
			{WorkflowId: fleetNightlyWorkflowId, Name: "Nightly", Path: ".github/workflows/nightly.yml", State: "disabled_manually"},
		},
		WorkflowFiles: map[string]string{
			".github/workflows/ci.yml":      "on: [push, pull_request]\n",
			".github/workflows/release.yml": "on:\n  workflow_dispatch:\n    inputs:\n      version:\n        description: Version to release\n        required: true\n",
			".github/workflows/nightly.yml": "on: workflow_dispatch\n",
		},
	}
}
