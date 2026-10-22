package github_commands

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRestAPIBaseURL(t *testing.T) {
	assert.Equal(t, "https://api.github.com", RestAPIBaseURL("github.com"))
	assert.Equal(t, "https://api.github.com", RestAPIBaseURL("GitHub.com"))
	assert.Equal(t, "https://api.acme.ghe.com", RestAPIBaseURL("acme.ghe.com"))
	assert.Equal(t, "https://git.acme.com/api/v3", RestAPIBaseURL("git.acme.com"))
}

type recordedRequest struct {
	method string
	path   string
	query  string
	auth   string
	body   string
}

// newTestClient returns a client talking to a fake server that answers every
// request with the given status and body, and records the requests it got.
func newTestClient(t *testing.T, status int, response string) (*Client, *[]recordedRequest) {
	t.Helper()

	requests := []recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, recordedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.RawQuery,
			auth:   r.Header.Get("Authorization"),
			body:   string(body),
		})
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)

	repo := Repo{Host: "github.com", Owner: "owner", Name: "repo", Token: "secret"}
	return NewClientWithBaseURL(repo, server.URL), &requests
}

func TestListRuns(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{
		"total_count": 1,
		"workflow_runs": [{"id": 42, "name": "CI", "head_branch": "main", "status": "completed", "conclusion": "success"}]
	}`)

	runs, err := client.ListRuns(context.Background(), ListRunsOptions{Branch: "main", Limit: 30})
	assert.NoError(t, err)

	assert.Len(t, runs, 1)
	assert.Equal(t, int64(42), runs[0].RunId)
	assert.Equal(t, "CI", runs[0].Name)

	assert.Equal(t, []recordedRequest{{
		method: "GET",
		path:   "/repos/owner/repo/actions/runs",
		query:  "branch=main&per_page=30",
		auth:   "Bearer secret",
	}}, *requests)
}

func TestListRunsForWorkflow(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"workflow_runs": []}`)

	_, err := client.ListRuns(context.Background(), ListRunsOptions{WorkflowId: 7, Status: "failure"})
	assert.NoError(t, err)

	assert.Equal(t, "/repos/owner/repo/actions/workflows/7/runs", (*requests)[0].path)
	assert.Equal(t, "per_page=100&status=failure", (*requests)[0].query)
}

func TestListJobsFollowsPages(t *testing.T) {
	pages := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pages++
		jobs := []map[string]any{}
		if r.URL.Query().Get("page") == "1" {
			for i := range 100 {
				jobs = append(jobs, map[string]any{"id": i})
			}
		} else {
			jobs = append(jobs, map[string]any{"id": 100})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"total_count": 101, "jobs": jobs})
	}))
	t.Cleanup(server.Close)

	client := NewClientWithBaseURL(Repo{Owner: "owner", Name: "repo"}, server.URL)
	jobs, err := client.ListJobs(context.Background(), 1)
	assert.NoError(t, err)

	assert.Len(t, jobs, 101)
	assert.Equal(t, 2, pages)
}

func TestRerunWithDebugLogging(t *testing.T) {
	client, requests := newTestClient(t, http.StatusCreated, ``)

	assert.NoError(t, client.RerunFailedJobs(context.Background(), 5, true))
	assert.NoError(t, client.RerunRun(context.Background(), 5, false))

	assert.Equal(t, "/repos/owner/repo/actions/runs/5/rerun-failed-jobs", (*requests)[0].path)
	assert.JSONEq(t, `{"enable_debug_logging": true}`, (*requests)[0].body)
	assert.Equal(t, "/repos/owner/repo/actions/runs/5/rerun", (*requests)[1].path)
	assert.Empty(t, (*requests)[1].body)
}

func TestDispatchWorkflow(t *testing.T) {
	client, requests := newTestClient(t, http.StatusNoContent, ``)

	err := client.DispatchWorkflow(context.Background(), 9, "main", map[string]string{"env": "prod"})
	assert.NoError(t, err)

	assert.Equal(t, "POST", (*requests)[0].method)
	assert.Equal(t, "/repos/owner/repo/actions/workflows/9/dispatches", (*requests)[0].path)
	assert.JSONEq(t, `{"ref": "main", "inputs": {"env": "prod"}}`, (*requests)[0].body)
}

func TestGetFileContentEscapesPath(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `name: CI`)

	content, err := client.GetFileContent(context.Background(), ".github/workflows/my ci.yml", "main")
	assert.NoError(t, err)

	assert.Equal(t, "name: CI", content)
	assert.Equal(t, "/repos/owner/repo/contents/.github/workflows/my ci.yml", (*requests)[0].path)
	assert.Equal(t, "ref=main", (*requests)[0].query)
}

func TestAPIError(t *testing.T) {
	client, _ := newTestClient(t, http.StatusNotFound, `{"message": "Not Found"}`)

	_, err := client.GetRun(context.Background(), 1)

	assert.EqualError(t, err, "GitHub API request failed: HTTP 404: Not Found")
	assert.True(t, IsNotFound(err))
}

func TestLogRedirectDropsAuthorization(t *testing.T) {
	var blobAuth string
	blobServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blobAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("the log"))
	}))
	t.Cleanup(blobServer.Close)

	// Serve the API from a different host name than the blob storage, as
	// GitHub does; 127.0.0.1 and localhost count as different hosts.
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, blobServer.URL+"/blob", http.StatusFound)
	}))
	t.Cleanup(apiServer.Close)
	apiURL := "http://localhost:" + strconv.Itoa(apiServer.Listener.Addr().(*net.TCPAddr).Port)

	client := NewClientWithBaseURL(Repo{Owner: "owner", Name: "repo", Token: "secret"}, apiURL)
	log, err := client.GetJobLog(context.Background(), 3)
	assert.NoError(t, err)

	assert.Equal(t, "the log", log)
	assert.Empty(t, blobAuth)
}
