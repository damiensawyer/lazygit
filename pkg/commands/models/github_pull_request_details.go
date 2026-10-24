package models

import (
	"fmt"
	"strings"
	"time"
)

// GithubPullRequestDetails is everything about a pull request beyond what the
// pull requests panel lists, loaded when one is selected.
type GithubPullRequestDetails struct {
	Body      string
	Assignees []string
	Milestone string
	// The users and teams whose review has been requested and who haven't
	// reviewed yet; teams are written as "org/team".
	ReviewRequests []string
	// The latest review of each reviewer.
	LatestReviews []*GithubReview
	// The issue comments and the reviews, oldest first.
	Timeline     []*GithubTimelineItem
	Checks       []*GithubCheck
	Files        []*GithubPullRequestFile
	CommitsCount int
	// "MERGEABLE", "CONFLICTING", or "UNKNOWN" while GitHub works it out.
	Mergeable string
	// e.g. "CLEAN", "BLOCKED", "BEHIND", "DIRTY", "UNSTABLE"; see
	// https://docs.github.com/en/graphql/reference/enums#mergestatestatus
	MergeStateStatus string
	// nil unless auto-merge has been enabled
	AutoMerge     *GithubAutoMerge
	MergedBy      string
	MergedAt      time.Time
	ClosedAt      time.Time
	BaseRefOid    string
	ReviewThreads []*GithubReviewThread
	// The review the viewer has started but not submitted yet, if any. Only
	// its author can see a pending review.
	PendingReview *GithubPendingReview
	ViewerLogin   string
	ViewerCanEdit bool

	Repository GithubRepositorySettings
}

// GithubRepositorySettings are the settings of the base repository that
// matter for acting on its pull requests.
type GithubRepositorySettings struct {
	DefaultBranch       string
	MergeCommitAllowed  bool
	SquashMergeAllowed  bool
	RebaseMergeAllowed  bool
	AutoMergeAllowed    bool
	DeleteBranchOnMerge bool
	ViewerDefaultMethod string // "MERGE", "SQUASH", or "REBASE"
	ViewerCanAdminister bool
	ViewerPermission    string // e.g. "ADMIN", "WRITE", "READ"
}

type GithubAutoMerge struct {
	// "MERGE", "SQUASH", or "REBASE"
	MergeMethod string
	EnabledBy   string
}

type GithubReview struct {
	NodeId string
	Author string
	// "APPROVED", "CHANGES_REQUESTED", "COMMENTED", "DISMISSED", or "PENDING"
	State       string
	Body        string
	SubmittedAt time.Time
	Url         string
}

// GithubTimelineItem is an entry in a pull request's conversation: a comment,
// or a review, which may have a comment of its own.
type GithubTimelineItem struct {
	Author    string
	Body      string
	CreatedAt time.Time
	Url       string
	// empty for a comment, otherwise the review's state
	ReviewState string
}

// GithubCheck is a check run (most often a GitHub Actions job) or a commit
// status reported for the pull request's head commit.
type GithubCheck struct {
	Name string
	// the name of the GitHub Actions workflow the check is a job of, if any
	Workflow string
	// a check run's status and conclusion, lowercased to match the REST API's
	// spelling; for a commit status, the equivalent of its state
	Status     string
	Conclusion string
	Url        string
	IsRequired bool
	// for a GitHub Actions job, the IDs of the workflow run and of the job
	RunId int64
	JobId int64
}

func (self *GithubCheck) State() GithubActionsState {
	return githubActionsState(self.Status, self.Conclusion)
}

// DisplayName is the check's name the way GitHub's checks list shows it:
// prefixed with its workflow's name.
func (self *GithubCheck) DisplayName() string {
	if self.Workflow == "" {
		return self.Name
	}
	return self.Workflow + " / " + self.Name
}

type GithubPullRequestFile struct {
	Path      string
	Additions int
	Deletions int
	// e.g. "ADDED", "MODIFIED", "DELETED", "RENAMED"
	ChangeType string
}

