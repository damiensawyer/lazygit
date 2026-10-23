package github_commands

import (
	"context"
	"errors"
)

// Commenting on lines of a pull request's diff. The comments are collected in
// the viewer's pending review, the way GitHub's web UI does it with "Start a
// review", and are published when the review is submitted.

// StartPendingReview starts a review of a pull request that the viewer adds
// comments to until they submit it, and returns its ID.
func (self *Client) StartPendingReview(ctx context.Context, prId string) (string, error) {
	var data struct {
		AddPullRequestReview struct {
			PullRequestReview struct {
				Id string `json:"id"`
			} `json:"pullRequestReview"`
		} `json:"addPullRequestReview"`
	}
	err := self.graphQL(ctx, `mutation PullRequestReviewStart($input: AddPullRequestReviewInput!) {
	addPullRequestReview(input: $input) { pullRequestReview { id } }
}`, map[string]any{"input": map[string]any{"pullRequestId": prId}}, &data)
	if err != nil {
		return "", err
	}
	id := data.AddPullRequestReview.PullRequestReview.Id
	if id == "" {
		return "", errors.New("GitHub didn't start a review")
	}
	return id, nil
}

// ReviewComment is a comment on a line, or a range of lines, of a pull
// request's diff.
type ReviewComment struct {
	Path string
	// "LEFT" for a line of the old version of the file, "RIGHT" for the new
	Side string
	Line int
	// for a comment on a range of lines, its first line, otherwise 0
	StartLine int
	StartSide string
	Body      string
}

// AddReviewComment adds a comment to a pending review, starting a thread.
func (self *Client) AddReviewComment(ctx context.Context, reviewId string, comment ReviewComment) error {
	input := map[string]any{
		"pullRequestReviewId": reviewId,
		"path":                comment.Path,
		"side":                comment.Side,
		"line":                comment.Line,
		"body":                comment.Body,
	}
	if comment.StartLine != 0 && comment.StartLine != comment.Line {
		input["startLine"] = comment.StartLine
		input["startSide"] = comment.StartSide
	}
	return self.graphQL(ctx, `mutation PullRequestReviewThreadAdd($input: AddPullRequestReviewThreadInput!) {
	addPullRequestReviewThread(input: $input) { thread { id } }
}`, map[string]any{"input": input}, nil)
}

// ReplyToReviewThread adds a reply to a review thread. With a pending review,
// the reply goes into it, to be published along with it.
func (self *Client) ReplyToReviewThread(ctx context.Context, threadId string, pendingReviewId string, body string) error {
	input := map[string]any{"pullRequestReviewThreadId": threadId, "body": body}
	if pendingReviewId != "" {
		input["pullRequestReviewId"] = pendingReviewId
	}
	return self.graphQL(ctx, `mutation PullRequestReviewThreadReply($input: AddPullRequestReviewThreadReplyInput!) {
	addPullRequestReviewThreadReply(input: $input) { comment { id } }
}`, map[string]any{"input": input}, nil)
}

func (self *Client) ResolveReviewThread(ctx context.Context, threadId string) error {
	return self.graphQL(ctx, `mutation PullRequestReviewThreadResolve($threadId: ID!) {
	resolveReviewThread(input: {threadId: $threadId}) { thread { id } }
}`, map[string]any{"threadId": threadId}, nil)
}

func (self *Client) UnresolveReviewThread(ctx context.Context, threadId string) error {
	return self.graphQL(ctx, `mutation PullRequestReviewThreadUnresolve($threadId: ID!) {
	unresolveReviewThread(input: {threadId: $threadId}) { thread { id } }
}`, map[string]any{"threadId": threadId}, nil)
}
