package github_commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestListViewerRepositories(t *testing.T) {
	queries := []string{}
	cursors := []any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		body, _ := io.ReadAll(r.Body)
		assert.NoError(t, json.Unmarshal(body, &request))
		queries = append(queries, request.Query)
		cursors = append(cursors, request.Variables["cursor"])

		page := `{"data": {"viewer": {"login": "me", "repositories": {
			"nodes": [
				{"nameWithOwner": "me/fork", "name": "fork", "owner": {"login": "me"}, "isFork": true,
				 "primaryLanguage": {"name": "Go"}, "url": "https://github.com/me/fork", "sshUrl": "git@github.com:me/fork.git",
				 "parent": {"nameWithOwner": "them/fork", "name": "fork", "owner": {"login": "them"},
				            "url": "https://github.com/them/fork", "sshUrl": "git@github.com:them/fork.git"}},
				{"nameWithOwner": "me/plain", "name": "plain", "owner": {"login": "me"}, "stargazerCount": 3}
			],
			"pageInfo": {"hasNextPage": true, "endCursor": "abc"}}}}}`
		_, _ = w.Write([]byte(page))
	}))
	t.Cleanup(server.Close)
	client := NewClientWithBaseURL(Repo{Host: "github.com", Token: "secret"}, server.URL)

	pages := [][]*models.GithubRepository{}
	login, err := client.ListViewerRepositories(context.Background(), ViewerOwnedRepositories, 3,
		func(repos []*models.GithubRepository) { pages = append(pages, repos) })
	assert.NoError(t, err)

	assert.Equal(t, "me", login)
	// The limit cuts the second page short and stops paging.
	assert.Equal(t, []any{nil, "abc"}, cursors)
	assert.Equal(t, []int{2, 1}, lo.Map(pages, func(page []*models.GithubRepository, _ int) int { return len(page) }))
	assert.Contains(t, queries[0], "ownerAffiliations: [OWNER]")

	fork := pages[0][0]
	assert.Equal(t, "me/fork", fork.FullName)
	assert.Equal(t, "me", fork.Owner)
	assert.Equal(t, "Go", fork.Language)
	assert.Equal(t, "https://github.com/me/fork.git", fork.CloneURL)
	assert.Equal(t, "them/fork", fork.Parent.FullName)
	assert.Equal(t, "git@github.com:them/fork.git", fork.Parent.SSHURL)
	assert.Nil(t, pages[0][1].Parent)
	assert.Empty(t, pages[0][1].Language)
}

func TestListViewerRepositoriesOfOthers(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK,
		`{"data": {"viewer": {"login": "me", "repositories": {"nodes": [], "pageInfo": {"hasNextPage": false}}}}}`)

	_, err := client.ListViewerRepositories(context.Background(), ViewerOtherRepositories, 100, func([]*models.GithubRepository) {})
	assert.NoError(t, err)

	assert.Len(t, *requests, 1)
	assert.Contains(t, (*requests)[0].body, "ownerAffiliations: [ORGANIZATION_MEMBER, COLLABORATOR]")
}

func TestListStarredRepositories(t *testing.T) {
	requestedPages := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/user/starred", r.URL.Path)
		page := r.URL.Query().Get("page")
		requestedPages = append(requestedPages, page)
		// A full first page, then a short one that ends the listing.
		count, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
		if page == "2" {
			count = 1
		}
		repos := lo.Times(count, func(i int) string {
			return fmt.Sprintf(`{"full_name": "o/r%s-%d", "owner": {"login": "o"}, "stargazers_count": 10}`, page, i)
		})
		_, _ = w.Write([]byte("[" + strings.Join(repos, ",") + "]"))
	}))
	t.Cleanup(server.Close)
	client := NewClientWithBaseURL(Repo{Host: "github.com", Token: "secret"}, server.URL)

	names := []string{}
	err := client.ListStarredRepositories(context.Background(), 2, func(repos []*models.GithubRepository) {
		for _, repo := range repos {
			names = append(names, repo.FullName)
		}
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"o/r1-0", "o/r1-1"}, names)
	assert.Equal(t, []string{"1"}, requestedPages)

	names = nil
	requestedPages = nil
	err = client.ListStarredRepositories(context.Background(), 1000, func(repos []*models.GithubRepository) {
		for _, repo := range repos {
			names = append(names, repo.FullName)
		}
	})
	assert.NoError(t, err)
	assert.Len(t, names, 101)
	assert.Equal(t, "o/r2-0", names[100])
	assert.Equal(t, []string{"1", "2"}, requestedPages)
}

