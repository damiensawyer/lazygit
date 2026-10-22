package github_actions

import (
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

const (
	ciWorkflowId      = 10
	releaseWorkflowId = 20

	failedRunId     = 101
	inProgressRunId = 102
	succeededRunId  = 103
	cancelledRunId  = 104

	buildJobId = 1001
	testJobId  = 1002
)

func setupGithubRepo(shell *Shell) {
	shell.EmptyCommit("initial commit")
	shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.com/owner/repo.git"})
}

// the times of the runs, all within an hour on the same day
func at(minute int, second int) time.Time {
	return time.Date(2024, time.March, 1, 10, minute, second, 0, time.UTC)
}

func githubRun(id int64, workflow string, workflowId int64, number int64, title string, status string, conclusion string) *models.GithubRun {
	return &models.GithubRun{
		RunId:        id,
		Name:         workflow,
		DisplayTitle: title,
		Number:       number,
		Attempt:      1,
		Event:        "push",
		Status:       status,
		Conclusion:   conclusion,
		WorkflowId:   workflowId,
		HeadBranch:   "main",
		HeadSha:      "0123456789abcdef0123456789abcdef01234567",
		Path:         ".github/workflows/ci.yml",
		Url:          fmt.Sprintf("https://github.com/owner/repo/actions/runs/%d", id),
		CreatedAt:    at(0, 0),
		UpdatedAt:    at(5, 0),
		RunStartedAt: at(0, 0),
		Actor:        models.GithubUser{Login: "octocat"},
	}
}

// githubRuns returns runs in each of the states worth telling apart, newest
// first, the way the API lists them.
func githubRuns() []*models.GithubRun {
	inProgress := githubRun(inProgressRunId, "CI", ciWorkflowId, 13, "Add a widget", "in_progress", "")
	inProgress.HeadBranch = "feature"
	inProgress.Event = "pull_request"

	failed := githubRun(failedRunId, "CI", ciWorkflowId, 12, "Fix the frobnicator", models.GithubStatusCompleted, models.GithubConclusionFailure)

	succeeded := githubRun(succeededRunId, "Release", releaseWorkflowId, 3, "Release v1.0", models.GithubStatusCompleted, models.GithubConclusionSuccess)
	succeeded.Event = "workflow_dispatch"
	succeeded.Path = ".github/workflows/release.yml"

	cancelled := githubRun(cancelledRunId, "CI", ciWorkflowId, 11, "Try something", models.GithubStatusCompleted, models.GithubConclusionCancelled)
	cancelled.HeadBranch = "experiment"

	return []*models.GithubRun{inProgress, failed, succeeded, cancelled}
}

func githubStep(number int, name string, conclusion string, start time.Time, end time.Time) models.GithubStep {
	return models.GithubStep{
		Number:      number,
		Name:        name,
		Status:      models.GithubStatusCompleted,
		Conclusion:  conclusion,
		StartedAt:   start,
		CompletedAt: end,
	}
}

// the jobs of the failed run: one that succeeded, and one that failed
func githubJobs() []*models.GithubJob {
	return []*models.GithubJob{
		{
			JobId:       buildJobId,
			RunId:       failedRunId,
			Name:        "build",
			Status:      models.GithubStatusCompleted,
			Conclusion:  models.GithubConclusionSuccess,
			StartedAt:   at(0, 0),
			CompletedAt: at(1, 0),
			Url:         "https://github.com/owner/repo/actions/runs/101/job/1001",
			Steps: []models.GithubStep{
				githubStep(1, "Set up job", models.GithubConclusionSuccess, at(0, 0), at(0, 5)),
				githubStep(2, "Build", models.GithubConclusionSuccess, at(0, 6), at(1, 0)),
			},
		},
		{
			JobId:       testJobId,
			RunId:       failedRunId,
			Name:        "test",
			Status:      models.GithubStatusCompleted,
			Conclusion:  models.GithubConclusionFailure,
			StartedAt:   at(1, 0),
			CompletedAt: at(3, 0),
			Url:         "https://github.com/owner/repo/actions/runs/101/job/1002",
			Steps: []models.GithubStep{
				githubStep(1, "Set up job", models.GithubConclusionSuccess, at(1, 0), at(1, 5)),
				githubStep(2, "Run tests", models.GithubConclusionFailure, at(1, 6), at(2, 0)),
				githubStep(3, "Upload coverage", models.GithubConclusionSkipped, time.Time{}, time.Time{}),
			},
		},
	}
}

func githubJobLogs() map[int64]string {
	return map[int64]string{
		buildJobId: "2024-03-01T10:00:00.0000000Z Preparing the runner\n" +
			"2024-03-01T10:00:06.0000000Z ##[group]Run make build\n" +
			"2024-03-01T10:00:30.0000000Z built it\n",
		testJobId: "2024-03-01T10:01:00.0000000Z Preparing the test runner\n" +
			"2024-03-01T10:01:06.0000000Z ##[group]Run make test\n" +
			"2024-03-01T10:01:30.0000000Z frobnicator_test.go:12: expected 1, got 2\n" +
			"2024-03-01T10:01:59.0000000Z ##[error]Process completed with exit code 1.\n",
	}
}

// a repository whose runs are in each of the states worth telling apart, and
// whose failed run has jobs with logs
func newFakeGithub() *FakeGithub {
	return &FakeGithub{
		Runs:    githubRuns(),
		Jobs:    map[int64][]*models.GithubJob{failedRunId: githubJobs()},
		JobLogs: githubJobLogs(),
		Annotations: map[int64][]*models.GithubAnnotation{
			testJobId: {{Path: "frobnicator_test.go", StartLine: 12, Level: "failure", Message: "expected 1, got 2"}},
		},
		Workflows: []*models.GithubWorkflow{
			{WorkflowId: ciWorkflowId, Name: "CI", Path: ".github/workflows/ci.yml", State: "active"},
			{WorkflowId: releaseWorkflowId, Name: "Release", Path: ".github/workflows/release.yml", State: "active"},
		},
	}
}

const nightlyWorkflowId = 30

// like newFakeGithub, but with a disabled workflow too, and with the
// workflows' definitions
func newFakeGithubWithWorkflowFiles() *FakeGithub {
	fake := newFakeGithub()
	fake.Workflows = append(fake.Workflows,
		&models.GithubWorkflow{WorkflowId: nightlyWorkflowId, Name: "Nightly", Path: ".github/workflows/nightly.yml", State: "disabled_manually"})
	fake.WorkflowFiles = map[string]string{
		".github/workflows/ci.yml": "on: [push, pull_request]\n",
		".github/workflows/release.yml": `on:
  workflow_dispatch:
    inputs:
      environment:
        description: Where to deploy
        type: choice
        options: [staging, production]
        default: staging
      version:
        description: Version to release
        required: true
      dry_run:
        type: boolean
        default: "false"
`,
		".github/workflows/nightly.yml": "on: workflow_dispatch\n",
	}
	return fake
}
