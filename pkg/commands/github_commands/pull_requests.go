package github_commands

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// The queries below ask for what the GitHub CLI asks for in `gh pr list` and
// `gh pr view` (cli/cli's api/query_builder.go), in the shape we show it in.

const pullRequestFields = `
fragment pr on PullRequest {
	id
	number
	title
	state
	isDraft
	url
	createdAt
	updatedAt
	author { login }
	baseRefName
	headRefName
	headRefOid
	headRepository { name }
	headRepositoryOwner { login }
	isCrossRepository
	maintainerCanModify
	reviewDecision
	additions
	deletions
	comments { totalCount }
	labels(first: 20) { nodes { name color } }
	commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }
}`

type pullRequestNode struct {
	Id        string    `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	IsDraft   bool      `json:"isDraft"`
	Url       string    `json:"url"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Author    *struct {
		Login string `json:"login"`
	} `json:"author"`
	BaseRefName    string `json:"baseRefName"`
	HeadRefName    string `json:"headRefName"`
	HeadRefOid     string `json:"headRefOid"`
	HeadRepository *struct {
		Name string `json:"name"`
	} `json:"headRepository"`
	HeadRepositoryOwner *struct {
		Login string `json:"login"`
	} `json:"headRepositoryOwner"`
	IsCrossRepository   bool   `json:"isCrossRepository"`
	MaintainerCanModify bool   `json:"maintainerCanModify"`
	ReviewDecision      string `json:"reviewDecision"`
	Additions           int    `json:"additions"`
	Deletions           int    `json:"deletions"`
	Comments            struct {
		TotalCount int `json:"totalCount"`
	} `json:"comments"`
	Labels struct {
		Nodes []models.GithubLabel `json:"nodes"`
	} `json:"labels"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

func login(actor *struct {
	Login string `json:"login"`
},
) string {
	if actor == nil {
		// e.g. a deleted account, which GitHub shows as "ghost"
		return "ghost"
	}
	return actor.Login
}

func (self *pullRequestNode) toModel() *models.GithubPullRequest {
	pr := &models.GithubPullRequest{
		NodeId:              self.Id,
		Number:              self.Number,
		Title:               self.Title,
		State:               lo.Ternary(self.IsDraft && self.State == "OPEN", "DRAFT", self.State),
		Url:                 self.Url,
		CreatedAt:           self.CreatedAt,
		UpdatedAt:           self.UpdatedAt,
		Author:              login(self.Author),
		BaseRefName:         self.BaseRefName,
		HeadRefName:         self.HeadRefName,
		HeadRefOid:          self.HeadRefOid,
		IsCrossRepository:   self.IsCrossRepository,
		MaintainerCanModify: self.MaintainerCanModify,
		ReviewDecision:      self.ReviewDecision,
		Additions:           self.Additions,
		Deletions:           self.Deletions,
		CommentsCount:       self.Comments.TotalCount,
		Labels:              self.Labels.Nodes,
	}
	if self.HeadRepository != nil {
		pr.HeadRepositoryName = self.HeadRepository.Name
	}
	if self.HeadRepositoryOwner != nil {
		pr.HeadRepositoryOwner = models.GithubRepositoryOwner{Login: self.HeadRepositoryOwner.Login}
	}
	if len(self.Commits.Nodes) > 0 && self.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
		pr.ChecksState = self.Commits.Nodes[0].Commit.StatusCheckRollup.State
	}
	return pr
}

// Pull request states for ListPullRequestsOptions.State.
const (
	PullRequestStateOpen   = "open"
	PullRequestStateClosed = "closed"
	PullRequestStateMerged = "merged"
	PullRequestStateAll    = "all"
)

// ListPullRequestsOptions narrows down which pull requests ListPullRequests
// returns. Zero values mean "don't filter by this", except for State, which
// defaults to open ones.
type ListPullRequestsOptions struct {
	State string
	// A login, or "@me" for the viewer.
	Author          string
	Assignee        string
	ReviewRequested string
	Labels          []string
	BaseBranch      string
	// Free text in GitHub's search syntax, e.g. "fix in:title".
	Search string
	Limit  int
}

// needsSearch reports whether the options can only be honoured by the search
// API. The search index lags behind a little, so a pull request that was just
// created only shows up in a listing of the repository's pull requests.
func (self ListPullRequestsOptions) needsSearch() bool {
	return self.Author != "" || self.Assignee != "" || self.ReviewRequested != "" ||
		len(self.Labels) > 0 || self.Search != ""
}

// searchQuery builds the search query for the options, like gh's
// SearchQueryBuild does.
func (self ListPullRequestsOptions) searchQuery(repo Repo) string {
	terms := []string{"repo:" + repo.FullName(), "is:pr", "sort:updated-desc"}
	switch self.State {
	case PullRequestStateClosed:
		terms = append(terms, "is:closed", "is:unmerged")
	case PullRequestStateMerged:
		terms = append(terms, "is:merged")
	case PullRequestStateAll:
	default:
		terms = append(terms, "is:open")
	}
	qualifier := func(name string, value string) {
		if value != "" {
			terms = append(terms, name+":"+quoteSearchValue(value))
		}
	}
	qualifier("author", self.Author)
	qualifier("assignee", self.Assignee)
	qualifier("review-requested", self.ReviewRequested)
	qualifier("base", self.BaseBranch)
	for _, label := range self.Labels {
		qualifier("label", label)
	}
	if self.Search != "" {
		terms = append(terms, self.Search)
	}
	return strings.Join(terms, " ")
}

func quoteSearchValue(value string) string {
	if strings.ContainsAny(value, " \t\"") {
		return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
	}
	return value
}

func (self ListPullRequestsOptions) states() []string {
	switch self.State {
	case PullRequestStateClosed:
		return []string{"CLOSED"}
	case PullRequestStateMerged:
		return []string{"MERGED"}
	case PullRequestStateAll:
		return []string{"OPEN", "CLOSED", "MERGED"}
	default:
		return []string{"OPEN"}
	}
}

const (
	// as many as `gh pr list` lists
	defaultPullRequestsLimit = 30
	// GitHub gives up on a page of many more pull requests of a big
	// repository, answering 502 when it takes too long
	pullRequestsPageSize = 50
)

// ListPullRequests returns the repository's pull requests that match the
// options, most recently updated first.
func (self *Client) ListPullRequests(ctx context.Context, opts ListPullRequestsOptions) ([]*models.GithubPullRequest, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = defaultPullRequestsLimit
	}

	type page struct {
		Nodes    []pullRequestNode `json:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
	}

	var query string
	variables := map[string]any{}
	if opts.needsSearch() {
		query = pullRequestFields + `
query PullRequestSearch($q: String!, $limit: Int!, $endCursor: String) {
	search(query: $q, type: ISSUE, first: $limit, after: $endCursor) {
		nodes { ...pr }
		pageInfo { hasNextPage endCursor }
	}
}`
		variables["q"] = opts.searchQuery(self.repo)
	} else {
		query = pullRequestFields + `
query PullRequestList($owner: String!, $repo: String!, $states: [PullRequestState!], $baseBranch: String, $limit: Int!, $endCursor: String) {
	repository(owner: $owner, name: $repo) {
		pullRequests(states: $states, baseRefName: $baseBranch, first: $limit, after: $endCursor, orderBy: {field: UPDATED_AT, direction: DESC}) {
			nodes { ...pr }
			pageInfo { hasNextPage endCursor }
		}
	}
}`
		variables["owner"] = self.repo.Owner
		variables["repo"] = self.repo.Name
		variables["states"] = opts.states()
		if opts.BaseBranch != "" {
			variables["baseBranch"] = opts.BaseBranch
		}
	}

	prs := []*models.GithubPullRequest{}
	seen := map[int]bool{}
	for {
		variables["limit"] = min(limit-len(prs), pullRequestsPageSize)

		var data struct {
			Search     *page `json:"search"`
			Repository *struct {
				PullRequests page `json:"pullRequests"`
			} `json:"repository"`
		}
		if err := self.graphQL(ctx, query, variables, &data); err != nil {
			return nil, err
		}
		var result *page
		switch {
		case data.Search != nil:
			result = data.Search
		case data.Repository != nil:
			result = &data.Repository.PullRequests
		default:
			return nil, fmt.Errorf("repository %s not found", self.repo.FullName())
		}

		for _, node := range result.Nodes {
			// Search results include issues as empty nodes, since the
			// fragment only applies to pull requests; and a pull request
			// updated while paging can show up on two pages.
			if node.Number == 0 || seen[node.Number] {
				continue
			}
			seen[node.Number] = true
			prs = append(prs, node.toModel())
		}

		if !result.PageInfo.HasNextPage || len(prs) >= limit {
			return prs, nil
		}
		variables["endCursor"] = result.PageInfo.EndCursor
	}
}

