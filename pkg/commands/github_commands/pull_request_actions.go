package github_commands

import (
	"context"
	"net/url"
	"strings"

	"github.com/samber/lo"
)

// The mutations below are the ones `gh pr merge`, `gh pr close`, `gh pr
// ready`, `gh pr review`, `gh pr comment` and `gh pr update-branch` make.

// Merge methods, as the GraphQL API spells them.
const (
	MergeMethodMerge  = "MERGE"
	MergeMethodSquash = "SQUASH"
	MergeMethodRebase = "REBASE"
)

type MergePullRequestOptions struct {
	Method string
	// for a merge or squash commit; GitHub's defaults are used if empty
	CommitHeadline string
	CommitBody     string
	// refuse to merge if the head has moved on since the pull request was
	// looked at
	ExpectedHeadOid string
}

func (self *Client) MergePullRequest(ctx context.Context, prId string, opts MergePullRequestOptions) error {
	input := map[string]any{"pullRequestId": prId, "mergeMethod": opts.Method}
	if opts.CommitHeadline != "" {
		input["commitHeadline"] = opts.CommitHeadline
	}
	if opts.CommitBody != "" {
		input["commitBody"] = opts.CommitBody
	}
	if opts.ExpectedHeadOid != "" {
		input["expectedHeadOid"] = opts.ExpectedHeadOid
	}
	return self.graphQL(ctx, `mutation PullRequestMerge($input: MergePullRequestInput!) {
	mergePullRequest(input: $input) { clientMutationId }
}`, map[string]any{"input": input}, nil)
}

// EnableAutoMerge makes GitHub merge the pull request once its requirements
// are met.
func (self *Client) EnableAutoMerge(ctx context.Context, prId string, method string) error {
	return self.graphQL(ctx, `mutation PullRequestAutoMerge($input: EnablePullRequestAutoMergeInput!) {
	enablePullRequestAutoMerge(input: $input) { clientMutationId }
}`, map[string]any{"input": map[string]any{"pullRequestId": prId, "mergeMethod": method}}, nil)
}

func (self *Client) DisableAutoMerge(ctx context.Context, prId string) error {
	return self.graphQL(ctx, `mutation PullRequestAutoMergeDisable($prID: ID!) {
	disablePullRequestAutoMerge(input: {pullRequestId: $prID}) { clientMutationId }
}`, map[string]any{"prID": prId}, nil)
}

func (self *Client) ClosePullRequest(ctx context.Context, prId string) error {
	return self.graphQL(ctx, `mutation PullRequestClose($prID: ID!) {
	closePullRequest(input: {pullRequestId: $prID}) { clientMutationId }
}`, map[string]any{"prID": prId}, nil)
}

func (self *Client) ReopenPullRequest(ctx context.Context, prId string) error {
	return self.graphQL(ctx, `mutation PullRequestReopen($prID: ID!) {
	reopenPullRequest(input: {pullRequestId: $prID}) { clientMutationId }
}`, map[string]any{"prID": prId}, nil)
}

func (self *Client) MarkPullRequestReadyForReview(ctx context.Context, prId string) error {
	return self.graphQL(ctx, `mutation PullRequestReadyForReview($prID: ID!) {
	markPullRequestReadyForReview(input: {pullRequestId: $prID}) { clientMutationId }
}`, map[string]any{"prID": prId}, nil)
}

func (self *Client) ConvertPullRequestToDraft(ctx context.Context, prId string) error {
	return self.graphQL(ctx, `mutation ConvertPullRequestToDraft($prID: ID!) {
	convertPullRequestToDraft(input: {pullRequestId: $prID}) { clientMutationId }
}`, map[string]any{"prID": prId}, nil)
}

// UpdatePullRequestBranch brings the head branch up to date with the base
// branch, by merging the base branch into it or by rebasing it.
func (self *Client) UpdatePullRequestBranch(ctx context.Context, prId string, rebase bool) error {
	method := "MERGE"
	if rebase {
		method = "REBASE"
	}
	return self.graphQL(ctx, `mutation PullRequestUpdateBranch($input: UpdatePullRequestBranchInput!) {
	updatePullRequestBranch(input: $input) { clientMutationId }
}`, map[string]any{"input": map[string]any{"pullRequestId": prId, "updateMethod": method}}, nil)
}

// AddComment adds a comment to the pull request's conversation.
func (self *Client) AddComment(ctx context.Context, prId string, body string) error {
	return self.graphQL(ctx, `mutation CommentCreate($input: AddCommentInput!) {
	addComment(input: $input) { clientMutationId }
}`, map[string]any{"input": map[string]any{"subjectId": prId, "body": body}}, nil)
}

// Review events, as the GraphQL API spells them.
const (
	ReviewEventApprove        = "APPROVE"
	ReviewEventRequestChanges = "REQUEST_CHANGES"
	ReviewEventComment        = "COMMENT"
)

// SubmitReview reviews a pull request. If the viewer has a pending review,
// pendingReviewId identifies it, and it is that review, with the comments
// collected in it, that is submitted.
func (self *Client) SubmitReview(ctx context.Context, prId string, pendingReviewId string, event string, body string) error {
	if pendingReviewId != "" {
		input := map[string]any{"pullRequestReviewId": pendingReviewId, "event": event}
		if body != "" {
			input["body"] = body
		}
		return self.graphQL(ctx, `mutation PullRequestReviewSubmit($input: SubmitPullRequestReviewInput!) {
	submitPullRequestReview(input: $input) { clientMutationId }
}`, map[string]any{"input": input}, nil)
	}

	input := map[string]any{"pullRequestId": prId, "event": event}
	if body != "" {
		input["body"] = body
	}
	return self.graphQL(ctx, `mutation PullRequestReviewAdd($input: AddPullRequestReviewInput!) {
	addPullRequestReview(input: $input) { clientMutationId }
}`, map[string]any{"input": input}, nil)
}

// DeletePendingReview discards the viewer's pending review and the comments
// collected in it.
func (self *Client) DeletePendingReview(ctx context.Context, reviewId string) error {
	return self.graphQL(ctx, `mutation PullRequestReviewDelete($input: DeletePullRequestReviewInput!) {
	deletePullRequestReview(input: $input) { clientMutationId }
}`, map[string]any{"input": map[string]any{"pullRequestReviewId": reviewId}}, nil)
}

// DeleteBranch deletes a branch of the repository, e.g. a pull request's head
// branch once it has been merged.
func (self *Client) DeleteBranch(ctx context.Context, branch string) error {
	escaped := strings.Join(lo.Map(strings.Split(branch, "/"), func(segment string, _ int) string {
		return url.PathEscape(segment)
	}), "/")
	return self.delete(ctx, self.repoPath("git/refs/heads/%s", escaped))
}
