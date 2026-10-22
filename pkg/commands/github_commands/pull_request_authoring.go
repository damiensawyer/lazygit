package github_commands

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// The requests below are the ones `gh pr create` and `gh pr edit` make, for
// writing a pull request and for changing who and what it involves.

// GetDefaultBranch returns the name of the repository's default branch.
func (self *Client) GetDefaultBranch(ctx context.Context) (string, error) {
	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}
	path := fmt.Sprintf("/repos/%s/%s", url.PathEscape(self.repo.Owner), url.PathEscape(self.repo.Name))
	if err := self.getJSON(ctx, path, nil, &repo); err != nil {
		return "", err
	}
	return repo.DefaultBranch, nil
}

type CreatePullRequestOptions struct {
	Title string
	Body  string
	// the branch to merge, prefixed with "owner:" if it is in a fork
	Head  string
	Base  string
	Draft bool
}

// CreatedPullRequest identifies a pull request that was just created.
type CreatedPullRequest struct {
	Number int    `json:"number"`
	Url    string `json:"html_url"`
}

func (self *Client) CreatePullRequest(ctx context.Context, opts CreatePullRequestOptions) (*CreatedPullRequest, error) {
	body, err := self.do(ctx, http.MethodPost, self.repoPath("pulls"), nil, map[string]any{
		"title": opts.Title,
		"body":  opts.Body,
		"head":  opts.Head,
		"base":  opts.Base,
		"draft": opts.Draft,
		// what gh and GitHub's web UI default to
		"maintainer_can_modify": true,
	}, "")
	if err != nil {
		return nil, err
	}
	var created CreatedPullRequest
	if err := json.Unmarshal(body, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdatePullRequestOptions are the things about a pull request that can be
// edited directly. Nil fields are left as they are.
type UpdatePullRequestOptions struct {
	Title *string
	Body  *string
	Base  *string
}

func (self *Client) UpdatePullRequest(ctx context.Context, number int, opts UpdatePullRequestOptions) error {
	fields := map[string]any{}
	if opts.Title != nil {
		fields["title"] = *opts.Title
	}
	if opts.Body != nil {
		fields["body"] = *opts.Body
	}
	if opts.Base != nil {
		fields["base"] = *opts.Base
	}
	_, err := self.do(ctx, http.MethodPatch, self.repoPath("pulls/%d", number), nil, fields, "")
	return err
}

// ListLabels returns the labels defined in the repository.
func (self *Client) ListLabels(ctx context.Context) ([]models.GithubLabel, error) {
	var labels []models.GithubLabel
	err := self.getJSON(ctx, self.repoPath("labels"), url.Values{"per_page": {"100"}}, &labels)
	return labels, err
}

func (self *Client) AddLabels(ctx context.Context, number int, labels []string) error {
	return self.post(ctx, self.repoPath("issues/%d/labels", number), map[string]any{"labels": labels})
}

func (self *Client) RemoveLabel(ctx context.Context, number int, label string) error {
	return self.delete(ctx, self.repoPath("issues/%d/labels/%s", number, url.PathEscape(label)))
}

// ListAssignableUsers returns the logins of the users that issues and pull
// requests of the repository can be assigned to, and whose review can be
// requested.
func (self *Client) ListAssignableUsers(ctx context.Context) ([]string, error) {
	var users []models.GithubUser
	if err := self.getJSON(ctx, self.repoPath("assignees"), url.Values{"per_page": {"100"}}, &users); err != nil {
		return nil, err
	}
	return lo.Map(users, func(user models.GithubUser, _ int) string { return user.Login }), nil
}

func (self *Client) AddAssignees(ctx context.Context, number int, logins []string) error {
	return self.post(ctx, self.repoPath("issues/%d/assignees", number), map[string]any{"assignees": logins})
}

func (self *Client) RemoveAssignees(ctx context.Context, number int, logins []string) error {
	_, err := self.do(ctx, http.MethodDelete, self.repoPath("issues/%d/assignees", number), nil,
		map[string]any{"assignees": logins}, "")
	return err
}

// splitReviewers tells the users among reviewers from the teams, which are
// written as "org/team".
func splitReviewers(reviewers []string) map[string]any {
	users := []string{}
	teams := []string{}
	for _, reviewer := range reviewers {
		if _, team, isTeam := strings.Cut(reviewer, "/"); isTeam {
			teams = append(teams, team)
		} else {
			users = append(users, reviewer)
		}
	}
	return map[string]any{"reviewers": users, "team_reviewers": teams}
}

// RequestReviews requests reviews of a pull request from users, and from
// teams written as "org/team".
func (self *Client) RequestReviews(ctx context.Context, number int, reviewers []string) error {
	return self.post(ctx, self.repoPath("pulls/%d/requested_reviewers", number), splitReviewers(reviewers))
}

func (self *Client) RemoveReviewRequests(ctx context.Context, number int, reviewers []string) error {
	_, err := self.do(ctx, http.MethodDelete, self.repoPath("pulls/%d/requested_reviewers", number), nil,
		splitReviewers(reviewers), "")
	return err
}
