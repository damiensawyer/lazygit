package github_commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// The calls in this file aren't about the client's repository, but about
// finding one, so they ignore its owner and name.

type restRepository struct {
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	Owner    struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description string          `json:"description"`
	Fork        bool            `json:"fork"`
	Private     bool            `json:"private"`
	Archived    bool            `json:"archived"`
	Stars       int             `json:"stargazers_count"`
	Forks       int             `json:"forks_count"`
	Language    string          `json:"language"`
	PushedAt    time.Time       `json:"pushed_at"`
	HTMLURL     string          `json:"html_url"`
	SSHURL      string          `json:"ssh_url"`
	CloneURL    string          `json:"clone_url"`
	Parent      *restRepository `json:"parent"`
}

func (self *restRepository) toModel() *models.GithubRepository {
	if self == nil {
		return nil
	}
	return &models.GithubRepository{
		FullName:    self.FullName,
		Owner:       self.Owner.Login,
		Name:        self.Name,
		Description: self.Description,
		IsFork:      self.Fork,
		IsPrivate:   self.Private,
		IsArchived:  self.Archived,
		Stars:       self.Stars,
		Forks:       self.Forks,
		Language:    self.Language,
		PushedAt:    self.PushedAt,
		HTMLURL:     self.HTMLURL,
		SSHURL:      self.SSHURL,
		CloneURL:    self.CloneURL,
		Parent:      self.Parent.toModel(),
	}
}

func restRepositoriesToModels(repos []*restRepository) []*models.GithubRepository {
	return lo.Map(repos, func(repo *restRepository, _ int) *models.GithubRepository {
		return repo.toModel()
	})
}

const graphQLRepositoryFields = `
	nameWithOwner name owner { login } description isFork isPrivate isArchived
	stargazerCount forkCount primaryLanguage { name } pushedAt url sshUrl
	parent { nameWithOwner name owner { login } url sshUrl }`

type graphQLRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
	Name          string `json:"name"`
	Owner         struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description     string `json:"description"`
	IsFork          bool   `json:"isFork"`
	IsPrivate       bool   `json:"isPrivate"`
	IsArchived      bool   `json:"isArchived"`
	StargazerCount  int    `json:"stargazerCount"`
	ForkCount       int    `json:"forkCount"`
	PrimaryLanguage *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	PushedAt time.Time          `json:"pushedAt"`
	URL      string             `json:"url"`
	SSHURL   string             `json:"sshUrl"`
	Parent   *graphQLRepository `json:"parent"`
}

func (self *graphQLRepository) toModel() *models.GithubRepository {
	if self == nil {
		return nil
	}
	language := ""
	if self.PrimaryLanguage != nil {
		language = self.PrimaryLanguage.Name
	}
	return &models.GithubRepository{
		FullName:    self.NameWithOwner,
		Owner:       self.Owner.Login,
		Name:        self.Name,
		Description: self.Description,
		IsFork:      self.IsFork,
		IsPrivate:   self.IsPrivate,
		IsArchived:  self.IsArchived,
		Stars:       self.StargazerCount,
		Forks:       self.ForkCount,
		Language:    language,
		PushedAt:    self.PushedAt,
		HTMLURL:     self.URL,
		SSHURL:      self.SSHURL,
		CloneURL:    self.URL + ".git",
		Parent:      self.Parent.toModel(),
	}
}

// ViewerRepositories selects which of the authenticated user's repositories
// ListViewerRepositories lists.
type ViewerRepositories int

const (
	// The repositories the user owns, as `gh repo list` lists them.
	ViewerOwnedRepositories ViewerRepositories = iota
	// The repositories of the user's organizations, and those the user
	// collaborates on, without the user's own.
	ViewerOtherRepositories
)

// ListViewerRepositories lists the authenticated user's repositories, most
// recently pushed to first, handing them to onPage a page at a time so that
// they can be shown while the rest are still loading. It stops after limit
// repositories, and returns the user's login.
func (self *Client) ListViewerRepositories(
	ctx context.Context,
	which ViewerRepositories,
	limit int,
	onPage func([]*models.GithubRepository),
) (string, error) {
	affiliations := "affiliations: [OWNER], ownerAffiliations: [OWNER]"
	if which == ViewerOtherRepositories {
		affiliations = "affiliations: [ORGANIZATION_MEMBER, COLLABORATOR], ownerAffiliations: [ORGANIZATION_MEMBER, COLLABORATOR]"
	}
	query := `query($perPage: Int!, $cursor: String) {
		viewer {
			login
			repositories(first: $perPage, after: $cursor, ` + affiliations + `, orderBy: {field: PUSHED_AT, direction: DESC}) {
				nodes {` + graphQLRepositoryFields + `}
				pageInfo { hasNextPage endCursor }
			}
		}
	}`

	login := ""
	count := 0
	variables := map[string]any{"perPage": min(limit, 100)}
	for count < limit {
		var result struct {
			Viewer struct {
				Login        string `json:"login"`
				Repositories struct {
					Nodes    []*graphQLRepository `json:"nodes"`
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
				} `json:"repositories"`
			} `json:"viewer"`
		}
		if err := self.graphQL(ctx, query, variables, &result); err != nil {
			return login, err
		}

		login = result.Viewer.Login
		nodes := result.Viewer.Repositories.Nodes
		nodes = nodes[:min(len(nodes), limit-count)]
		count += len(nodes)
		onPage(lo.Map(nodes, func(node *graphQLRepository, _ int) *models.GithubRepository {
			return node.toModel()
		}))

		pageInfo := result.Viewer.Repositories.PageInfo
		if !pageInfo.HasNextPage {
			break
		}
		variables["cursor"] = pageInfo.EndCursor
	}
	return login, nil
}