func TestSearchRepositories(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"total_count": 71854, "incomplete_results": true, "items": [
		{"full_name": "jesseduffield/lazygit", "name": "lazygit", "owner": {"login": "jesseduffield"},
		 "description": "simple terminal UI for git commands", "stargazers_count": 58000, "forks_count": 2100,
		 "language": "Go", "pushed_at": "2026-10-25T10:00:00Z", "html_url": "https://github.com/jesseduffield/lazygit",
		 "ssh_url": "git@github.com:jesseduffield/lazygit.git", "clone_url": "https://github.com/jesseduffield/lazygit.git"}
	]}`)

	result, err := client.SearchRepositories(context.Background(), "laz user:jesseduffield", 2, 100)
	assert.NoError(t, err)

	assert.Equal(t, "/search/repositories", (*requests)[0].path)
	assert.Equal(t, "page=2&per_page=100&q=laz+user%3Ajesseduffield", (*requests)[0].query)
	assert.Equal(t, 71854, result.TotalCount)
	assert.True(t, result.Incomplete)
	assert.Len(t, result.Repositories, 1)
	repo := result.Repositories[0]
	assert.Equal(t, "jesseduffield/lazygit", repo.FullName)
	assert.Equal(t, "lazygit", repo.Name)
	assert.Equal(t, "jesseduffield", repo.Owner)
	assert.Equal(t, 58000, repo.Stars)
	assert.Equal(t, 2100, repo.Forks)
	assert.Equal(t, 2026, repo.PushedAt.Year())
	assert.Equal(t, "git@github.com:jesseduffield/lazygit.git", repo.SSHURL)
}

func TestGetRepository(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"full_name": "me/lazygit", "fork": true,
		"parent": {"full_name": "jesseduffield/lazygit", "clone_url": "https://github.com/jesseduffield/lazygit.git"}}`)

	repo, err := client.GetRepository(context.Background(), "me", "lazygit")
	assert.NoError(t, err)

	assert.Equal(t, "/repos/me/lazygit", (*requests)[0].path)
	assert.True(t, repo.IsFork)
	assert.Equal(t, "https://github.com/jesseduffield/lazygit.git", repo.Parent.CloneURL)
}

func TestCreateFork(t *testing.T) {
	client, requests := newTestClient(t, http.StatusAccepted, `{"full_name": "me/lazygit", "owner": {"login": "me"},
		"clone_url": "https://github.com/me/lazygit.git"}`)

	fork, err := client.CreateFork(context.Background(), "jesseduffield", "lazygit")
	assert.NoError(t, err)

	assert.Equal(t, http.MethodPost, (*requests)[0].method)
	assert.Equal(t, "/repos/jesseduffield/lazygit/forks", (*requests)[0].path)
	assert.Equal(t, "me/lazygit", fork.FullName)
	assert.Equal(t, "https://github.com/me/lazygit.git", fork.CloneURL)
}

func TestParseRepositoryName(t *testing.T) {
	scenarios := []struct {
		input string
		owner string
		name  string
	}{
		{"jesseduffield/lazygit", "jesseduffield", "lazygit"},
		{"  jesseduffield/lazygit  ", "jesseduffield", "lazygit"},
		{"https://github.com/jesseduffield/lazygit", "jesseduffield", "lazygit"},
		{"https://www.github.com/jesseduffield/lazygit/", "jesseduffield", "lazygit"},
		{"https://github.com/jesseduffield/lazygit/pulls/123", "jesseduffield", "lazygit"},
		{"github.com/jesseduffield/lazy.nvim", "jesseduffield", "lazy.nvim"},
		{"https://github.com/jesseduffield/lazygit.git", "jesseduffield", "lazygit"},
		{"https://user@github.com/jesseduffield/lazygit.git", "jesseduffield", "lazygit"},
		{"git@github.com:jesseduffield/lazygit.git", "jesseduffield", "lazygit"},
		{"ssh://git@github.com/jesseduffield/lazygit.git", "jesseduffield", "lazygit"},
		{"jes laz", "", ""},
		{"lazygit", "", ""},
		{"jesseduffield/lazygit/more", "", ""},
		{"https://gitlab.com/jesseduffield/lazygit", "", ""},
		{"git@gitlab.com:jesseduffield/lazygit.git", "", ""},
		{"user:jesseduffield", "", ""},
		{"https://github.com/jesseduffield", "", ""},
	}

	for _, s := range scenarios {
		t.Run(s.input, func(t *testing.T) {
			owner, name, ok := ParseRepositoryName("github.com", s.input)
			assert.Equal(t, s.owner != "", ok)
			assert.Equal(t, s.owner, owner)
			assert.Equal(t, s.name, name)
		})
	}
}