const pullRequestDetailsQuery = pullRequestFields + `
query PullRequestDetails($owner: String!, $repo: String!, $number: Int!) {
	viewer { login }
	repository(owner: $owner, name: $repo) {
		defaultBranchRef { name }
		mergeCommitAllowed
		squashMergeAllowed
		rebaseMergeAllowed
		autoMergeAllowed
		deleteBranchOnMerge
		viewerDefaultMergeMethod
		viewerCanAdminister
		viewerPermission
		pullRequest(number: $number) {
			...pr
			body
			viewerCanUpdate
			mergeable
			mergeStateStatus
			mergedAt
			closedAt
			baseRefOid
			mergedBy { login }
			autoMergeRequest { mergeMethod enabledBy { login } }
			milestone { title }
			assignees(first: 20) { nodes { login } }
			reviewRequests(first: 50) {
				nodes {
					requestedReviewer {
						__typename
						... on User { login }
						... on Bot { login }
						... on Mannequin { login }
						... on Team { organization { login } slug }
					}
				}
			}
			latestReviews(first: 50) { nodes { author { login } state submittedAt body url } }
			reviews(first: 100) { nodes { id author { login } state submittedAt body url } }
			pendingReviews: reviews(first: 1, states: PENDING) { nodes { id comments { totalCount } } }
			conversation: comments(first: 100) { nodes { author { login } body createdAt url } }
			files(first: 100) { nodes { path additions deletions changeType } }
			allCommits: commits { totalCount }
			reviewThreads(first: 100) {
				nodes {
					id
					path
					isResolved
					isOutdated
					line
					originalLine
					diffSide
					comments(first: 50) {
						nodes { id databaseId author { login } body createdAt url diffHunk state }
					}
				}
			}
			checks: commits(last: 1) {
				nodes {
					commit {
						statusCheckRollup {
							contexts(first: 100) {
								nodes {
									__typename
									... on StatusContext {
										context
										state
										targetUrl
										createdAt
										isRequired(pullRequestNumber: $number)
									}
									... on CheckRun {
										name
										status
										conclusion
										detailsUrl
										databaseId
										startedAt
										isRequired(pullRequestNumber: $number)
										checkSuite { workflowRun { databaseId event workflow { name } } }
									}
								}
							}
						}
					}
				}
			}
		}
	}
}`

