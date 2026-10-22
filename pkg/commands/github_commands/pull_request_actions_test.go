package github_commands

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullRequestMutations(t *testing.T) {
	scenarios := []struct {
		name              string
		call              func(client *Client) error
		expectedOperation string
		expectedVariables map[string]any
	}{
		{
			name: "merge",
			call: func(client *Client) error {
				return client.MergePullRequest(context.Background(), "PR_1", MergePullRequestOptions{
					Method: MergeMethodSquash, CommitHeadline: "Add a thing (#1)", ExpectedHeadOid: "abc",
				})
			},
			expectedOperation: "PullRequestMerge",
			expectedVariables: map[string]any{"input": map[string]any{
				"pullRequestId": "PR_1", "mergeMethod": "SQUASH", "commitHeadline": "Add a thing (#1)", "expectedHeadOid": "abc",
			}},
		},
		{
			name: "enable auto-merge",
			call: func(client *Client) error {
				return client.EnableAutoMerge(context.Background(), "PR_1", MergeMethodRebase)
			},
			expectedOperation: "PullRequestAutoMerge",
			expectedVariables: map[string]any{"input": map[string]any{"pullRequestId": "PR_1", "mergeMethod": "REBASE"}},
		},
		{
			name:              "disable auto-merge",
			call:              func(client *Client) error { return client.DisableAutoMerge(context.Background(), "PR_1") },
			expectedOperation: "PullRequestAutoMergeDisable",
			expectedVariables: map[string]any{"prID": "PR_1"},
		},
		{
			name:              "close",
			call:              func(client *Client) error { return client.ClosePullRequest(context.Background(), "PR_1") },
			expectedOperation: "PullRequestClose",
			expectedVariables: map[string]any{"prID": "PR_1"},
		},
		{
			name:              "reopen",
			call:              func(client *Client) error { return client.ReopenPullRequest(context.Background(), "PR_1") },
			expectedOperation: "PullRequestReopen",
			expectedVariables: map[string]any{"prID": "PR_1"},
		},
		{
			name:              "ready for review",
			call:              func(client *Client) error { return client.MarkPullRequestReadyForReview(context.Background(), "PR_1") },
			expectedOperation: "PullRequestReadyForReview",
			expectedVariables: map[string]any{"prID": "PR_1"},
		},
		{
			name:              "convert to draft",
			call:              func(client *Client) error { return client.ConvertPullRequestToDraft(context.Background(), "PR_1") },
			expectedOperation: "ConvertPullRequestToDraft",
			expectedVariables: map[string]any{"prID": "PR_1"},
		},
		{
			name:              "update branch by rebasing",
			call:              func(client *Client) error { return client.UpdatePullRequestBranch(context.Background(), "PR_1", true) },
			expectedOperation: "PullRequestUpdateBranch",
			expectedVariables: map[string]any{"input": map[string]any{"pullRequestId": "PR_1", "updateMethod": "REBASE"}},
		},
		{
			name:              "comment",
			call:              func(client *Client) error { return client.AddComment(context.Background(), "PR_1", "Nice") },
			expectedOperation: "CommentCreate",
			expectedVariables: map[string]any{"input": map[string]any{"subjectId": "PR_1", "body": "Nice"}},
		},
		{
			name: "review",
			call: func(client *Client) error {
				return client.SubmitReview(context.Background(), "PR_1", "", ReviewEventApprove, "")
			},
			expectedOperation: "PullRequestReviewAdd",
			expectedVariables: map[string]any{"input": map[string]any{"pullRequestId": "PR_1", "event": "APPROVE"}},
		},
		{
			name: "submit pending review",
			call: func(client *Client) error {
				return client.SubmitReview(context.Background(), "PR_1", "R_1", ReviewEventRequestChanges, "Please fix")
			},
			expectedOperation: "PullRequestReviewSubmit",
			expectedVariables: map[string]any{"input": map[string]any{"pullRequestReviewId": "R_1", "event": "REQUEST_CHANGES", "body": "Please fix"}},
		},
		{
			name: "comment on a range of lines",
			call: func(client *Client) error {
				return client.AddReviewComment(context.Background(), "R_1", ReviewComment{
					Path: "a.go", Side: "RIGHT", Line: 9, StartLine: 7, StartSide: "RIGHT", Body: "Why?",
				})
			},
			expectedOperation: "PullRequestReviewThreadAdd",
			expectedVariables: map[string]any{"input": map[string]any{
				"pullRequestReviewId": "R_1", "path": "a.go", "side": "RIGHT", "line": float64(9),
				"startLine": float64(7), "startSide": "RIGHT", "body": "Why?",
			}},
		},
		{
			name: "comment on one line",
			call: func(client *Client) error {
				return client.AddReviewComment(context.Background(), "R_1", ReviewComment{
					Path: "a.go", Side: "LEFT", Line: 3, StartLine: 3, StartSide: "LEFT", Body: "Gone?",
				})
			},
			expectedOperation: "PullRequestReviewThreadAdd",
			expectedVariables: map[string]any{"input": map[string]any{
				"pullRequestReviewId": "R_1", "path": "a.go", "side": "LEFT", "line": float64(3), "body": "Gone?",
			}},
		},
		{
			name: "reply into a pending review",
			call: func(client *Client) error {
				return client.ReplyToReviewThread(context.Background(), "T_1", "R_1", "Fair")
			},
			expectedOperation: "PullRequestReviewThreadReply",
			expectedVariables: map[string]any{"input": map[string]any{
				"pullRequestReviewThreadId": "T_1", "pullRequestReviewId": "R_1", "body": "Fair",
			}},
		},
		{
			name:              "resolve thread",
			call:              func(client *Client) error { return client.ResolveReviewThread(context.Background(), "T_1") },
			expectedOperation: "PullRequestReviewThreadResolve",
			expectedVariables: map[string]any{"threadId": "T_1"},
		},
		{
			name:              "discard pending review",
			call:              func(client *Client) error { return client.DeletePendingReview(context.Background(), "R_1") },
			expectedOperation: "PullRequestReviewDelete",
			expectedVariables: map[string]any{"input": map[string]any{"pullRequestReviewId": "R_1"}},
		},
	}

	operationName := regexp.MustCompile(`mutation (\w+)`)
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			client, requests := newTestClient(t, http.StatusOK, `{"data": {}}`)
			assert.NoError(t, s.call(client))

			assert.Len(t, *requests, 1)
			var body struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			assert.NoError(t, json.Unmarshal([]byte((*requests)[0].body), &body))
			assert.Equal(t, s.expectedOperation, operationName.FindStringSubmatch(body.Query)[1])
			assert.Equal(t, s.expectedVariables, body.Variables)
		})
	}
}

func TestDeleteBranch(t *testing.T) {
	client, requests := newTestClient(t, http.StatusNoContent, "")
	assert.NoError(t, client.DeleteBranch(context.Background(), "feature/a b"))
	assert.Equal(t, "DELETE", (*requests)[0].method)
	assert.Equal(t, "/repos/owner/repo/git/refs/heads/feature/a b", (*requests)[0].path)
}

func TestStartPendingReview(t *testing.T) {
	client, _ := newTestClient(t, http.StatusOK, `{"data": {"addPullRequestReview": {"pullRequestReview": {"id": "R_9"}}}}`)
	id, err := client.StartPendingReview(context.Background(), "PR_1")
	assert.NoError(t, err)
	assert.Equal(t, "R_9", id)
}
