package helpers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestGithubTokens(t *testing.T) {
	var tokens GithubTokens
	assert.Empty(t, tokens.get("github.com"))

	tokens.set("github.com", "gho_one")
	tokens.set("git.acme.com", "ghe_two")
	assert.Equal(t, "gho_one", tokens.get("github.com"))
	assert.Equal(t, "ghe_two", tokens.get("git.acme.com"))
	assert.Empty(t, tokens.get("GitHub.com"))

	tokens.set("github.com", "gho_three")
	assert.Equal(t, "gho_three", tokens.get("github.com"))

	// Forgetting a token that turned out to be invalid.
	tokens.set("github.com", "")
	assert.Empty(t, tokens.get("github.com"))
	assert.Equal(t, "ghe_two", tokens.get("git.acme.com"))
}

func TestGithubTokensConcurrentAccess(t *testing.T) {
	var tokens GithubTokens
	var wg sync.WaitGroup
	for i := range 20 {
		host := fmt.Sprintf("host%d", i%4)
		wg.Go(func() {
			tokens.set(host, "token")
			_ = tokens.get(host)
		})
	}
	wg.Wait()

	for i := range 4 {
		assert.Equal(t, "token", tokens.get(fmt.Sprintf("host%d", i)))
	}
}

// newFakeGithubClient returns a client talking to a fake server that answers
// each path with the response given for it, and 404 for any other, and the
// paths that were requested.
func newFakeGithubClient(t *testing.T, responses map[string]string) (*github_commands.Client, func() []string) {
	t.Helper()

	var mutex sync.Mutex
	paths := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		paths = append(paths, r.URL.Path)
		mutex.Unlock()

		response, ok := responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Not Found"}`))
			return
		}
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	client := github_commands.NewClientWithBaseURL(github_commands.Repo{Owner: "owner", Name: "repo"}, server.URL)
	return client, func() []string {
		mutex.Lock()
		defer mutex.Unlock()
		return append([]string{}, paths...)
	}
}

func TestLoadRunDetailsLoadsAnnotationsOfFailedJobs(t *testing.T) {
	client, requestedPaths := newFakeGithubClient(t, map[string]string{
		"/repos/owner/repo/actions/runs/7/jobs": `{"total_count": 4, "jobs": [
			{"id": 1, "name": "lint", "status": "completed", "conclusion": "success"},
			{"id": 2, "name": "test", "status": "completed", "conclusion": "failure"},
			{"id": 3, "name": "e2e", "status": "completed", "conclusion": "cancelled"},
			{"id": 4, "name": "deploy", "status": "in_progress"}
		]}`,
		"/repos/owner/repo/check-runs/2/annotations": `[{"annotation_level": "failure", "message": "boom"}]`,
		// the annotations of job 3 fail to load
	})
	updatedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	details := loadRunDetails(client, &models.GithubRun{RunId: 7, UpdatedAt: updatedAt})

	assert.NoError(t, details.Err)
	assert.Equal(t, []string{"lint", "test", "e2e", "deploy"}, jobNames(details.Jobs))
	assert.Equal(t, map[int64][]*models.GithubAnnotation{
		2: {{Level: "failure", Message: "boom"}},
	}, details.Annotations)
	assert.Equal(t, updatedAt, details.updatedAt)
	assert.False(t, details.stale)
	assert.Equal(t, []string{
		"/repos/owner/repo/actions/runs/7/jobs",
		"/repos/owner/repo/check-runs/2/annotations",
		"/repos/owner/repo/check-runs/3/annotations",
	}, requestedPaths())
}

func TestLoadRunDetailsWhenJobsFailToLoad(t *testing.T) {
	client, requestedPaths := newFakeGithubClient(t, map[string]string{})

	details := loadRunDetails(client, &models.GithubRun{RunId: 7})

	assert.EqualError(t, details.Err, "GitHub API request failed: HTTP 404: Not Found")
	assert.Empty(t, details.Jobs)
	assert.Empty(t, details.Annotations)
	assert.Equal(t, []string{"/repos/owner/repo/actions/runs/7/jobs"}, requestedPaths())
}

