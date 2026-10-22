// Package github_commands talks to the GitHub REST API about a repository's
// GitHub Actions: its workflows, their runs, and the jobs, logs, and artifacts
// those runs produce.
package github_commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
)

// RestAPIBaseURL returns the REST API root for a GitHub host: github.com and
// tenancy (ghe.com) hosts serve their API from a dedicated api. subdomain,
// while GitHub Enterprise Server instances serve it from /api/v3 on the host
// itself.
func RestAPIBaseURL(host string) string {
	if auth.IsEnterprise(host) {
		return fmt.Sprintf("https://%s/api/v3", host)
	}
	return fmt.Sprintf("https://api.%s", auth.NormalizeHostname(host))
}

// Repo identifies a repository on a GitHub instance, along with the token to
// access it with.
type Repo struct {
	// e.g. "github.com", or "git.acme.com" for a GitHub Enterprise Server
	Host  string
	Owner string
	Name  string
	Token string
}

func (self Repo) FullName() string {
	return self.Owner + "/" + self.Name
}

// Client makes requests against one repository.
type Client struct {
	httpClient *http.Client
	baseURL    string
	repo       Repo
}

func NewClient(repo Repo) *Client {
	return NewClientWithBaseURL(repo, RestAPIBaseURL(repo.Host))
}

// NewClientWithBaseURL is for tests, which point the client at a fake server.
func NewClientWithBaseURL(repo Repo, baseURL string) *Client {
	return &Client{
		// Downloading a large log or artifact can take a while, but a request
		// that hangs indefinitely shouldn't leave a spinner up forever.
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		repo:       repo,
	}
}

func (self *Client) Repo() Repo {
	return self.repo
}

// APIError is a non-2xx response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (self *APIError) Error() string {
	if self.Message == "" {
		return fmt.Sprintf("GitHub API request failed: HTTP %d", self.StatusCode)
	}
	return fmt.Sprintf("GitHub API request failed: HTTP %d: %s", self.StatusCode, self.Message)
}

// IsNotFound reports whether err is a 404 from the API. GitHub also answers
// 404 for private repositories the token can't see.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// repoPath returns the API path for something under the repository, e.g.
// repoPath("actions/runs") -> "/repos/owner/name/actions/runs".
func (self *Client) repoPath(format string, args ...any) string {
	return fmt.Sprintf("/repos/%s/%s/", url.PathEscape(self.repo.Owner), url.PathEscape(self.repo.Name)) +
		fmt.Sprintf(format, args...)
}

// do sends a request and returns the response body of a successful response.
// body, if non-nil, is sent as JSON.
func (self *Client) do(ctx context.Context, method string, path string, query url.Values, body any, accept string) ([]byte, error) {
	u := self.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, err
	}
	if accept == "" {
		accept = "application/vnd.github+json"
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if self.repo.Token != "" {
		// Go drops this header when following a redirect to another host, which
		// matters for logs and artifacts: the API redirects those to a
		// pre-signed blob storage URL that rejects requests carrying it.
		req.Header.Set("Authorization", "Bearer "+self.repo.Token)
	}

	resp, err := self.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		var parsed struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &parsed) == nil {
			apiErr.Message = parsed.Message
		}
		return nil, apiErr
	}

	return respBody, nil
}

func (self *Client) getJSON(ctx context.Context, path string, query url.Values, result any) error {
	body, err := self.do(ctx, http.MethodGet, path, query, nil, "")
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func (self *Client) post(ctx context.Context, path string, body any) error {
	_, err := self.do(ctx, http.MethodPost, path, nil, body, "")
	return err
}

func (self *Client) put(ctx context.Context, path string) error {
	_, err := self.do(ctx, http.MethodPut, path, nil, nil, "")
	return err
}

func (self *Client) delete(ctx context.Context, path string) error {
	_, err := self.do(ctx, http.MethodDelete, path, nil, nil, "")
	return err
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
