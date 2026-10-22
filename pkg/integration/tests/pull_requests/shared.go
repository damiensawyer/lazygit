package pull_requests

import (
	"fmt"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

func setupGithubRepo(shell *Shell) {
	shell.EmptyCommit("initial commit")
	shell.RunCommand([]string{"git", "remote", "add", "origin", "https://github.com/owner/repo.git"})
}

func at(day int) time.Time {
	return time.Date(2024, time.March, day, 10, 0, 0, 0, time.UTC)
}

func pullRequest(number int, title string, author string, state string, headBranch string) *models.GithubPullRequest {
	return &models.GithubPullRequest{
		NodeId:              fmt.Sprintf("PR_%d", number),
		Number:              number,
		Title:               title,
		State:               state,
		Url:                 fmt.Sprintf("https://github.com/owner/repo/pull/%d", number),
		Author:              author,
		BaseRefName:         "master",
		HeadRefName:         headBranch,
		HeadRefOid:          "0123456789abcdef0123456789abcdef01234567",
		HeadRepositoryName:  "repo",
		HeadRepositoryOwner: models.GithubRepositoryOwner{Login: "owner"},
		CreatedAt:           at(1),
		UpdatedAt:           at(2),
	}
}

// pullRequests returns open pull requests in the states worth telling apart,
// most recently updated first.
func pullRequests() []*models.GithubPullRequest {
	widget := pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")
	widget.ChecksState = "FAILURE"
	widget.ReviewDecision = "CHANGES_REQUESTED"
	widget.Additions = 40
	widget.Deletions = 3

	docs := pullRequest(11, "Document the frobnicator", "me", "DRAFT", "docs")
	docs.ChecksState = "PENDING"

	fix := pullRequest(10, "Fix the frobnicator", "hubot", "OPEN", "fix")
	fix.ChecksState = "SUCCESS"
	fix.ReviewDecision = "APPROVED"

	return []*models.GithubPullRequest{widget, docs, fix}
}

func widgetDetails() *models.GithubPullRequestDetails {
	return &models.GithubPullRequestDetails{
		Body:           "This adds a widget.\r\n\r\n<!-- template comment -->\r\nFixes #3.",
		Assignees:      []string{"me"},
		ReviewRequests: []string{"alice"},
		LatestReviews:  []*models.GithubReview{{Author: "bob", State: "CHANGES_REQUESTED", SubmittedAt: at(2)}},
		Timeline: []*models.GithubTimelineItem{
			{Author: "carol", Body: "Looks useful", CreatedAt: at(1)},
			{Author: "bob", Body: "Please add tests", CreatedAt: at(2), ReviewState: "CHANGES_REQUESTED"},
		},
		Checks: []*models.GithubCheck{
			{Name: "build", Workflow: "CI", Status: "completed", Conclusion: "success", RunId: 101, JobId: 1001},
			{Name: "test", Workflow: "CI", Status: "completed", Conclusion: "failure", IsRequired: true, RunId: 101, JobId: 1002},
		},
		CommitsCount:     2,
		MergeStateStatus: "BLOCKED",
		ReviewThreads: []*models.GithubReviewThread{{
			NodeId: "T_1", Path: "widget.go", Line: 7, DiffSide: "RIGHT",
			Comments: []*models.GithubReviewComment{{
				NodeId: "C_1", DatabaseId: 1, Author: "bob", Body: "Why a pointer?", CreatedAt: at(2),
				DiffHunk: "@@ -1,3 +1,7 @@\n package widget\n+\n+type Widget struct {\n+\tframe *Frame",
			}},
		}},
	}
}
