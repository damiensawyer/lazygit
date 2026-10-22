package github_commands

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

// capturedRequest is everything about a request that the endpoint wrappers
// are responsible for getting right.
type capturedRequest struct {
	method      string
	path        string
	rawPath     string
	query       string
	body        string
	accept      string
	apiVersion  string
	contentType string
	auth        string
}

// newCapturingClient returns a client talking to a fake server that answers
// every request with the given status and body, and records each request in
// full.
func newCapturingClient(t *testing.T, token string, status int, response string) (*Client, *[]capturedRequest) {
	t.Helper()

	requests := []capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, capturedRequest{
			method:      r.Method,
			path:        r.URL.Path,
			rawPath:     r.URL.EscapedPath(),
			query:       r.URL.RawQuery,
			body:        string(body),
			accept:      r.Header.Get("Accept"),
			apiVersion:  r.Header.Get("X-GitHub-Api-Version"),
			contentType: r.Header.Get("Content-Type"),
			auth:        r.Header.Get("Authorization"),
		})
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	repo := Repo{Host: "github.com", Owner: "owner", Name: "repo", Token: token}
	return NewClientWithBaseURL(repo, server.URL), &requests
}

func TestEndpointRequests(t *testing.T) {
	const jsonAccept = "application/vnd.github+json"

	scenarios := []struct {
		name     string
		status   int
		response string
		call     func(*Client) error
		expected capturedRequest
	}{
		{
			name:     "list runs without filters",
			response: `{"workflow_runs": []}`,
			call: func(c *Client) error {
				_, err := c.ListRuns(context.Background(), ListRunsOptions{})
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/runs", query: "per_page=100"},
		},
		{
			name:     "list runs with every filter",
			response: `{"workflow_runs": []}`,
			call: func(c *Client) error {
				_, err := c.ListRuns(context.Background(), ListRunsOptions{
					Branch: "feature/x", WorkflowId: 3, Status: "in_progress", Event: "pull_request", HeadSha: "abc123", Limit: 5,
				})
				return err
			},
			expected: capturedRequest{
				method: "GET", path: "/repos/owner/repo/actions/workflows/3/runs",
				query: "branch=feature%2Fx&event=pull_request&head_sha=abc123&per_page=5&status=in_progress",
			},
		},
		{
			name:     "get run",
			response: `{"id": 17}`,
			call: func(c *Client) error {
				_, err := c.GetRun(context.Background(), 17)
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/runs/17"},
		},
		{
			name:     "list jobs",
			response: `{"total_count": 0, "jobs": []}`,
			call: func(c *Client) error {
				_, err := c.ListJobs(context.Background(), 17)
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/runs/17/jobs", query: "page=1&per_page=100"},
		},
		{
			name:     "get job log",
			response: `log`,
			call: func(c *Client) error {
				_, err := c.GetJobLog(context.Background(), 23)
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/jobs/23/logs"},
		},
		{
			name:     "list annotations",
			response: `[]`,
			call: func(c *Client) error {
				_, err := c.ListAnnotations(context.Background(), 23)
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/check-runs/23/annotations", query: "per_page=100"},
		},
		{
			name:     "list workflows",
			response: `{"workflows": []}`,
			call: func(c *Client) error {
				_, err := c.ListWorkflows(context.Background())
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/workflows", query: "per_page=100"},
		},
		{
			name:     "list artifacts",
			response: `{"artifacts": []}`,
			call: func(c *Client) error {
				_, err := c.ListArtifacts(context.Background(), 17)
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/runs/17/artifacts", query: "per_page=100"},
		},
		{
			name:     "download artifact",
			response: `zip`,
			call: func(c *Client) error {
				_, err := c.DownloadArtifact(context.Background(), 31)
				return err
			},
			expected: capturedRequest{method: "GET", path: "/repos/owner/repo/actions/artifacts/31/zip"},
		},
		{
			name:   "rerun run with debug logging",
			status: http.StatusCreated,
			call:   func(c *Client) error { return c.RerunRun(context.Background(), 17, true) },
			expected: capturedRequest{
				method: "POST", path: "/repos/owner/repo/actions/runs/17/rerun",
				body: `{"enable_debug_logging":true}`, contentType: "application/json",
			},
		},
		{
			name:     "rerun failed jobs",
			status:   http.StatusCreated,
			call:     func(c *Client) error { return c.RerunFailedJobs(context.Background(), 17, false) },
			expected: capturedRequest{method: "POST", path: "/repos/owner/repo/actions/runs/17/rerun-failed-jobs"},
		},
		{
			name:     "rerun job",
			status:   http.StatusCreated,
			call:     func(c *Client) error { return c.RerunJob(context.Background(), 23, false) },
			expected: capturedRequest{method: "POST", path: "/repos/owner/repo/actions/jobs/23/rerun"},
		},
		{
			name:   "rerun job with debug logging",
			status: http.StatusCreated,
			call:   func(c *Client) error { return c.RerunJob(context.Background(), 23, true) },
			expected: capturedRequest{
				method: "POST", path: "/repos/owner/repo/actions/jobs/23/rerun",
				body: `{"enable_debug_logging":true}`, contentType: "application/json",
			},
		},
		{
			name:     "cancel run",
			status:   http.StatusAccepted,
			call:     func(c *Client) error { return c.CancelRun(context.Background(), 17) },
			expected: capturedRequest{method: "POST", path: "/repos/owner/repo/actions/runs/17/cancel"},
		},
		{
			name:     "force cancel run",
			status:   http.StatusAccepted,
			call:     func(c *Client) error { return c.ForceCancelRun(context.Background(), 17) },
			expected: capturedRequest{method: "POST", path: "/repos/owner/repo/actions/runs/17/force-cancel"},
		},
		{
			name:     "delete run",
			status:   http.StatusNoContent,
			call:     func(c *Client) error { return c.DeleteRun(context.Background(), 17) },
			expected: capturedRequest{method: "DELETE", path: "/repos/owner/repo/actions/runs/17"},
		},
		{
			name:   "dispatch workflow without inputs",
			status: http.StatusNoContent,
			call:   func(c *Client) error { return c.DispatchWorkflow(context.Background(), 3, "main", nil) },
			expected: capturedRequest{
				method: "POST", path: "/repos/owner/repo/actions/workflows/3/dispatches",
				body: `{"ref":"main"}`, contentType: "application/json",
			},
		},
		{
			name:     "enable workflow",
			status:   http.StatusNoContent,
			call:     func(c *Client) error { return c.EnableWorkflow(context.Background(), 3) },
			expected: capturedRequest{method: "PUT", path: "/repos/owner/repo/actions/workflows/3/enable"},
		},
		{
			name:     "disable workflow",
			status:   http.StatusNoContent,
			call:     func(c *Client) error { return c.DisableWorkflow(context.Background(), 3) },
			expected: capturedRequest{method: "PUT", path: "/repos/owner/repo/actions/workflows/3/disable"},
		},
		{
			name:     "get file content without ref",
			response: `name: CI`,
			call: func(c *Client) error {
				_, err := c.GetFileContent(context.Background(), ".github/workflows/ci.yml", "")
				return err
			},
			expected: capturedRequest{
				method: "GET", path: "/repos/owner/repo/contents/.github/workflows/ci.yml",
				accept: "application/vnd.github.raw+json",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			status := s.status
			if status == 0 {
				status = http.StatusOK
			}
			client, requests := newCapturingClient(t, "secret", status, s.response)

			assert.NoError(t, s.call(client))

			expected := s.expected
			expected.rawPath = expected.path
			if expected.accept == "" {
				expected.accept = jsonAccept
			}
			expected.apiVersion = "2022-11-28"
			expected.auth = "Bearer secret"
			assert.Equal(t, []capturedRequest{expected}, *requests)
		})
	}
}

func TestRequestWithoutTokenSendsNoAuthorization(t *testing.T) {
	client, requests := newCapturingClient(t, "", http.StatusOK, `{"workflows": []}`)

	_, err := client.ListWorkflows(context.Background())
	assert.NoError(t, err)

	assert.Len(t, *requests, 1)
	assert.Empty(t, (*requests)[0].auth)
	assert.Equal(t, "application/vnd.github+json", (*requests)[0].accept)
}

func TestRequestPathsAreEscaped(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.EscapedPath()))
	}))
	t.Cleanup(server.Close)

	client := NewClientWithBaseURL(Repo{Owner: "my org", Name: "re#po"}, server.URL)
	content, err := client.GetFileContent(context.Background(), "dir/a b#c?.yml", "")
	assert.NoError(t, err)

	assert.Equal(t, "/repos/my%20org/re%23po/contents/dir/a%20b%23c%3F.yml", content)
}

func TestBaseURLTrailingSlashIsIgnored(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"id": 1}`))
	}))
	t.Cleanup(server.Close)

	client := NewClientWithBaseURL(Repo{Owner: "owner", Name: "repo"}, server.URL+"/api/v3/")
	_, err := client.GetRun(context.Background(), 1)
	assert.NoError(t, err)

	assert.Equal(t, "/api/v3/repos/owner/repo/actions/runs/1", path)
}

func TestListRunsLimit(t *testing.T) {
	scenarios := []struct {
		limit    int
		expected string
	}{
		{limit: -1, expected: "per_page=100"},
		{limit: 0, expected: "per_page=100"},
		{limit: 1, expected: "per_page=1"},
		{limit: 100, expected: "per_page=100"},
		{limit: 101, expected: "per_page=100"},
	}

	for _, s := range scenarios {
		t.Run(itoa(int64(s.limit)), func(t *testing.T) {
			client, requests := newCapturingClient(t, "", http.StatusOK, `{"workflow_runs": []}`)

			_, err := client.ListRuns(context.Background(), ListRunsOptions{Limit: s.limit})
			assert.NoError(t, err)

			assert.Equal(t, s.expected, (*requests)[0].query)
		})
	}
}

func TestResponseDecoding(t *testing.T) {
	client, _ := newCapturingClient(t, "", http.StatusOK, `{
		"total_count": 1,
		"workflow_runs": [{
			"id": 42,
			"name": "CI",
			"display_title": "Fix it",
			"run_number": 7,
			"run_attempt": 2,
			"event": "push",
			"status": "completed",
			"conclusion": "failure",
			"workflow_id": 3,
			"head_branch": "main",
			"head_sha": "abc",
			"path": ".github/workflows/ci.yml",
			"html_url": "https://github.com/owner/repo/actions/runs/42",
			"created_at": "2026-10-21T00:00:00Z",
			"updated_at": "2026-10-21T00:05:00Z",
			"run_started_at": "2026-10-21T00:01:00Z",
			"actor": {"login": "octocat"},
			"head_commit": {"message": "Fix it\n\nbody"},
			"unknown_field": true
		}]
	}`)

	runs, err := client.ListRuns(context.Background(), ListRunsOptions{})
	assert.NoError(t, err)

	expected := &models.GithubRun{
		RunId:        42,
		Name:         "CI",
		DisplayTitle: "Fix it",
		Number:       7,
		Attempt:      2,
		Event:        "push",
		Status:       "completed",
		Conclusion:   "failure",
		WorkflowId:   3,
		HeadBranch:   "main",
		HeadSha:      "abc",
		Path:         ".github/workflows/ci.yml",
		Url:          "https://github.com/owner/repo/actions/runs/42",
		CreatedAt:    time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 10, 21, 0, 5, 0, 0, time.UTC),
		RunStartedAt: time.Date(2026, 10, 21, 0, 1, 0, 0, time.UTC),
		Actor:        models.GithubUser{Login: "octocat"},
	}
	expected.HeadCommit.Message = "Fix it\n\nbody"
	assert.Equal(t, []*models.GithubRun{expected}, runs)
}

func TestListJobsDecodesSteps(t *testing.T) {
	client, _ := newCapturingClient(t, "", http.StatusOK, `{
		"total_count": 1,
		"jobs": [{
			"id": 5, "run_id": 42, "name": "test", "status": "in_progress", "conclusion": null,
			"started_at": "2026-10-21T00:01:00Z", "completed_at": null,
			"html_url": "https://github.com/owner/repo/actions/runs/42/job/5",
			"runner_name": "GitHub Actions 2", "labels": ["ubuntu-latest"],
			"steps": [{"number": 1, "name": "Set up job", "status": "completed", "conclusion": "success",
				"started_at": "2026-10-21T00:01:00Z", "completed_at": "2026-10-21T00:01:02Z"}]
		}]
	}`)

	jobs, err := client.ListJobs(context.Background(), 42)
	assert.NoError(t, err)

	assert.Equal(t, []*models.GithubJob{{
		JobId:      5,
		RunId:      42,
		Name:       "test",
		Status:     "in_progress",
		StartedAt:  time.Date(2026, 10, 21, 0, 1, 0, 0, time.UTC),
		Url:        "https://github.com/owner/repo/actions/runs/42/job/5",
		RunnerName: "GitHub Actions 2",
		Labels:     []string{"ubuntu-latest"},
		Steps: []models.GithubStep{{
			Number: 1, Name: "Set up job", Status: "completed", Conclusion: "success",
			StartedAt:   time.Date(2026, 10, 21, 0, 1, 0, 0, time.UTC),
			CompletedAt: time.Date(2026, 10, 21, 0, 1, 2, 0, time.UTC),
		}},
	}}, jobs)
}

func TestListWorkflowsArtifactsAndAnnotationsDecoding(t *testing.T) {
	client, _ := newCapturingClient(t, "", http.StatusOK, `{"workflows": [
		{"id": 3, "name": "CI", "path": ".github/workflows/ci.yml", "state": "disabled_manually", "html_url": "u"}
	]}`)
	workflows, err := client.ListWorkflows(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubWorkflow{
		{WorkflowId: 3, Name: "CI", Path: ".github/workflows/ci.yml", State: "disabled_manually", Url: "u"},
	}, workflows)

	client, _ = newCapturingClient(t, "", http.StatusOK, `{"total_count": 1, "artifacts": [
		{"id": 31, "name": "coverage", "size_in_bytes": 2048, "expired": true}
	]}`)
	artifacts, err := client.ListArtifacts(context.Background(), 42)
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubArtifact{
		{ArtifactId: 31, Name: "coverage", SizeInBytes: 2048, Expired: true},
	}, artifacts)

	client, _ = newCapturingClient(t, "", http.StatusOK, `[
		{"path": "main.go", "start_line": 12, "annotation_level": "warning", "title": "lint", "message": "unused"}
	]`)
	annotations, err := client.ListAnnotations(context.Background(), 5)
	assert.NoError(t, err)
	assert.Equal(t, []*models.GithubAnnotation{
		{Path: "main.go", StartLine: 12, Level: "warning", Title: "lint", Message: "unused"},
	}, annotations)
}

func TestRawResponsesAreReturnedVerbatim(t *testing.T) {
	content := byteOrderMark + "2026-10-21T00:00:00.0Z héllo\r\n\x00binary"
	client, _ := newCapturingClient(t, "", http.StatusOK, content)

	log, err := client.GetJobLog(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, content, log)

	archive, err := client.DownloadArtifact(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, []byte(content), archive)
}

func TestErrorResponses(t *testing.T) {
	scenarios := []struct {
		name          string
		status        int
		response      string
		expectedError string
		notFound      bool
	}{
		{
			name:          "unauthorized",
			status:        http.StatusUnauthorized,
			response:      `{"message": "Bad credentials", "documentation_url": "https://docs.github.com/rest"}`,
			expectedError: "GitHub API request failed: HTTP 401: Bad credentials",
		},
		{
			name:          "forbidden",
			status:        http.StatusForbidden,
			response:      `{"message": "Resource not accessible by integration"}`,
			expectedError: "GitHub API request failed: HTTP 403: Resource not accessible by integration",
		},
		{
			name:          "primary rate limit",
			status:        http.StatusForbidden,
			response:      `{"message": "API rate limit exceeded for user ID 1."}`,
			expectedError: "GitHub API request failed: HTTP 403: API rate limit exceeded for user ID 1.",
		},
		{
			name:          "secondary rate limit",
			status:        http.StatusTooManyRequests,
			response:      `{"message": "You have exceeded a secondary rate limit."}`,
			expectedError: "GitHub API request failed: HTTP 429: You have exceeded a secondary rate limit.",
		},
		{
			name:          "not found",
			status:        http.StatusNotFound,
			response:      `{"message": "Not Found"}`,
			expectedError: "GitHub API request failed: HTTP 404: Not Found",
			notFound:      true,
		},
		{
			name:          "validation failed",
			status:        http.StatusUnprocessableEntity,
			response:      `{"message": "Unexpected inputs provided: [\"foo\"]", "errors": []}`,
			expectedError: "GitHub API request failed: HTTP 422: Unexpected inputs provided: [\"foo\"]",
		},
		{
			name:          "conflict",
			status:        http.StatusConflict,
			response:      `{"message": "Cannot cancel a workflow run that is completed."}`,
			expectedError: "GitHub API request failed: HTTP 409: Cannot cancel a workflow run that is completed.",
		},
		{
			name:          "server error with non-JSON body",
			status:        http.StatusBadGateway,
			response:      `<html>Bad Gateway</html>`,
			expectedError: "GitHub API request failed: HTTP 502",
		},
		{
			name:          "empty body",
			status:        http.StatusInternalServerError,
			response:      ``,
			expectedError: "GitHub API request failed: HTTP 500",
		},
		{
			name:          "JSON body without a message",
			status:        http.StatusGone,
			response:      `{"error": "gone"}`,
			expectedError: "GitHub API request failed: HTTP 410",
		},
		{
			name:          "redirect that isn't followed",
			status:        http.StatusNotModified,
			response:      ``,
			expectedError: "GitHub API request failed: HTTP 304",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			client, _ := newCapturingClient(t, "secret", s.status, s.response)

			for name, call := range map[string]func() error{
				"get":    func() error { _, err := client.GetRun(context.Background(), 1); return err },
				"raw":    func() error { _, err := client.GetJobLog(context.Background(), 1); return err },
				"post":   func() error { return client.CancelRun(context.Background(), 1) },
				"put":    func() error { return client.EnableWorkflow(context.Background(), 1) },
				"delete": func() error { return client.DeleteRun(context.Background(), 1) },
			} {
				err := call()
				assert.EqualError(t, err, s.expectedError, name)
				assert.Equal(t, s.notFound, IsNotFound(err), name)

				var apiErr *APIError
				assert.ErrorAs(t, err, &apiErr, name)
				assert.Equal(t, s.status, apiErr.StatusCode, name)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	assert.False(t, IsNotFound(nil))
	assert.False(t, IsNotFound(io.EOF))
	assert.False(t, IsNotFound(&APIError{StatusCode: http.StatusForbidden}))
	assert.True(t, IsNotFound(&APIError{StatusCode: http.StatusNotFound}))
	assert.True(t, IsNotFound(errorWrapper{&APIError{StatusCode: http.StatusNotFound}}))
}

type errorWrapper struct{ err error }

func (self errorWrapper) Error() string { return "wrapped: " + self.err.Error() }
func (self errorWrapper) Unwrap() error { return self.err }

func TestMalformedJSONResponses(t *testing.T) {
	scenarios := []struct {
		name     string
		response string
		call     func(*Client) error
	}{
		{
			name:     "list runs",
			response: `{"workflow_runs": [`,
			call: func(c *Client) error {
				_, err := c.ListRuns(context.Background(), ListRunsOptions{})
				return err
			},
		},
		{
			name:     "get run",
			response: `not json`,
			call: func(c *Client) error {
				_, err := c.GetRun(context.Background(), 1)
				return err
			},
		},
		{
			name:     "list jobs",
			response: `{"jobs": "nope"}`,
			call: func(c *Client) error {
				_, err := c.ListJobs(context.Background(), 1)
				return err
			},
		},
		{
			name:     "list annotations",
			response: `{"annotations": []}`,
			call: func(c *Client) error {
				_, err := c.ListAnnotations(context.Background(), 1)
				return err
			},
		},
		{
			name:     "list workflows",
			response: ``,
			call: func(c *Client) error {
				_, err := c.ListWorkflows(context.Background())
				return err
			},
		},
		{
			name:     "list artifacts",
			response: `{"artifacts": [{"id": "thirty-one"}]}`,
			call: func(c *Client) error {
				_, err := c.ListArtifacts(context.Background(), 1)
				return err
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			client, _ := newCapturingClient(t, "", http.StatusOK, s.response)
			assert.Error(t, s.call(client))
		})
	}
}

func TestListJobsStopsAtAnEmptyPage(t *testing.T) {
	// A total_count that overstates the number of jobs mustn't make us ask
	// for pages forever.
	pages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		if r.URL.Query().Get("page") == "1" {
			_, _ = w.Write([]byte(`{"total_count": 500, "jobs": [{"id": 1}, {"id": 2}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"total_count": 500, "jobs": []}`))
	}))
	t.Cleanup(server.Close)

	client := NewClientWithBaseURL(Repo{Owner: "owner", Name: "repo"}, server.URL)
	jobs, err := client.ListJobs(context.Background(), 1)
	assert.NoError(t, err)

	assert.Len(t, jobs, 2)
	assert.Equal(t, 2, pages)
}

func TestListJobsFailsWhenALaterPageFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			_, _ = w.Write([]byte(`{"total_count": 2, "jobs": [{"id": 1}]}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	t.Cleanup(server.Close)

	client := NewClientWithBaseURL(Repo{Owner: "owner", Name: "repo"}, server.URL)
	jobs, err := client.ListJobs(context.Background(), 1)

	assert.EqualError(t, err, "GitHub API request failed: HTTP 403: API rate limit exceeded")
	assert.Nil(t, jobs)
}

func TestRequestHonoursContextCancellation(t *testing.T) {
	client, requests := newCapturingClient(t, "", http.StatusOK, `{}`)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.GetRun(ctx, 1)

	assert.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, *requests)
}

func TestRequestFailsWhenServerIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()

	client := NewClientWithBaseURL(Repo{Owner: "owner", Name: "repo"}, url)
	_, err := client.GetRun(context.Background(), 1)

	assert.Error(t, err)
	var apiErr *APIError
	assert.NotErrorAs(t, err, &apiErr)
}

func TestRepoFullName(t *testing.T) {
	repo := Repo{Host: "github.com", Owner: "jesseduffield", Name: "lazygit"}
	assert.Equal(t, "jesseduffield/lazygit", repo.FullName())
	assert.Equal(t, repo, NewClient(repo).Repo())
}

func TestRestAPIBaseURLForEnterpriseWithPort(t *testing.T) {
	assert.Equal(t, "https://git.acme.com:8443/api/v3", RestAPIBaseURL("git.acme.com:8443"))
}

func TestExtractArtifactEdgeCases(t *testing.T) {
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	_, err := writer.Create("empty-dir/")
	assert.NoError(t, err)
	header := &zip.FileHeader{Name: "bin/tool", Method: zip.Deflate}
	header.SetMode(0o755)
	w, err := writer.CreateHeader(header)
	assert.NoError(t, err)
	_, _ = w.Write([]byte("echo hi"))
	w, err = writer.Create("/absolute.txt")
	assert.NoError(t, err)
	_, _ = w.Write([]byte("contained"))
	w, err = writer.Create("existing.txt")
	assert.NoError(t, err)
	_, _ = w.Write([]byte("new"))
	assert.NoError(t, writer.Close())

	dir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("old content that is longer"), 0o644))

	assert.NoError(t, ExtractArtifact(archive.Bytes(), dir))

	assert.DirExists(t, filepath.Join(dir, "empty-dir"))

	info, err := os.Stat(filepath.Join(dir, "bin", "tool"))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())

	// An absolute path in the archive is taken to be relative to dir.
	content, err := os.ReadFile(filepath.Join(dir, "absolute.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "contained", string(content))

	content, err = os.ReadFile(filepath.Join(dir, "existing.txt"))
	assert.NoError(t, err)
	assert.Equal(t, "new", string(content))
}

func TestExtractArtifactRejectsInvalidArchive(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "artifact")

	assert.Error(t, ExtractArtifact([]byte("not a zip"), dir))
	assert.NoDirExists(t, dir)
}

func TestExtractArtifactRefusesSiblingDirWithSamePrefix(t *testing.T) {
	// "../artifact-evil" starts with the name of dir, but isn't inside it.
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	_, err := writer.Create("../artifact-evil/x.txt")
	assert.NoError(t, err)
	assert.NoError(t, writer.Close())

	parent := t.TempDir()
	err = ExtractArtifact(archive.Bytes(), filepath.Join(parent, "artifact"))

	assert.EqualError(t, err, "artifact contains an invalid path: ../artifact-evil/x.txt")
	assert.NoDirExists(t, filepath.Join(parent, "artifact-evil"))
}
