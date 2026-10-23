package github_commands

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func graphQLVariables(t *testing.T, request recordedRequest) map[string]any {
	t.Helper()
	var body struct {
		Variables map[string]any `json:"variables"`
	}
	assert.NoError(t, json.Unmarshal([]byte(request.body), &body))
	return body.Variables
}

const pullRequestNodeJSON = `{
	"id": "PR_1", "number": 12, "title": "Add a thing", "state": "OPEN", "isDraft": true,
	"url": "https://github.com/owner/repo/pull/12",
	"createdAt": "2026-10-01T10:00:00Z", "updatedAt": "2026-10-02T10:00:00Z",
	"author": {"login": "octocat"},
	"baseRefName": "master", "headRefName": "thing", "headRefOid": "abc123",
	"headRepository": {"name": "repo-fork"}, "headRepositoryOwner": {"login": "octocat"},
	"isCrossRepository": true, "maintainerCanModify": true,
	"reviewDecision": "REVIEW_REQUIRED", "additions": 10, "deletions": 2,
	"comments": {"totalCount": 3},
	"labels": {"nodes": [{"name": "bug", "color": "d73a4a"}]},
	"commits": {"nodes": [{"commit": {"statusCheckRollup": {"state": "FAILURE"}}}]}
}`

func TestListPullRequests(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"data": {"repository": {"pullRequests": {
		"nodes": [`+pullRequestNodeJSON+`],
		"pageInfo": {"hasNextPage": false}
	}}}}`)

	prs, err := client.ListPullRequests(context.Background(), ListPullRequestsOptions{BaseBranch: "master"})
	assert.NoError(t, err)

	assert.Equal(t, []*models.GithubPullRequest{{
		NodeId:              "PR_1",
		Number:              12,
		Title:               "Add a thing",
		State:               "DRAFT",
		ChecksState:         "FAILURE",
		Url:                 "https://github.com/owner/repo/pull/12",
		HeadRefName:         "thing",
		HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "octocat"},
		Author:              "octocat",
		BaseRefName:         "master",
		HeadRefOid:          "abc123",
		HeadRepositoryName:  "repo-fork",
		IsCrossRepository:   true,
		MaintainerCanModify: true,
		ReviewDecision:      "REVIEW_REQUIRED",
		Labels:              []models.GithubLabel{{Name: "bug", Color: "d73a4a"}},
		Additions:           10,
		Deletions:           2,
		CommentsCount:       3,
		CreatedAt:           time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
	}}, prs)

	assert.Len(t, *requests, 1)
	request := (*requests)[0]
	assert.Equal(t, "POST", request.method)
	assert.Equal(t, "/graphql", request.path)
	assert.Equal(t, "Bearer secret", request.auth)
	assert.Equal(t, map[string]any{
		"owner":      "owner",
		"repo":       "repo",
		"states":     []any{"OPEN"},
		"baseBranch": "master",
		"limit":      float64(30),
	}, graphQLVariables(t, request))
}

func TestListPullRequestsWithSearch(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"data": {"search": {
		"nodes": [`+pullRequestNodeJSON+`, {}],
		"pageInfo": {"hasNextPage": false}
	}}}`)

	prs, err := client.ListPullRequests(context.Background(), ListPullRequestsOptions{
		State:           PullRequestStateMerged,
		Author:          "@me",
		ReviewRequested: "hubot",
		Labels:          []string{"bug", "needs review"},
		Search:          "fix in:title",
		Limit:           30,
	})
	assert.NoError(t, err)
	assert.Len(t, prs, 1)

	assert.Equal(t, map[string]any{
		"q":     `repo:owner/repo is:pr sort:updated-desc is:merged author:@me review-requested:hubot label:bug label:"needs review" fix in:title`,
		"limit": float64(30),
	}, graphQLVariables(t, (*requests)[0]))
}

func TestListPullRequestsReportsGraphQLErrors(t *testing.T) {
	client, _ := newTestClient(t, http.StatusOK, `{"errors": [{"message": "Something went wrong"}, {"message": "and more"}]}`)

	_, err := client.ListPullRequests(context.Background(), ListPullRequestsOptions{})
	assert.EqualError(t, err, "GitHub API request failed: Something went wrong; and more")
}