// ListStarredRepositories lists the repositories the authenticated user has
// starred, most recently starred first, handing them to onPage a page at a
// time. It stops after limit repositories.
func (self *Client) ListStarredRepositories(ctx context.Context, limit int, onPage func([]*models.GithubRepository)) error {
	perPage := min(limit, 100)
	count := 0
	for page := 1; count < limit; page++ {
		var repos []*restRepository
		query := url.Values{"per_page": {itoa(int64(perPage))}, "page": {itoa(int64(page))}}
		if err := self.getJSON(ctx, "/user/starred", query, &repos); err != nil {
			return err
		}

		repos = repos[:min(len(repos), limit-count)]
		count += len(repos)
		onPage(restRepositoriesToModels(repos))

		if len(repos) < perPage {
			break
		}
	}
	return nil
}

// SearchRepositoriesResult is one page of repository search results.
type SearchRepositoriesResult struct {
	// How many repositories match in all, of which the search API serves only
	// the first thousand.
	TotalCount int
	// The search timed out before finding all matches.
	Incomplete   bool
	Repositories []*models.GithubRepository
}

// SearchRepositories runs a repository search, the way `gh search repos`
// does, returning one page of the matches in GitHub's best-match order. The
// query may contain qualifiers like "user:jesseduffield".
func (self *Client) SearchRepositories(ctx context.Context, query string, page int, perPage int) (*SearchRepositoriesResult, error) {
	var result struct {
		TotalCount        int               `json:"total_count"`
		IncompleteResults bool              `json:"incomplete_results"`
		Items             []*restRepository `json:"items"`
	}
	params := url.Values{
		"q":        {query},
		"page":     {itoa(int64(page))},
		"per_page": {itoa(int64(perPage))},
	}
	if err := self.getJSON(ctx, "/search/repositories", params, &result); err != nil {
		return nil, err
	}
	return &SearchRepositoriesResult{
		TotalCount:   result.TotalCount,
		Incomplete:   result.IncompleteResults,
		Repositories: restRepositoriesToModels(result.Items),
	}, nil
}

// GetRepository looks up a repository by owner and name. Unlike the
// listings, the result says what the repository is a fork of.
func (self *Client) GetRepository(ctx context.Context, owner string, name string) (*models.GithubRepository, error) {
	var repo restRepository
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name)
	if err := self.getJSON(ctx, path, nil, &repo); err != nil {
		return nil, err
	}
	return repo.toModel(), nil
}

// CreateFork forks a repository into the authenticated user's account, and
// returns the fork. If the user has forked it before, GitHub returns the
// existing fork instead. A new fork's git data may take a few seconds to
// become available, so cloning it right away can fail.
func (self *Client) CreateFork(ctx context.Context, owner string, name string) (*models.GithubRepository, error) {
	path := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(name) + "/forks"
	body, err := self.do(ctx, http.MethodPost, path, nil, map[string]any{}, "")
	if err != nil {
		return nil, err
	}
	var repo restRepository
	if err := json.Unmarshal(body, &repo); err != nil {
		return nil, err
	}
	return repo.toModel(), nil
}

// ParseRepositoryName parses "owner/name", or the URL of a repository's web
// page or of one of its git remotes on the given host, into owner and name.
func ParseRepositoryName(host string, s string) (owner string, name string, ok bool) {
	s = strings.TrimSpace(s)
	hasScheme := false
	for _, scheme := range []string{"https://", "http://", "ssh://"} {
		if strings.HasPrefix(s, scheme) {
			s = strings.TrimPrefix(s, scheme)
			hasScheme = true
		}
	}
	// Drop user info, as in git@github.com:owner/name.git
	if at := strings.Index(s, "@"); at >= 0 && at < strings.IndexAny(s, "/:") {
		s = s[at+1:]
	}
	s = strings.TrimPrefix(s, "www.")

	hasHost := false
	for _, separator := range []string{"/", ":"} {
		if rest, found := strings.CutPrefix(s, host+separator); found {
			s = rest
			hasHost = true
		}
	}
	if !hasHost && (hasScheme || strings.Contains(s, ":")) {
		return "", "", false
	}

	parts := strings.Split(strings.TrimSuffix(s, "/"), "/")
	// A URL may go on to a page of the repository, e.g. ".../pulls", but a
	// bare name has nothing after it.
	if len(parts) < 2 || (!hasHost && len(parts) > 2) {
		return "", "", false
	}
	owner, name = parts[0], strings.TrimSuffix(parts[1], ".git")
	if !isValidRepositoryNamePart(owner) || !isValidRepositoryNamePart(name) {
		return "", "", false
	}
	return owner, name, true
}

func isValidRepositoryNamePart(s string) bool {
	return s != "" && strings.IndexFunc(s, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.')
	}) == -1
}