// GithubReviewThread is a conversation about a line, or range of lines, of the
// pull request's diff.
type GithubReviewThread struct {
	NodeId     string
	Path       string
	IsResolved bool
	// set once the lines it is about have changed since it was started
	IsOutdated bool
	// the line in the current diff, or 0 if it is outdated
	Line int
	// the line in the diff it was started on
	OriginalLine int
	// "LEFT" for a line of the old version of the file, "RIGHT" for the new
	DiffSide string
	Comments []*GithubReviewComment
}

type GithubReviewComment struct {
	NodeId     string
	DatabaseId int64
	Author     string
	Body       string
	CreatedAt  time.Time
	Url        string
	// the diff around the line the thread is about, ending with that line
	DiffHunk string
	// set for a comment of the viewer's pending review
	IsPending bool
}

// GithubPendingReview is a review that its author has started, collecting
// comments in it, but hasn't submitted yet.
type GithubPendingReview struct {
	NodeId        string
	CommentsCount int
}

// CheckCounts counts the checks by how they are doing.
type CheckCounts struct {
	Passed, Failed, Pending, Skipped int
}

func (self *GithubPullRequestDetails) CheckCounts() CheckCounts {
	var counts CheckCounts
	for _, check := range self.Checks {
		switch state := check.State(); {
		case state.IsActive():
			counts.Pending++
		case state.IsFailed():
			counts.Failed++
		case state == GithubActionsStateSkipped || state == GithubActionsStateNeutral:
			counts.Skipped++
		default:
			counts.Passed++
		}
	}
	return counts
}

// UnresolvedThreads returns the review threads that haven't been resolved.
func (self *GithubPullRequestDetails) UnresolvedThreads() []*GithubReviewThread {
	threads := []*GithubReviewThread{}
	for _, thread := range self.ReviewThreads {
		if !thread.IsResolved {
			threads = append(threads, thread)
		}
	}
	return threads
}

// ThreadsForPath returns the review threads about a file.
func (self *GithubPullRequestDetails) ThreadsForPath(path string) []*GithubReviewThread {
	threads := []*GithubReviewThread{}
	for _, thread := range self.ReviewThreads {
		if thread.Path == path {
			threads = append(threads, thread)
		}
	}
	return threads
}

// ReviewRequestedFrom reports whether a review of the pull request has been
// requested from a user, case-insensitively as logins are.
func (self *GithubPullRequestDetails) ReviewRequestedFrom(login string) bool {
	for _, reviewer := range self.ReviewRequests {
		if strings.EqualFold(reviewer, login) {
			return true
		}
	}
	return false
}

// GithubPullRequestRef is the local ref that a pull request's head is fetched
// into for looking at its commits: refs/pull/<number>/head, which is what
// GitHub calls it too.
type GithubPullRequestRef struct {
	Number int
	Title  string
}

var _ Ref = GithubPullRequestRef{}

func (self GithubPullRequestRef) FullRefName() string {
	return fmt.Sprintf("refs/pull/%d/head", self.Number)
}

func (self GithubPullRequestRef) RefName() string {
	return self.FullRefName()
}

func (self GithubPullRequestRef) ShortRefName() string {
	return fmt.Sprintf("#%d", self.Number)
}

func (self GithubPullRequestRef) ParentRefName() string {
	return self.FullRefName() + "^"
}

func (self GithubPullRequestRef) Description() string {
	return fmt.Sprintf("#%d %s", self.Number, self.Title)
}

// GithubPullRequestChangesRef is a pull request's head as the changes the pull
// request makes: diffed against MergeBase, the commit where it branched off its
// base branch, as GitHub's "Files changed" tab does.
type GithubPullRequestChangesRef struct {
	GithubPullRequestRef
	MergeBase string
}

var _ Ref = GithubPullRequestChangesRef{}

func (self GithubPullRequestChangesRef) ParentRefName() string {
	return self.MergeBase
}