type actor = struct {
	Login string `json:"login"`
}

type pullRequestDetailsNode struct {
	pullRequestNode
	Body             string    `json:"body"`
	ViewerCanUpdate  bool      `json:"viewerCanUpdate"`
	Mergeable        string    `json:"mergeable"`
	MergeStateStatus string    `json:"mergeStateStatus"`
	MergedAt         time.Time `json:"mergedAt"`
	ClosedAt         time.Time `json:"closedAt"`
	BaseRefOid       string    `json:"baseRefOid"`
	MergedBy         *actor    `json:"mergedBy"`
	AutoMergeRequest *struct {
		MergeMethod string `json:"mergeMethod"`
		EnabledBy   *actor `json:"enabledBy"`
	} `json:"autoMergeRequest"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Assignees struct {
		Nodes []actor `json:"nodes"`
	} `json:"assignees"`
	ReviewRequests struct {
		Nodes []struct {
			RequestedReviewer *struct {
				Typename     string `json:"__typename"`
				Login        string `json:"login"`
				Slug         string `json:"slug"`
				Organization *actor `json:"organization"`
			} `json:"requestedReviewer"`
		} `json:"nodes"`
	} `json:"reviewRequests"`
	LatestReviews struct {
		Nodes []reviewNode `json:"nodes"`
	} `json:"latestReviews"`
	Reviews struct {
		Nodes []reviewNode `json:"nodes"`
	} `json:"reviews"`
	PendingReviews struct {
		Nodes []struct {
			Id       string `json:"id"`
			Comments struct {
				TotalCount int `json:"totalCount"`
			} `json:"comments"`
		} `json:"nodes"`
	} `json:"pendingReviews"`
	IssueComments struct {
		Nodes []struct {
			Author    *actor    `json:"author"`
			Body      string    `json:"body"`
			CreatedAt time.Time `json:"createdAt"`
			Url       string    `json:"url"`
		} `json:"nodes"`
	} `json:"conversation"`
	Files struct {
		Nodes []*models.GithubPullRequestFile `json:"nodes"`
	} `json:"files"`
	AllCommits struct {
		TotalCount int `json:"totalCount"`
	} `json:"allCommits"`
	ReviewThreads struct {
		Nodes []struct {
			Id           string `json:"id"`
			Path         string `json:"path"`
			IsResolved   bool   `json:"isResolved"`
			IsOutdated   bool   `json:"isOutdated"`
			Line         int    `json:"line"`
			OriginalLine int    `json:"originalLine"`
			DiffSide     string `json:"diffSide"`
			Comments     struct {
				Nodes []struct {
					Id         string    `json:"id"`
					DatabaseId int64     `json:"databaseId"`
					Author     *actor    `json:"author"`
					Body       string    `json:"body"`
					CreatedAt  time.Time `json:"createdAt"`
					Url        string    `json:"url"`
					DiffHunk   string    `json:"diffHunk"`
					State      string    `json:"state"`
				} `json:"nodes"`
			} `json:"comments"`
		} `json:"nodes"`
	} `json:"reviewThreads"`
	Checks struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					Contexts struct {
						Nodes []checkContextNode `json:"nodes"`
					} `json:"contexts"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"checks"`
}

type reviewNode struct {
	Id          string    `json:"id"`
	Author      *actor    `json:"author"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submittedAt"`
	Body        string    `json:"body"`
	Url         string    `json:"url"`
}