func TestGetPullRequest(t *testing.T) {
	client, requests := newTestClient(t, http.StatusOK, `{"data": {
		"viewer": {"login": "me"},
		"repository": {
			"defaultBranchRef": {"name": "master"},
			"mergeCommitAllowed": true, "squashMergeAllowed": true, "rebaseMergeAllowed": false,
			"autoMergeAllowed": true, "deleteBranchOnMerge": true,
			"viewerDefaultMergeMethod": "SQUASH", "viewerCanAdminister": false, "viewerPermission": "WRITE",
			"pullRequest": {
				"id": "PR_1", "number": 12, "title": "Add a thing", "state": "OPEN", "isDraft": false,
				"author": null,
				"body": "It adds a thing.",
				"viewerCanUpdate": true,
				"mergeable": "CONFLICTING", "mergeStateStatus": "DIRTY",
				"baseRefOid": "base123",
				"autoMergeRequest": {"mergeMethod": "SQUASH", "enabledBy": {"login": "hubot"}},
				"milestone": {"title": "v1"},
				"assignees": {"nodes": [{"login": "me"}]},
				"reviewRequests": {"nodes": [
					{"requestedReviewer": {"__typename": "User", "login": "alice"}},
					{"requestedReviewer": {"__typename": "Team", "slug": "core", "organization": {"login": "acme"}}}
				]},
				"latestReviews": {"nodes": [
					{"author": {"login": "bob"}, "state": "APPROVED", "submittedAt": "2026-10-03T00:00:00Z"}
				]},
				"reviews": {"nodes": [
					{"id": "R_1", "author": {"login": "bob"}, "state": "APPROVED", "body": "", "submittedAt": "2026-10-03T00:00:00Z"},
					{"id": "R_2", "author": {"login": "carol"}, "state": "COMMENTED", "body": "", "submittedAt": "2026-10-04T00:00:00Z"},
					{"id": "R_3", "author": {"login": "me"}, "state": "PENDING", "body": ""}
				]},
				"pendingReviews": {"nodes": [{"id": "R_3", "comments": {"totalCount": 2}}]},
				"conversation": {"nodes": [
					{"author": {"login": "carol"}, "body": "Nice", "createdAt": "2026-10-02T00:00:00Z", "url": "u1"}
				]},
				"files": {"nodes": [{"path": "a.go", "additions": 3, "deletions": 1, "changeType": "MODIFIED"}]},
				"allCommits": {"totalCount": 4},
				"reviewThreads": {"nodes": [{
					"id": "T_1", "path": "a.go", "isResolved": false, "isOutdated": false,
					"line": 7, "originalLine": 7, "diffSide": "RIGHT",
					"comments": {"nodes": [{"id": "C_1", "databaseId": 99, "author": {"login": "carol"},
						"body": "Why?", "createdAt": "2026-10-04T00:00:00Z", "url": "u2",
						"diffHunk": "@@ -1,2 +1,3 @@", "state": "SUBMITTED"}]}
				}]},
				"checks": {"nodes": [{"commit": {"statusCheckRollup": {"contexts": {"nodes": [
					{"__typename": "CheckRun", "name": "test", "status": "COMPLETED", "conclusion": "FAILURE",
						"detailsUrl": "d1", "databaseId": 555, "isRequired": true,
						"checkSuite": {"workflowRun": {"databaseId": 444, "workflow": {"name": "CI"}}}},
					{"__typename": "StatusContext", "context": "ci/legacy", "state": "PENDING", "targetUrl": "d2"}
				]}}}}]}
			}
		}
	}}`)

	pr, details, err := client.GetPullRequest(context.Background(), 12)
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"owner": "owner", "repo": "repo", "number": float64(12)}, graphQLVariables(t, (*requests)[0]))

	assert.Equal(t, "ghost", pr.Author)
	assert.Equal(t, "OPEN", pr.State)

	assert.Equal(t, &models.GithubPullRequestDetails{
		Body:             "It adds a thing.",
		Assignees:        []string{"me"},
		Milestone:        "v1",
		ReviewRequests:   []string{"alice", "acme/core"},
		LatestReviews:    []*models.GithubReview{{Author: "bob", State: "APPROVED", SubmittedAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)}},
		Mergeable:        "CONFLICTING",
		MergeStateStatus: "DIRTY",
		AutoMerge:        &models.GithubAutoMerge{MergeMethod: "SQUASH", EnabledBy: "hubot"},
		BaseRefOid:       "base123",
		CommitsCount:     4,
		Files:            []*models.GithubPullRequestFile{{Path: "a.go", Additions: 3, Deletions: 1, ChangeType: "MODIFIED"}},
		Timeline: []*models.GithubTimelineItem{
			{Author: "carol", Body: "Nice", CreatedAt: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC), Url: "u1"},
			{Author: "bob", CreatedAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), ReviewState: "APPROVED"},
		},
		ReviewThreads: []*models.GithubReviewThread{{
			NodeId: "T_1", Path: "a.go", Line: 7, OriginalLine: 7, DiffSide: "RIGHT",
			Comments: []*models.GithubReviewComment{{
				NodeId: "C_1", DatabaseId: 99, Author: "carol", Body: "Why?",
				CreatedAt: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC), Url: "u2", DiffHunk: "@@ -1,2 +1,3 @@",
			}},
		}},
		Checks: []*models.GithubCheck{
			{Name: "test", Workflow: "CI", Status: "completed", Conclusion: "failure", Url: "d1", IsRequired: true, RunId: 444, JobId: 555},
			{Name: "ci/legacy", Status: "pending", Url: "d2"},
		},
		PendingReview: &models.GithubPendingReview{NodeId: "R_3", CommentsCount: 2},
		ViewerLogin:   "me",
		ViewerCanEdit: true,
		Repository: models.GithubRepositorySettings{
			DefaultBranch:       "master",
			MergeCommitAllowed:  true,
			SquashMergeAllowed:  true,
			AutoMergeAllowed:    true,
			DeleteBranchOnMerge: true,
			ViewerDefaultMethod: "SQUASH",
			ViewerPermission:    "WRITE",
		},
	}, details)

	assert.Equal(t, models.CheckCounts{Failed: 1, Pending: 1}, details.CheckCounts())
}

