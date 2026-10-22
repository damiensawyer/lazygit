package components

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// FakeGithub stands in for the GitHub API in tests of the GitHub Actions view
// and of the pull requests panel. While a test that has one runs, it is served over HTTP, and lazygit
// talks to it instead of to GitHub, without needing a token.
//
// It serves the canned data below for any repository, and records the
// requests it gets. Requests that would change something on GitHub, like
// re-running or deleting a run, succeed but change nothing, so that what the
// view shows afterwards only depends on what the test set up. A test can
// change the data while it runs with TestDriver.FakeGithub().Update.
type FakeGithub struct {
	Runs []*models.GithubRun
	// keyed by run ID
	Jobs map[int64][]*models.GithubJob
	// keyed by job ID
	JobLogs map[int64]string
	// keyed by job ID
	Annotations map[int64][]*models.GithubAnnotation
	Workflows   []*models.GithubWorkflow
	// workflow definitions, keyed by their path in the repository, e.g.
	// ".github/workflows/ci.yml"
	WorkflowFiles map[string]string
	// keyed by run ID
	Artifacts map[int64][]*models.GithubArtifact
	// the content of each artifact's archive: keyed by artifact ID, and then
	// by the path of a file in the archive
	ArtifactFiles map[int64]map[string]string
	// the repository's pull requests, as the GraphQL API lists them
	PullRequests []*models.GithubPullRequest
	// keyed by pull request number
	PullRequestDetails map[int]*models.GithubPullRequestDetails
	// the repository's labels
	Labels []models.GithubLabel
	// the logins of the users that pull requests can be assigned to
	AssignableUsers []string
	// responses to give instead of the canned data, keyed by method and path,
	// e.g. "GET /repos/owner/repo/actions/runs"
	Errors map[string]FakeGithubError

	// guards the fields above while the fake is being served
	dataMutex sync.Mutex

	mutex    sync.Mutex
	server   *httptest.Server
	requests []string
}

// FakeGithubError is an error response from the fake GitHub API.
type FakeGithubError struct {
	StatusCode int
	Message    string
}

func (self *FakeGithub) start() {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.server = httptest.NewServer(self.handler())
}

func (self *FakeGithub) stop() {
	self.mutex.Lock()
	server := self.server
	self.mutex.Unlock()
	server.Close()
}

// URL returns the base URL the fake API is served at, or "" if it isn't
// being served.
func (self *FakeGithub) URL() string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.server == nil {
		return ""
	}
	return self.server.URL
}

// Requests returns the requests the fake API has received so far, in the
// order they came in. Each is described by its method and path, followed by
// its query (sorted by key) and its body, if any, e.g.
// "POST /repos/owner/repo/actions/runs/1/rerun".
func (self *FakeGithub) Requests() []string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return append([]string{}, self.requests...)
}

func (self *FakeGithub) record(r *http.Request, body []byte) {
	description := r.Method + " " + r.URL.Path
	if len(r.URL.Query()) > 0 {
		description += "?" + r.URL.Query().Encode()
	}
	if r.URL.Path == "/graphql" {
		description = describeGraphQLRequest(body)
	} else if len(body) > 0 {
		description += " " + string(body)
	}

	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.requests = append(self.requests, description)
}