func (self reviewNode) toModel() *models.GithubReview {
	return &models.GithubReview{
		NodeId:      self.Id,
		Author:      login(self.Author),
		State:       self.State,
		Body:        self.Body,
		SubmittedAt: self.SubmittedAt,
		Url:         self.Url,
	}
}

type checkContextNode struct {
	Typename string `json:"__typename"`
	// StatusContext
	Context   string    `json:"context"`
	State     string    `json:"state"`
	TargetUrl string    `json:"targetUrl"`
	CreatedAt time.Time `json:"createdAt"`
	// CheckRun
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	DetailsUrl string    `json:"detailsUrl"`
	DatabaseId int64     `json:"databaseId"`
	StartedAt  time.Time `json:"startedAt"`
	CheckSuite *struct {
		WorkflowRun *workflowRunNode `json:"workflowRun"`
	} `json:"checkSuite"`
	IsRequired bool `json:"isRequired"`
}

type workflowRunNode struct {
	DatabaseId int64  `json:"databaseId"`
	Event      string `json:"event"`
	Workflow   struct {
		Name string `json:"name"`
	} `json:"workflow"`
}

func (self checkContextNode) toModel() *models.GithubCheck {
	if self.Typename == "StatusContext" {
		// A commit status has a single state; express it as the status and
		// conclusion of a check run.
		check := &models.GithubCheck{Name: self.Context, Url: self.TargetUrl, IsRequired: self.IsRequired}
		switch self.State {
		case "SUCCESS":
			check.Status, check.Conclusion = models.GithubStatusCompleted, models.GithubConclusionSuccess
		case "FAILURE", "ERROR":
			check.Status, check.Conclusion = models.GithubStatusCompleted, models.GithubConclusionFailure
		default:
			check.Status = models.GithubStatusPending
		}
		return check
	}

	check := &models.GithubCheck{
		Name:       self.Name,
		Status:     strings.ToLower(self.Status),
		Conclusion: strings.ToLower(self.Conclusion),
		Url:        self.DetailsUrl,
		IsRequired: self.IsRequired,
	}
	if self.CheckSuite != nil && self.CheckSuite.WorkflowRun != nil {
		check.Workflow = self.CheckSuite.WorkflowRun.Workflow.Name
		check.RunId = self.CheckSuite.WorkflowRun.DatabaseId
		check.JobId = self.DatabaseId
	}
	return check
}

