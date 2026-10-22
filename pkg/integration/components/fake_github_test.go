package components

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func startFakeGithub(t *testing.T, fake *FakeGithub) *github_commands.Client {
	t.Helper()
	fake.start()
	t.Cleanup(fake.stop)
	return github_commands.NewClientWithBaseURL(
		github_commands.Repo{Host: "github.com", Owner: "owner", Name: "repo", Token: "token"},
		fake.URL())
}

func TestFakeGithubServesWhatTheClientAsksFor(t *testing.T) {
	fake := &FakeGithub{
		Runs: []*models.GithubRun{
			{RunId: 1, WorkflowId: 10, HeadBranch: "main", Event: "push", Status: "completed", Conclusion: "failure"},
			{RunId: 2, WorkflowId: 20, HeadBranch: "feature", Event: "pull_request", Status: "in_progress"},
		},
		Jobs:          map[int64][]*models.GithubJob{1: {{JobId: 100, Name: "build"}}},
		JobLogs:       map[int64]string{100: "the log\n"},
		Workflows:     []*models.GithubWorkflow{{WorkflowId: 10, Name: "CI"}},
		WorkflowFiles: map[string]string{".github/workflows/ci.yml": "on: push\n"},
		Artifacts:     map[int64][]*models.GithubArtifact{1: {{ArtifactId: 1000, Name: "dist"}}},
		ArtifactFiles: map[int64]map[string]string{1000: {"bin/app": "binary"}},
	}
	client := startFakeGithub(t, fake)
	ctx := context.Background()

	runs, err := client.ListRuns(ctx, github_commands.ListRunsOptions{})
	assert.NoError(t, err)
	assert.Len(t, runs, 2)

	runs, err = client.ListRuns(ctx, github_commands.ListRunsOptions{Status: "failure"})
	assert.NoError(t, err)
	assert.Equal(t, []int64{1}, runIds(runs))

	runs, err = client.ListRuns(ctx, github_commands.ListRunsOptions{Branch: "feature", Event: "pull_request"})
	assert.NoError(t, err)
	assert.Equal(t, []int64{2}, runIds(runs))

	runs, err = client.ListRuns(ctx, github_commands.ListRunsOptions{WorkflowId: 10})
	assert.NoError(t, err)
	assert.Equal(t, []int64{1}, runIds(runs))

	jobs, err := client.ListJobs(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubJob{{JobId: 100, Name: "build"}}, jobs)

	jobs, err = client.ListJobs(ctx, 2)
	assert.NoError(t, err)
	assert.Empty(t, jobs)

	log, err := client.GetJobLog(ctx, 100)
	assert.NoError(t, err)
	assert.Equal(t, "the log\n", log)

	workflows, err := client.ListWorkflows(ctx)
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubWorkflow{{WorkflowId: 10, Name: "CI"}}, workflows)

	content, err := client.GetFileContent(ctx, ".github/workflows/ci.yml", "main")
	assert.NoError(t, err)
	assert.Equal(t, "on: push\n", content)

	artifacts, err := client.ListArtifacts(ctx, 1)
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubArtifact{{ArtifactId: 1000, Name: "dist"}}, artifacts)

	archive, err := client.DownloadArtifact(ctx, 1000)
	assert.NoError(t, err)
	dir := t.TempDir()
	assert.NoError(t, github_commands.ExtractArtifact(archive, dir))
	extracted, err := os.ReadFile(filepath.Join(dir, "bin", "app"))
	assert.NoError(t, err)
	assert.Equal(t, "binary", string(extracted))
}

func TestFakeGithubRecordsRequests(t *testing.T) {
	fake := &FakeGithub{}
	client := startFakeGithub(t, fake)
	ctx := context.Background()

	_, err := client.ListRuns(ctx, github_commands.ListRunsOptions{Branch: "main", Limit: 5})
	assert.NoError(t, err)
	assert.NoError(t, client.RerunRun(ctx, 1, true))
	assert.NoError(t, client.CancelRun(ctx, 2))
	assert.NoError(t, client.DeleteRun(ctx, 3))
	assert.NoError(t, client.DispatchWorkflow(ctx, 10, "main", map[string]string{"env": "prod"}))
	assert.NoError(t, client.DisableWorkflow(ctx, 10))

	assert.Equal(t, []string{
		"GET /repos/owner/repo/actions/runs?branch=main&per_page=5",
		`POST /repos/owner/repo/actions/runs/1/rerun {"enable_debug_logging":true}`,
		"POST /repos/owner/repo/actions/runs/2/cancel",
		"DELETE /repos/owner/repo/actions/runs/3",
		`POST /repos/owner/repo/actions/workflows/10/dispatches {"inputs":{"env":"prod"},"ref":"main"}`,
		"PUT /repos/owner/repo/actions/workflows/10/disable",
	}, fake.Requests())
}

func TestFakeGithubRespondsWithConfiguredErrors(t *testing.T) {
	client := startFakeGithub(t, &FakeGithub{
		Errors: map[string]FakeGithubError{
			"GET /repos/owner/repo/actions/runs":          {StatusCode: 500, Message: "Server Error"},
			"POST /repos/owner/repo/actions/runs/1/rerun": {StatusCode: 403, Message: "Forbidden"},
		},
	})
	ctx := context.Background()

	_, err := client.ListRuns(ctx, github_commands.ListRunsOptions{})
	assert.Equal(t, &github_commands.APIError{StatusCode: 500, Message: "Server Error"}, err)

	err = client.RerunRun(ctx, 1, false)
	assert.Equal(t, &github_commands.APIError{StatusCode: 403, Message: "Forbidden"}, err)
}

func runIds(runs []*models.GithubRun) []int64 {
	ids := make([]int64, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.RunId)
	}
	return ids
}

func TestFakeGithubServesUpdatedData(t *testing.T) {
	fake := &FakeGithub{Runs: []*models.GithubRun{{RunId: 1}}}
	client := startFakeGithub(t, fake)
	ctx := context.Background()

	(&FakeGithubDriver{fake: fake}).Update(func(fake *FakeGithub) {
		fake.Runs = append([]*models.GithubRun{{RunId: 2}}, fake.Runs...)
	})

	runs, err := client.ListRuns(ctx, github_commands.ListRunsOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []int64{2, 1}, runIds(runs))
}
