package components

import (
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// The fake's GraphQL API answers the queries lazygit makes about pull
// requests, telling them apart by their operation names, from the pull
// requests in FakeGithub.PullRequests and their details in
// FakeGithub.PullRequestDetails. Mutations succeed but change nothing.

var graphQLOperationName = regexp.MustCompile(`(?m)^\s*(?:query|mutation)\s+(\w+)`)

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

func parseGraphQLRequest(body []byte) (graphQLRequest, string) {
	var request graphQLRequest
	_ = json.Unmarshal(body, &request)
	operation := ""
	if match := graphQLOperationName.FindStringSubmatch(request.Query); match != nil {
		operation = match[1]
	}
	return request, operation
}

// describeGraphQLRequest describes a GraphQL request for FakeGithub.Requests
// by its operation name and variables, since the query text is long and
// doesn't say much.
func describeGraphQLRequest(body []byte) string {
	request, operation := parseGraphQLRequest(body)
	description := "POST /graphql " + operation
	if len(request.Variables) > 0 {
		variables, _ := json.Marshal(request.Variables)
		description += " " + string(variables)
	}
	return description
}

func (self *FakeGithub) serveGraphQL(w http.ResponseWriter, body []byte) {
	request, operation := parseGraphQLRequest(body)
	if response, found := self.Errors["POST /graphql "+operation]; found {
		writeJSON(w, map[string]any{"errors": []map[string]string{{"message": response.Message}}})
		return
	}

	switch operation {
	case "PullRequestList":
		states, _ := request.Variables["states"].([]any)
		prs := lo.Filter(self.PullRequests, func(pr *models.GithubPullRequest, _ int) bool {
			state := lo.Ternary(pr.State == "DRAFT", "OPEN", pr.State)
			return slices.Contains(states, any(state))
		})
		writeJSON(w, map[string]any{"data": map[string]any{
			"repository": map[string]any{"pullRequests": pullRequestsPage(prs)},
		}})
	case "PullRequestSearch":
		q, _ := request.Variables["q"].(string)
		prs := lo.Filter(self.PullRequests, func(pr *models.GithubPullRequest, _ int) bool {
			return matchesSearch(pr, q)
		})
		writeJSON(w, map[string]any{"data": map[string]any{"search": pullRequestsPage(prs)}})
	case "PullRequestDetails":
		number := int(request.Variables["number"].(float64))
		pr, found := lo.Find(self.PullRequests, func(pr *models.GithubPullRequest) bool { return pr.Number == number })
		if !found {
			writeJSON(w, map[string]any{"data": map[string]any{
				"viewer": map[string]any{"login": "me"}, "repository": map[string]any{"pullRequest": nil},
			}})
			return
		}
		writeJSON(w, map[string]any{"data": self.pullRequestDetailsData(pr)})
	default:
		writeJSON(w, map[string]any{"data": map[string]any{}})
	}
}

// matchesSearch understands just enough of GitHub's search syntax for the
// queries lazygit builds: the state, and the author.
func matchesSearch(pr *models.GithubPullRequest, q string) bool {
	for _, term := range strings.Fields(q) {
		switch {
		case term == "is:open" && !pr.IsOpen():
			return false
		case term == "is:merged" && pr.State != "MERGED":
			return false
		case term == "is:closed" && pr.State != "CLOSED" && pr.State != "MERGED":
			return false
		case term == "is:unmerged" && pr.State == "MERGED":
			return false
		case strings.HasPrefix(term, "author:"):
			author := strings.TrimPrefix(term, "author:")
			if author == "@me" {
				author = "me"
			}
			if pr.Author != author {
				return false
			}
		}
	}
	return true
}

func pullRequestsPage(prs []*models.GithubPullRequest) map[string]any {
	return map[string]any{
		"nodes":    lo.Map(prs, func(pr *models.GithubPullRequest, _ int) map[string]any { return pullRequestNode(pr) }),
		"pageInfo": map[string]any{"hasNextPage": false},
	}
}

func login(name string) map[string]any {
	if name == "" {
		return nil
	}
	return map[string]any{"login": name}
}

func pullRequestNode(pr *models.GithubPullRequest) map[string]any {
	state := lo.Ternary(pr.State == "DRAFT", "OPEN", pr.State)
	node := map[string]any{
		"id":                  lo.CoalesceOrEmpty(pr.NodeId, "PR_"+pr.ID()),
		"number":              pr.Number,
		"title":               pr.Title,
		"state":               state,
		"isDraft":             pr.State == "DRAFT",
		"url":                 lo.CoalesceOrEmpty(pr.Url, "https://github.com/owner/repo/pull/"+pr.ID()),
		"createdAt":           pr.CreatedAt,
		"updatedAt":           pr.UpdatedAt,
		"author":              login(pr.Author),
		"baseRefName":         pr.BaseRefName,
		"headRefName":         pr.HeadRefName,
		"headRefOid":          pr.HeadRefOid,
		"headRepositoryOwner": login(pr.HeadRepositoryOwner.Login),
		"isCrossRepository":   pr.IsCrossRepository,
		"maintainerCanModify": pr.MaintainerCanModify,
		"reviewDecision":      pr.ReviewDecision,
		"additions":           pr.Additions,
		"deletions":           pr.Deletions,
		"comments":            map[string]any{"totalCount": pr.CommentsCount},
		"labels":              map[string]any{"nodes": lo.CoalesceSliceOrEmpty(pr.Labels)},
		"commits": map[string]any{"nodes": []any{map[string]any{"commit": map[string]any{
			"statusCheckRollup": lo.Ternary[any](pr.ChecksState == "", nil, map[string]any{"state": pr.ChecksState}),
		}}}},
	}
	if pr.HeadRepositoryName != "" {
		node["headRepository"] = map[string]any{"name": pr.HeadRepositoryName}
	}
	return node
}

func (self *FakeGithub) pullRequestDetailsData(pr *models.GithubPullRequest) map[string]any {
	details := self.PullRequestDetails[pr.Number]
	if details == nil {
		details = &models.GithubPullRequestDetails{}
	}
	settings := details.Repository
	if settings == (models.GithubRepositorySettings{}) {
		settings = models.GithubRepositorySettings{
			DefaultBranch:       "master",
			MergeCommitAllowed:  true,
			SquashMergeAllowed:  true,
			RebaseMergeAllowed:  true,
			ViewerDefaultMethod: "MERGE",
			ViewerPermission:    "WRITE",
		}
	}

	node := pullRequestNode(pr)
	node["body"] = details.Body
	node["viewerCanUpdate"] = true
	node["mergeable"] = lo.CoalesceOrEmpty(details.Mergeable, "MERGEABLE")
	node["mergeStateStatus"] = lo.CoalesceOrEmpty(details.MergeStateStatus, "CLEAN")
	node["baseRefOid"] = details.BaseRefOid
	node["mergedBy"] = login(details.MergedBy)
	node["milestone"] = lo.Ternary[any](details.Milestone == "", nil, map[string]any{"title": details.Milestone})
	node["assignees"] = map[string]any{"nodes": lo.Map(details.Assignees, func(a string, _ int) any { return login(a) })}
	node["reviewRequests"] = map[string]any{"nodes": lo.Map(details.ReviewRequests, func(reviewer string, _ int) any {
		if org, slug, isTeam := strings.Cut(reviewer, "/"); isTeam {
			return map[string]any{"requestedReviewer": map[string]any{"__typename": "Team", "slug": slug, "organization": login(org)}}
		}
		return map[string]any{"requestedReviewer": map[string]any{"__typename": "User", "login": reviewer}}
	})}
	reviewNode := func(review *models.GithubReview, _ int) any {
		return map[string]any{
			"id": review.NodeId, "author": login(review.Author), "state": review.State,
			"submittedAt": review.SubmittedAt, "body": review.Body, "url": review.Url,
		}
	}
	node["latestReviews"] = map[string]any{"nodes": lo.Map(details.LatestReviews, reviewNode)}
	reviews := lo.Filter(details.Timeline, func(item *models.GithubTimelineItem, _ int) bool { return item.ReviewState != "" })
	comments := lo.Filter(details.Timeline, func(item *models.GithubTimelineItem, _ int) bool { return item.ReviewState == "" })
	node["reviews"] = map[string]any{"nodes": lo.Map(reviews, func(item *models.GithubTimelineItem, _ int) any {
		return map[string]any{
			"author": login(item.Author), "state": item.ReviewState,
			"submittedAt": item.CreatedAt, "body": item.Body, "url": item.Url,
		}
	})}
	commentNodes := lo.Map(comments, func(item *models.GithubTimelineItem, _ int) any {
		return map[string]any{"author": login(item.Author), "body": item.Body, "createdAt": item.CreatedAt, "url": item.Url}
	})
	node["comments"] = map[string]any{"totalCount": len(comments)}
	node["conversation"] = map[string]any{"nodes": commentNodes}
	pending := []any{}
	if details.PendingReview != nil {
		pending = append(pending, map[string]any{
			"id": details.PendingReview.NodeId, "comments": map[string]any{"totalCount": details.PendingReview.CommentsCount},
		})
	}
	node["pendingReviews"] = map[string]any{"nodes": pending}
	node["files"] = map[string]any{"nodes": lo.CoalesceSliceOrEmpty(details.Files)}
	node["allCommits"] = map[string]any{"totalCount": details.CommitsCount}
	node["reviewThreads"] = map[string]any{"nodes": lo.Map(details.ReviewThreads, func(thread *models.GithubReviewThread, _ int) any {
		return map[string]any{
			"id": thread.NodeId, "path": thread.Path, "isResolved": thread.IsResolved, "isOutdated": thread.IsOutdated,
			"line": thread.Line, "originalLine": thread.OriginalLine, "diffSide": thread.DiffSide,
			"comments": map[string]any{"nodes": lo.Map(thread.Comments, func(comment *models.GithubReviewComment, _ int) any {
				return map[string]any{
					"id": comment.NodeId, "databaseId": comment.DatabaseId, "author": login(comment.Author),
					"body": comment.Body, "createdAt": comment.CreatedAt, "url": comment.Url,
					"diffHunk": comment.DiffHunk, "state": lo.Ternary(comment.IsPending, "PENDING", "SUBMITTED"),
				}
			})},
		}
	})}
	node["checks"] = map[string]any{"nodes": []any{map[string]any{"commit": map[string]any{"statusCheckRollup": map[string]any{
		"contexts": map[string]any{"nodes": lo.Map(details.Checks, func(check *models.GithubCheck, _ int) any {
			result := map[string]any{
				"__typename": "CheckRun", "name": check.Name, "status": strings.ToUpper(check.Status),
				"conclusion": strings.ToUpper(check.Conclusion), "detailsUrl": check.Url,
				"databaseId": check.JobId, "isRequired": check.IsRequired,
			}
			if check.RunId != 0 {
				result["checkSuite"] = map[string]any{"workflowRun": map[string]any{
					"databaseId": check.RunId, "workflow": map[string]any{"name": check.Workflow},
				}}
			}
			return result
		})},
	}}}}}
	if details.AutoMerge != nil {
		node["autoMergeRequest"] = map[string]any{"mergeMethod": details.AutoMerge.MergeMethod, "enabledBy": login(details.AutoMerge.EnabledBy)}
	}

	return map[string]any{
		"viewer": map[string]any{"login": lo.CoalesceOrEmpty(details.ViewerLogin, "me")},
		"repository": map[string]any{
			"defaultBranchRef":         map[string]any{"name": settings.DefaultBranch},
			"mergeCommitAllowed":       settings.MergeCommitAllowed,
			"squashMergeAllowed":       settings.SquashMergeAllowed,
			"rebaseMergeAllowed":       settings.RebaseMergeAllowed,
			"autoMergeAllowed":         settings.AutoMergeAllowed,
			"deleteBranchOnMerge":      settings.DeleteBranchOnMerge,
			"viewerDefaultMergeMethod": settings.ViewerDefaultMethod,
			"viewerCanAdminister":      settings.ViewerCanAdminister,
			"viewerPermission":         settings.ViewerPermission,
			"pullRequest":              node,
		},
	}
}