// GetPullRequest returns a pull request along with its details.
func (self *Client) GetPullRequest(ctx context.Context, number int) (*models.GithubPullRequest, *models.GithubPullRequestDetails, error) {
	var data struct {
		Viewer     actor `json:"viewer"`
		Repository *struct {
			DefaultBranchRef *struct {
				Name string `json:"name"`
			} `json:"defaultBranchRef"`
			MergeCommitAllowed       bool                    `json:"mergeCommitAllowed"`
			SquashMergeAllowed       bool                    `json:"squashMergeAllowed"`
			RebaseMergeAllowed       bool                    `json:"rebaseMergeAllowed"`
			AutoMergeAllowed         bool                    `json:"autoMergeAllowed"`
			DeleteBranchOnMerge      bool                    `json:"deleteBranchOnMerge"`
			ViewerDefaultMergeMethod string                  `json:"viewerDefaultMergeMethod"`
			ViewerCanAdminister      bool                    `json:"viewerCanAdminister"`
			ViewerPermission         string                  `json:"viewerPermission"`
			PullRequest              *pullRequestDetailsNode `json:"pullRequest"`
		} `json:"repository"`
	}
	err := self.graphQL(ctx, pullRequestDetailsQuery, map[string]any{
		"owner":  self.repo.Owner,
		"repo":   self.repo.Name,
		"number": number,
	}, &data)
	if err != nil {
		return nil, nil, err
	}
	if data.Repository == nil || data.Repository.PullRequest == nil {
		return nil, nil, fmt.Errorf("pull request #%d not found in %s", number, self.repo.FullName())
	}

	node := data.Repository.PullRequest
	repo := data.Repository
	details := &models.GithubPullRequestDetails{
		Body:             node.Body,
		Mergeable:        node.Mergeable,
		MergeStateStatus: node.MergeStateStatus,
		MergedAt:         node.MergedAt,
		ClosedAt:         node.ClosedAt,
		BaseRefOid:       node.BaseRefOid,
		CommitsCount:     node.AllCommits.TotalCount,
		Files:            node.Files.Nodes,
		ViewerLogin:      data.Viewer.Login,
		ViewerCanEdit:    node.ViewerCanUpdate,
		Assignees:        lo.Map(node.Assignees.Nodes, func(a actor, _ int) string { return a.Login }),
		LatestReviews: lo.FilterMap(node.LatestReviews.Nodes, func(review reviewNode, _ int) (*models.GithubReview, bool) {
			return review.toModel(), review.State != "PENDING"
		}),
		Repository: models.GithubRepositorySettings{
			MergeCommitAllowed:  repo.MergeCommitAllowed,
			SquashMergeAllowed:  repo.SquashMergeAllowed,
			RebaseMergeAllowed:  repo.RebaseMergeAllowed,
			AutoMergeAllowed:    repo.AutoMergeAllowed,
			DeleteBranchOnMerge: repo.DeleteBranchOnMerge,
			ViewerDefaultMethod: repo.ViewerDefaultMergeMethod,
			ViewerCanAdminister: repo.ViewerCanAdminister,
			ViewerPermission:    repo.ViewerPermission,
		},
	}
	if repo.DefaultBranchRef != nil {
		details.Repository.DefaultBranch = repo.DefaultBranchRef.Name
	}
	if node.MergedBy != nil {
		details.MergedBy = node.MergedBy.Login
	}
	if node.AutoMergeRequest != nil {
		details.AutoMerge = &models.GithubAutoMerge{
			MergeMethod: node.AutoMergeRequest.MergeMethod,
			EnabledBy:   login(node.AutoMergeRequest.EnabledBy),
		}
	}
	if node.Milestone != nil {
		details.Milestone = node.Milestone.Title
	}
	for _, request := range node.ReviewRequests.Nodes {
		reviewer := request.RequestedReviewer
		switch {
		case reviewer == nil:
		case reviewer.Typename == "Team" && reviewer.Organization != nil:
			details.ReviewRequests = append(details.ReviewRequests, reviewer.Organization.Login+"/"+reviewer.Slug)
		default:
			details.ReviewRequests = append(details.ReviewRequests, reviewer.Login)
		}
	}
	if len(node.PendingReviews.Nodes) > 0 {
		pending := node.PendingReviews.Nodes[0]
		details.PendingReview = &models.GithubPendingReview{NodeId: pending.Id, CommentsCount: pending.Comments.TotalCount}
	}

	for _, comment := range node.IssueComments.Nodes {
		details.Timeline = append(details.Timeline, &models.GithubTimelineItem{
			Author:    login(comment.Author),
			Body:      comment.Body,
			CreatedAt: comment.CreatedAt,
			Url:       comment.Url,
		})
	}
	for _, review := range node.Reviews.Nodes {
		// A review that only commented on lines has nothing to say of its
		// own; its comments are shown with the threads they started.
		if review.State == "PENDING" || (review.State == "COMMENTED" && strings.TrimSpace(review.Body) == "") {
			continue
		}
		details.Timeline = append(details.Timeline, &models.GithubTimelineItem{
			Author:      login(review.Author),
			Body:        review.Body,
			CreatedAt:   review.SubmittedAt,
			Url:         review.Url,
			ReviewState: review.State,
		})
	}
	slices.SortStableFunc(details.Timeline, func(a, b *models.GithubTimelineItem) int {
		return a.CreatedAt.Compare(b.CreatedAt)
	})

	for _, threadNode := range node.ReviewThreads.Nodes {
		thread := &models.GithubReviewThread{
			NodeId:       threadNode.Id,
			Path:         threadNode.Path,
			IsResolved:   threadNode.IsResolved,
			IsOutdated:   threadNode.IsOutdated,
			Line:         threadNode.Line,
			OriginalLine: threadNode.OriginalLine,
			DiffSide:     threadNode.DiffSide,
		}
		for _, comment := range threadNode.Comments.Nodes {
			thread.Comments = append(thread.Comments, &models.GithubReviewComment{
				NodeId:     comment.Id,
				DatabaseId: comment.DatabaseId,
				Author:     login(comment.Author),
				Body:       comment.Body,
				CreatedAt:  comment.CreatedAt,
				Url:        comment.Url,
				DiffHunk:   comment.DiffHunk,
				IsPending:  comment.State == "PENDING",
			})
		}
		details.ReviewThreads = append(details.ReviewThreads, thread)
	}

	if len(node.Checks.Nodes) > 0 && node.Checks.Nodes[0].Commit.StatusCheckRollup != nil {
		details.Checks = lo.Map(eliminateDuplicateChecks(node.Checks.Nodes[0].Commit.StatusCheckRollup.Contexts.Nodes),
			func(context checkContextNode, _ int) *models.GithubCheck { return context.toModel() })
	}

	return node.toModel(), details, nil
}

// eliminateDuplicateChecks keeps only the latest of the checks that ran more
// than once, e.g. because a workflow was triggered again, the way `gh pr
// checks` does: the same check of the same workflow for the same event, or a
// commit status of the same context.
func eliminateDuplicateChecks(contexts []checkContextNode) []checkContextNode {
	key := func(context checkContextNode) string {
		if context.Typename == "StatusContext" {
			return "status/" + context.Context
		}
		workflow, event := "", ""
		if context.CheckSuite != nil && context.CheckSuite.WorkflowRun != nil {
			workflow = context.CheckSuite.WorkflowRun.Workflow.Name
			event = context.CheckSuite.WorkflowRun.Event
		}
		return "check/" + context.Name + "/" + workflow + "/" + event
	}
	started := func(context checkContextNode) time.Time {
		return lo.Ternary(context.Typename == "StatusContext", context.CreatedAt, context.StartedAt)
	}

	latest := map[string]int{}
	for i, context := range contexts {
		k := key(context)
		if j, seen := latest[k]; !seen || started(context).After(started(contexts[j])) {
			latest[k] = i
		}
	}
	return lo.Filter(contexts, func(context checkContextNode, i int) bool {
		return latest[key(context)] == i
	})
}