func (self *FakeGithub) handler() http.Handler {
	mux := http.NewServeMux()
	repo := "/repos/{owner}/{repo}/"

	mux.HandleFunc("GET "+repo+"actions/runs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"workflow_runs": self.filterRuns(r, 0)})
	})
	mux.HandleFunc("GET "+repo+"actions/workflows/{id}/runs", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"workflow_runs": self.filterRuns(r, pathID(r))})
	})
	mux.HandleFunc("GET "+repo+"actions/runs/{id}", func(w http.ResponseWriter, r *http.Request) {
		run, found := lo.Find(self.Runs, func(run *models.GithubRun) bool { return run.RunId == pathID(r) })
		if !found {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		writeJSON(w, run)
	})
	mux.HandleFunc("GET "+repo+"actions/runs/{id}/jobs", func(w http.ResponseWriter, r *http.Request) {
		jobs := lo.CoalesceSliceOrEmpty(self.Jobs[pathID(r)])
		writeJSON(w, map[string]any{"total_count": len(jobs), "jobs": jobs})
	})
	mux.HandleFunc("GET "+repo+"actions/jobs/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		log, found := self.JobLogs[pathID(r)]
		if !found {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		_, _ = io.WriteString(w, log)
	})
	mux.HandleFunc("GET "+repo+"check-runs/{id}/annotations", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, lo.CoalesceSliceOrEmpty(self.Annotations[pathID(r)]))
	})
	mux.HandleFunc("GET "+repo+"actions/workflows", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"workflows": lo.CoalesceSliceOrEmpty(self.Workflows)})
	})
	mux.HandleFunc("GET "+repo+"actions/runs/{id}/artifacts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"artifacts": lo.CoalesceSliceOrEmpty(self.Artifacts[pathID(r)])})
	})
	mux.HandleFunc("GET "+repo+"actions/artifacts/{id}/zip", func(w http.ResponseWriter, r *http.Request) {
		files, found := self.ArtifactFiles[pathID(r)]
		if !found {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		archive, err := zipFiles(files)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_, _ = w.Write(archive)
	})
	mux.HandleFunc("GET "+repo+"contents/{path...}", func(w http.ResponseWriter, r *http.Request) {
		content, found := self.WorkflowFiles[r.PathValue("path")]
		if !found {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		_, _ = io.WriteString(w, content)
	})

	accept := func(status int) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }
	}
	mux.HandleFunc("POST "+repo+"actions/runs/{id}/rerun", accept(http.StatusCreated))
	mux.HandleFunc("POST "+repo+"actions/runs/{id}/rerun-failed-jobs", accept(http.StatusCreated))
	mux.HandleFunc("POST "+repo+"actions/jobs/{id}/rerun", accept(http.StatusCreated))
	mux.HandleFunc("POST "+repo+"actions/runs/{id}/cancel", accept(http.StatusAccepted))
	mux.HandleFunc("POST "+repo+"actions/runs/{id}/force-cancel", accept(http.StatusAccepted))
	mux.HandleFunc("DELETE "+repo+"actions/runs/{id}", accept(http.StatusNoContent))
	mux.HandleFunc("POST "+repo+"actions/workflows/{id}/dispatches", accept(http.StatusNoContent))
	mux.HandleFunc("PUT "+repo+"actions/workflows/{id}/enable", accept(http.StatusNoContent))
	mux.HandleFunc("PUT "+repo+"actions/workflows/{id}/disable", accept(http.StatusNoContent))
	mux.HandleFunc("DELETE "+repo+"git/refs/heads/{branch...}", accept(http.StatusNoContent))
	mux.HandleFunc("GET /repos/{owner}/{repo}", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"default_branch": "master"})
	})
	mux.HandleFunc("POST "+repo+"pulls", self.createPullRequest)
	mux.HandleFunc("PATCH "+repo+"pulls/{number}", accept(http.StatusOK))
	mux.HandleFunc("GET "+repo+"labels", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, lo.CoalesceSliceOrEmpty(self.Labels))
	})
	mux.HandleFunc("POST "+repo+"issues/{number}/labels", accept(http.StatusOK))
	mux.HandleFunc("DELETE "+repo+"issues/{number}/labels/{label}", accept(http.StatusOK))
	mux.HandleFunc("GET "+repo+"assignees", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, lo.Map(self.AssignableUsers, func(login string, _ int) models.GithubUser {
			return models.GithubUser{Login: login}
		}))
	})
	mux.HandleFunc("POST "+repo+"issues/{number}/assignees", accept(http.StatusCreated))
	mux.HandleFunc("DELETE "+repo+"issues/{number}/assignees", accept(http.StatusOK))
	mux.HandleFunc("POST "+repo+"pulls/{number}/requested_reviewers", accept(http.StatusCreated))
	mux.HandleFunc("DELETE "+repo+"pulls/{number}/requested_reviewers", accept(http.StatusOK))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		self.record(r, body)
		// for the handlers that read it too
		r.Body = io.NopCloser(bytes.NewReader(body))

		self.dataMutex.Lock()
		defer self.dataMutex.Unlock()
		if response, found := self.Errors[r.Method+" "+r.URL.Path]; found {
			writeError(w, response.StatusCode, response.Message)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/graphql" {
			self.serveGraphQL(w, body)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

// filterRuns returns the runs that match the request's query, the way the
// API filters them, and those of the given workflow if it isn't 0.
func (self *FakeGithub) filterRuns(r *http.Request, workflowId int64) []*models.GithubRun {
	query := r.URL.Query()
	runs := lo.Filter(self.Runs, func(run *models.GithubRun, _ int) bool {
		matches := func(param string, value string) bool {
			return query.Get(param) == "" || query.Get(param) == value
		}
		status := query.Get("status")
		return (workflowId == 0 || run.WorkflowId == workflowId) &&
			matches("branch", run.HeadBranch) &&
			matches("event", run.Event) &&
			matches("head_sha", run.HeadSha) &&
			(status == "" || status == run.Status || status == run.Conclusion)
	})
	if limit, err := strconv.Atoi(query.Get("per_page")); err == nil && limit < len(runs) {
		runs = runs[:limit]
	}
	return lo.CoalesceSliceOrEmpty(runs)
}

func pathID(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func zipFiles(files map[string]string) ([]byte, error) {
	paths := lo.Keys(files)
	sort.Strings(paths)

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, path := range paths {
		file, err := writer.Create(path)
		if err != nil {
			return nil, err
		}
		if _, err := io.WriteString(file, files[path]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// FakeGithubDriver makes assertions about the requests a test's FakeGithub
// has received.
type FakeGithubDriver struct {
	t    *TestDriver
	fake *FakeGithub
}

// ChangeRequests asserts which requests that would change something on
// GitHub (anything but a GET) the fake API has received, in order.
func (self *FakeGithubDriver) ChangeRequests(matchers ...*TextMatcher) *FakeGithubDriver {
	self.t.assertWithRetries(func() (bool, string) {
		requests := lo.Reject(self.fake.Requests(), func(request string, _ int) bool {
			return strings.HasPrefix(request, http.MethodGet+" ")
		})
		if len(requests) != len(matchers) {
			return false, fmt.Sprintf("Expected %d change requests to the fake GitHub API, but got %d:\n%s",
				len(matchers), len(requests), strings.Join(requests, "\n"))
		}
		for i, matcher := range matchers {
			if ok, message := matcher.context(fmt.Sprintf("change request %d", i)).test(requests[i]); !ok {
				return false, message
			}
		}
		return true, ""
	})
	return self
}

// Update changes the data the fake API serves, e.g. to add a run that has
// started since the test began.
func (self *FakeGithubDriver) Update(change func(fake *FakeGithub)) *FakeGithubDriver {
	self.fake.dataMutex.Lock()
	defer self.fake.dataMutex.Unlock()
	change(self.fake)
	return self
}

// ReceivedRequest asserts that the fake API has received a request, of any
// kind, matching the matcher.
func (self *FakeGithubDriver) ReceivedRequest(matcher *TextMatcher) *FakeGithubDriver {
	self.t.assertWithRetries(func() (bool, string) {
		requests := self.fake.Requests()
		if lo.SomeBy(requests, func(request string) bool {
			ok, _ := matcher.test(request)
			return ok
		}) {
			return true, ""
		}
		return false, fmt.Sprintf("Expected the fake GitHub API to have received a request that %s, but it got:\n%s",
			matcher.name(), strings.Join(requests, "\n"))
	})
	return self
}