func TestLoadJobLog(t *testing.T) {
	log := "2026-10-21T00:00:01.5Z setting up\n" +
		"2026-10-21T00:00:03.5Z ##[group]Run make\n" +
		"2026-10-21T00:00:03.6Z ##[error]boom\n"
	client, requestedPaths := newFakeGithubClient(t, map[string]string{
		"/repos/owner/repo/actions/jobs/5/logs":      log,
		"/repos/owner/repo/check-runs/5/annotations": `[{"annotation_level": "failure", "message": "boom"}]`,
	})
	steps := []models.GithubStep{
		githubStep(1, "Set up job", "2026-10-21T00:00:01Z", "2026-10-21T00:00:02Z", "success"),
		githubStep(2, "Build", "2026-10-21T00:00:03Z", "2026-10-21T00:00:04Z", "failure"),
	}

	jobLog := loadJobLog(client, &models.GithubJob{JobId: 5, Status: "completed", Conclusion: "failure", Steps: steps})

	assert.Equal(t, &GithubJobLog{
		Log: &github_commands.JobLog{
			Lines: []string{"setting up", "##[group]Run make", "##[error]boom"},
			Steps: []github_commands.StepLog{
				{StepNumber: 1, Lines: []string{"setting up"}},
				{StepNumber: 2, Lines: []string{"##[group]Run make", "##[error]boom"}},
			},
		},
		Annotations: []*models.GithubAnnotation{{Level: "failure", Message: "boom"}},
	}, jobLog)
	assert.Equal(t, []string{
		"/repos/owner/repo/actions/jobs/5/logs",
		"/repos/owner/repo/check-runs/5/annotations",
	}, requestedPaths())
}

func TestLoadJobLogOfSuccessfulJobSkipsAnnotations(t *testing.T) {
	client, requestedPaths := newFakeGithubClient(t, map[string]string{
		"/repos/owner/repo/actions/jobs/5/logs": "2026-10-21T00:00:01.5Z ok\n",
	})

	jobLog := loadJobLog(client, &models.GithubJob{JobId: 5, Status: "completed", Conclusion: "success"})

	assert.NoError(t, jobLog.Err)
	assert.Equal(t, []string{"ok"}, jobLog.Log.Lines)
	assert.Nil(t, jobLog.Annotations)
	assert.Equal(t, []string{"/repos/owner/repo/actions/jobs/5/logs"}, requestedPaths())
}

func TestLoadJobLogKeepsLogWhenAnnotationsFail(t *testing.T) {
	client, _ := newFakeGithubClient(t, map[string]string{
		"/repos/owner/repo/actions/jobs/5/logs": "2026-10-21T00:00:01.5Z boom\n",
	})

	jobLog := loadJobLog(client, &models.GithubJob{JobId: 5, Status: "completed", Conclusion: "failure"})

	assert.NoError(t, jobLog.Err)
	assert.Equal(t, []string{"boom"}, jobLog.Log.Lines)
	assert.Empty(t, jobLog.Annotations)
}

func TestLoadJobLogWhenLogFailsToLoad(t *testing.T) {
	client, requestedPaths := newFakeGithubClient(t, map[string]string{})

	jobLog := loadJobLog(client, &models.GithubJob{JobId: 5, Status: "completed", Conclusion: "failure"})

	assert.Equal(t, &GithubJobLog{Err: &github_commands.APIError{StatusCode: http.StatusNotFound, Message: "Not Found"}}, jobLog)
	assert.Equal(t, []string{"/repos/owner/repo/actions/jobs/5/logs"}, requestedPaths())
}

func jobNames(jobs []*models.GithubJob) []string {
	names := make([]string, 0, len(jobs))
	for _, job := range jobs {
		names = append(names, job.Name)
	}
	return names
}

func githubStep(number int, name string, started string, completed string, conclusion string) models.GithubStep {
	parse := func(s string) time.Time {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			panic(err)
		}
		return t
	}
	return models.GithubStep{
		Number:      number,
		Name:        name,
		Status:      "completed",
		Conclusion:  conclusion,
		StartedAt:   parse(started),
		CompletedAt: parse(completed),
	}
}
