// Package github_commands talks to the GitHub API about a repository: its pull
// requests, and its GitHub Actions workflows, their runs, and the jobs, logs,
// and artifacts those runs produce.
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
	"github.com/samber/lo"
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

// GraphQLURL returns the GraphQL API endpoint for a GitHub host. GitHub
// Enterprise Server serves it from /api/graphql rather than next to the REST
// API's /api/v3.
func GraphQLURL(host string) string {
	if auth.IsEnterprise(host) {
		return fmt.Sprintf("https://%s/api/graphql", host)
	}
	return RestAPIBaseURL(host) + "/graphql"
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
	graphQLURL string
	repo       Repo
}

func NewClient(repo Repo) *Client {
	client := NewClientWithBaseURL(repo, RestAPIBaseURL(repo.Host))
	client.graphQLURL = GraphQLURL(repo.Host)
	return client
}

// NewClientWithBaseURL is for tests, which point the client at a fake server.
// It serves the GraphQL API from /graphql, the way github.com does.
func NewClientWithBaseURL(repo Repo, baseURL string) *Client {
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &Client{
		// Downloading a large log or artifact can take a while, but a request
		// that hangs indefinitely shouldn't leave a spinner up forever.
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		baseURL:    baseURL,
		graphQLURL: baseURL + "/graphql",
		repo:       repo,
	}
}

// WithToken returns a client for the same repository that authenticates with
// another token.
func (self *Client) WithToken(token string) *Client {
	client := *self
	client.repo.Token = token
	return &client
}

func (self *Client) Repo() Repo {
	return self.repo
}

// APIError is a non-2xx response from the API.
type APIError struct {
	StatusCode int
	Message    string
	// When the request was refused for exceeding a rate limit, the time at
	// which the limit resets; zero otherwise.
	RateLimitReset time.Time
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

// RateLimitReset reports whether err is a request refused for exceeding a rate
// limit, and if so, when it is worth trying again.
func RateLimitReset(err error) (time.Time, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) && !apiErr.RateLimitReset.IsZero() {
		return apiErr.RateLimitReset, true
	}
	return time.Time{}, false
}

// IsUnprocessable reports whether err is a 422 from the API, which it answers
// e.g. for deleting a branch that doesn't exist any more.
func IsUnprocessable(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnprocessableEntity
}

// repoPath returns the API path for something under the repository, e.g.
// repoPath("actions/runs") -> "/repos/owner/name/actions/runs".
func (self *Client) repoPath(format string, args ...any) string {
	return fmt.Sprintf("/repos/%s/%s/", url.PathEscape(self.repo.Owner), url.PathEscape(self.repo.Name)) +
		fmt.Sprintf(format, args...)
}

// do sends a request to the REST API and returns the response body of a
// successful response. body, if non-nil, is sent as JSON.
func (self *Client) do(ctx context.Context, method string, path string, query url.Values, body any, accept string) ([]byte, error) {
	u := self.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return self.send(ctx, method, u, body, accept)
}

// A GET that GitHub answers with a gateway error is tried once more after
// this long: those are usually a passing hiccup on GitHub's side, not worth
// interrupting anyone over. Other methods aren't retried, since they may have
// taken effect.
const gatewayErrorRetryDelay = 500 * time.Millisecond

func (self *Client) send(ctx context.Context, method string, u string, body any, accept string) ([]byte, error) {
	respBody, err := self.sendOnce(ctx, method, u, body, accept)
	if method != http.MethodGet || !isGatewayError(err) {
		return respBody, err
	}
	select {
	case <-ctx.Done():
		return respBody, err
	case <-time.After(gatewayErrorRetryDelay):
	}
	return self.sendOnce(ctx, method, u, body, accept)
}

func isGatewayError(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.StatusCode {
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

func (self *Client) sendOnce(ctx context.Context, method string, u string, body any, accept string) ([]byte, error) {
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
		apiErr := &APIError{StatusCode: resp.StatusCode, RateLimitReset: rateLimitReset(resp, time.Now())}
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

// rateLimitReset returns when a refused request may be retried, if it was
// refused for exceeding a rate limit. GitHub answers 403 or 429 for both its
// primary limits, saying in X-RateLimit-Reset when they reset, and its
// secondary ones, saying in Retry-After how long to wait.
func rateLimitReset(resp *http.Response, now time.Time) time.Time {
	if resp.StatusCode != http.StatusForbidden && resp.StatusCode != http.StatusTooManyRequests {
		return time.Time{}
	}
	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		return now.Add(time.Duration(seconds) * time.Second)
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if epoch, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			return time.Unix(epoch, 0)
		}
	}
	return time.Time{}
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

// GraphQLError is a GraphQL request that GitHub answered with errors, e.g.
// because a mutation wasn't allowed.
type GraphQLError struct {
	Messages []string
}

func (self *GraphQLError) Error() string {
	return "GitHub API request failed: " + strings.Join(self.Messages, "; ")
}

type graphQLErrorMessage struct {
	Message string `json:"message"`
}

// graphQL sends a GraphQL query or mutation and decodes the data of the
// response into result.
func (self *Client) graphQL(ctx context.Context, query string, variables map[string]any, result any) error {
	body, err := self.send(ctx, http.MethodPost, self.graphQLURL, map[string]any{
		"query":     query,
		"variables": variables,
	}, "application/json")
	if err != nil {
		return err
	}

	var response struct {
		Data   json.RawMessage       `json:"data"`
		Errors []graphQLErrorMessage `json:"errors"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	if len(response.Errors) > 0 {
		return &GraphQLError{Messages: lo.Map(response.Errors, func(e graphQLErrorMessage, _ int) string {
			return e.Message
		})}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(response.Data, result)
}

func (self *Client) delete(ctx context.Context, path string) error {
	_, err := self.do(ctx, http.MethodDelete, path, nil, nil, "")
	return err
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
