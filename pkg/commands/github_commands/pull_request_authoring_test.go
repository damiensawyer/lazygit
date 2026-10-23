package github_commands

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func requestBody(t *testing.T, request recordedRequest) map[string]any {
	t.Helper()
	var body map[string]any
	assert.NoError(t, json.Unmarshal([]byte(request.body), &body))
	return body
}

func TestCreatePullRequest(t *testing.T) {
	client, requests := newTestClient(t, http.StatusCreated, `{"number": 13, "html_url": "https://github.com/owner/repo/pull/13"}`)

	created, err := client.CreatePullRequest(context.Background(), CreatePullRequestOptions{
		Title: "Add a widget", Body: "It's useful.", Head: "fork-owner:widget", Base: "master", Draft: true,
	})
	assert.NoError(t, err)
	assert.Equal(t, &CreatedPullRequest{Number: 13, Url: "https://github.com/owner/repo/pull/13"}, created)

	assert.Equal(t, "POST", (*requests)[0].method)
	assert.Equal(t, "/repos/owner/repo/pulls", (*requests)[0].path)
	assert.Equal(t, map[string]any{
		"title": "Add a widget", "body": "It's useful.", "head": "fork-owner:widget", "base": "master",
		"draft": true, "maintainer_can_modify": true,
	}, requestBody(t, (*requests)[0]))
}

func TestUpdatePullRequestOnlySendsWhatChanged(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{}`)

	base := "develop"
	assert.NoError(t, client.UpdatePullRequest(context.Background(), 12, UpdatePullRequestOptions{Base: &base}))
	assert.Equal(t, "PATCH", (*requests)[0].method)
	assert.Equal(t, "/repos/owner/repo/pulls/12", (*requests)[0].path)
	assert.Equal(t, map[string]any{"base": "develop"}, requestBody(t, (*requests)[0]))
}

func TestRequestReviewsFromUsersAndTeams(t *testing.T) {
	client, requests := newTestClient(t, http.StatusCreated, `{}`)

	assert.NoError(t, client.RequestReviews(context.Background(), 12, []string{"alice", "acme/core"}))
	assert.Equal(t, "/repos/owner/repo/pulls/12/requested_reviewers", (*requests)[0].path)
	assert.Equal(t, map[string]any{"reviewers": []any{"alice"}, "team_reviewers": []any{"core"}}, requestBody(t, (*requests)[0]))
}

func TestGetDefaultBranch(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"default_branch": "main"}`)

	branch, err := client.GetDefaultBranch(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "main", branch)
	assert.Equal(t, "/repos/owner/repo", (*requests)[0].path)
}

func TestRemoveLabelEscapesTheName(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `[]`)

	assert.NoError(t, client.RemoveLabel(context.Background(), 12, "needs review"))
	assert.Equal(t, "DELETE", (*requests)[0].method)
	assert.Equal(t, "/repos/owner/repo/issues/12/labels/needs review", (*requests)[0].path)
}