func TestGetPullRequestNotFound(t *testing.T) {
	client, _ := newTestClient(t, http.StatusOK, `{"data": {"viewer": {"login": "me"}, "repository": {"pullRequest": null}}}`)

	_, _, err := client.GetPullRequest(context.Background(), 12)
	assert.EqualError(t, err, "pull request #12 not found in owner/repo")
}

func TestGraphQLURL(t *testing.T) {
	assert.Equal(t, "https://api.github.com/graphql", GraphQLURL("github.com"))
	assert.Equal(t, "https://api.acme.ghe.com/graphql", GraphQLURL("acme.ghe.com"))
	assert.Equal(t, "https://git.acme.com/api/graphql", GraphQLURL("git.acme.com"))
}

func TestEliminateDuplicateChecks(t *testing.T) {
	run := func(name string, workflow string, event string, started string) checkContextNode {
		node := checkContextNode{Typename: "CheckRun", Name: name}
		node.StartedAt, _ = time.Parse(time.RFC3339, started)
		node.CheckSuite = &struct {
			WorkflowRun *workflowRunNode `json:"workflowRun"`
		}{WorkflowRun: &workflowRunNode{Event: event}}
		node.CheckSuite.WorkflowRun.Workflow.Name = workflow
		return node
	}
	older := run("label", "Labels", "pull_request", "2026-10-01T00:00:00Z")
	newer := run("label", "Labels", "pull_request", "2026-10-02T00:00:00Z")
	otherEvent := run("label", "Labels", "pull_request_target", "2026-10-01T00:00:00Z")
	test := run("test", "CI", "pull_request", "2026-10-01T00:00:00Z")

	assert.Equal(t, []checkContextNode{newer, otherEvent, test},
		eliminateDuplicateChecks([]checkContextNode{older, newer, otherEvent, test}))
}
