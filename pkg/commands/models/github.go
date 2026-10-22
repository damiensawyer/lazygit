package models

import (
	"fmt"
	"strconv"
	"time"
)

type GithubPullRequest struct {
	HeadRefName         string                `json:"headRefName"`
	Number              int                   `json:"number"`
	Title               string                `json:"title"`
	State               string                `json:"state"` // "MERGED", "OPEN", "CLOSED", "DRAFT"
	ChecksState         string                `json:"checksState"`
	Url                 string                `json:"url"`
	HeadRepositoryOwner GithubRepositoryOwner `json:"headRepositoryOwner"`

	// The fields below are only filled in for the pull requests that the
	// pull requests panel lists, not for those found for the local branches.

	// The GraphQL node ID, which mutations identify the pull request by.
	NodeId string `json:"id"`
	Author string `json:"author"`
	// e.g. "master"
	BaseRefName string `json:"baseRefName"`
	// The commit the head branch points at.
	HeadRefOid string `json:"headRefOid"`
	// The name of the repository the head branch is in, which differs from
	// the base repository's for a pull request from a fork; empty if that
	// repository has been deleted.
	HeadRepositoryName  string `json:"headRepositoryName"`
	IsCrossRepository   bool   `json:"isCrossRepository"`
	MaintainerCanModify bool   `json:"maintainerCanModify"`
	// "APPROVED", "CHANGES_REQUESTED", "REVIEW_REQUIRED", or empty if the
	// repository doesn't require reviews.
	ReviewDecision string        `json:"reviewDecision"`
	Labels         []GithubLabel `json:"labels"`
	Additions      int           `json:"additions"`
	Deletions      int           `json:"deletions"`
	CommentsCount  int           `json:"commentsCount"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type GithubLabel struct {
	Name string `json:"name"`
	// a hex color without the leading "#", e.g. "d73a4a"
	Color string `json:"color"`
}

func (pr *GithubPullRequest) ID() string {
	return strconv.Itoa(pr.Number)
}

func (pr *GithubPullRequest) Description() string {
	return fmt.Sprintf("#%d %s", pr.Number, pr.Title)
}

func (pr *GithubPullRequest) IsOpen() bool {
	return pr.State == "OPEN" || pr.State == "DRAFT"
}

func (pr *GithubPullRequest) IsDraft() bool {
	return pr.State == "DRAFT"
}

// HeadLabel is how GitHub names a pull request's head branch: prefixed with
// its owner if it is in a fork, e.g. "octocat:my-feature".
func (pr *GithubPullRequest) HeadLabel() string {
	if pr.IsCrossRepository {
		return pr.HeadRepositoryOwner.Login + ":" + pr.HeadRefName
	}
	return pr.HeadRefName
}

func (pr *GithubPullRequest) UserName() string {
	// e.g. 'jesseduffield'
	return pr.HeadRepositoryOwner.Login
}

func (pr *GithubPullRequest) BranchName() string {
	// e.g. 'feature/my-feature'
	return pr.HeadRefName
}

type GithubRepositoryOwner struct {
	Login string `json:"login"`
}
