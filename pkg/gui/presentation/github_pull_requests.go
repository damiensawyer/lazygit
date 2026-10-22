package presentation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/icons"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type PullRequestListOpts struct {
	PullRequests []*models.GithubPullRequest
	// the local branch tracking each pull request's head branch, keyed by
	// the pull request's number
	LocalBranches    map[int]string
	CheckedOutBranch string
	FullDescription  bool
	Now              time.Time
	Tr               *i18n.TranslationSet
}

func GetPullRequestListDisplayStrings(opts PullRequestListOpts) [][]string {
	return lo.Map(opts.PullRequests, func(pr *models.GithubPullRequest, _ int) []string {
		localBranch := opts.LocalBranches[pr.Number]
		title := pr.Title
		if pr.IsDraft() {
			title = githubDim.Sprint(title)
		}

		columns := []string{
			pullRequestStateIcon(pr.State),
			style.FgCyan.Sprintf("#%d", pr.Number),
			localBranchMarker(localBranch, opts.CheckedOutBranch),
			checksStateIcon(pr.ChecksState, opts.Tr),
			reviewDecisionIcon(pr.ReviewDecision),
		}
		if opts.FullDescription {
			columns = append(columns,
				style.FgBlue.Sprint(utils.UnixToTimeAgo(pr.UpdatedAt.Unix())),
				style.FgMagenta.Sprint(pr.Author),
				style.FgYellow.Sprint(pr.HeadLabel()),
				githubDim.Sprint("→ "+pr.BaseRefName),
				formatDiffStat(pr.Additions, pr.Deletions),
				commentsCount(pr.CommentsCount),
				title+formatLabels(pr.Labels),
			)
		} else {
			columns = append(columns,
				style.FgMagenta.Sprint(utils.TruncateWithEllipsis(pr.Author, 12)),
				title,
			)
		}
		return columns
	})
}

func pullRequestStateIcon(state string) string {
	symbols := map[string]string{"OPEN": "●", "DRAFT": "○", "MERGED": "◆", "CLOSED": "✗"}
	if icons.IsIconEnabled() {
		symbols = map[string]string{"OPEN": "\uf407", "DRAFT": "\uebdb", "MERGED": "\uf4c9", "CLOSED": "\uebda"}
	}
	return WithPrColor(state, symbols[state], false)
}

func localBranchMarker(localBranch string, checkedOutBranch string) string {
	switch localBranch {
	case "":
		return " "
	case checkedOutBranch:
		return style.FgGreen.Sprint("*")
	default:
		return githubDim.Sprint("•")
	}
}

func checksStateIcon(state string, tr *i18n.TranslationSet) string {
	icon, _, textStyle := checksStatePresentation(state, tr)
	if icon == "" {
		return " "
	}
	return textStyle.Sprint(icon)
}

func reviewDecisionIcon(decision string) string {
	switch decision {
	case "APPROVED":
		return style.FgGreen.Sprint("✔")
	case "CHANGES_REQUESTED":
		return style.FgRed.Sprint("±")
	case "REVIEW_REQUIRED":
		return style.FgYellow.Sprint("◷")
	default:
		return " "
	}
}

func formatDiffStat(additions int, deletions int) string {
	return style.FgGreen.Sprintf("+%d", additions) + " " + style.FgRed.Sprintf("-%d", deletions)
}

func commentsCount(count int) string {
	if count == 0 {
		return ""
	}
	return githubDim.Sprintf("💬%d", count)
}

func formatLabels(labels []models.GithubLabel) string {
	if len(labels) == 0 {
		return ""
	}
	return " " + strings.Join(lo.Map(labels, func(label models.GithubLabel, _ int) string {
		return FormatGithubLabel(label)
	}), " ")
}

// FormatGithubLabel shows a label. GitHub shows it in a color of its own, but
// that is a hex color, which may not go with the terminal's palette.
func FormatGithubLabel(label models.GithubLabel) string {
	return style.FgCyan.Sprint("[" + label.Name + "]")
}

// PullRequestDetailsOpts is what FormatPullRequestDetails shows: a pull
// request, and its details once they have been loaded.
type PullRequestDetailsOpts struct {
	PullRequest *models.GithubPullRequest
	Details     *models.GithubPullRequestDetails
	DetailsErr  error
	LocalBranch string
	Now         time.Time
}

var (
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	blankLines  = regexp.MustCompile(`\n{3,}`)
)

// FormatPullRequestBody prepares a pull request's or comment's markdown for
// showing as plain text: without the HTML comments of a pull request
// template, which GitHub doesn't show either, and without the carriage
// returns that text typed into GitHub's web UI comes with, nor the blank lines
// the comments leave behind.
func FormatPullRequestBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = htmlComment.ReplaceAllString(body, "")
	body = blankLines.ReplaceAllString(body, "\n\n")
	return strings.TrimSpace(body)
}

// FormatPullRequestDetails describes a pull request in roughly the way
// `gh pr view` does, followed by its conversation and review threads.
func FormatPullRequestDetails(tr *i18n.TranslationSet, opts PullRequestDetailsOpts) string {
	pr := opts.PullRequest
	details := opts.Details
	t := tr.PullRequests
	var b strings.Builder
	line := func(format string, args ...any) {
		fmt.Fprintf(&b, format+"\n", args...)
	}
	resolve := func(template string, values map[string]string) string {
		return utils.ResolvePlaceholderString(template, values)
	}

	b.WriteString(FormatPullRequestHeader(pr, tr))

	line("%s", resolve(t.HeadIntoBase, map[string]string{
		"author": style.FgMagenta.Sprint(pr.Author),
		"head":   style.FgYellow.Sprint(pr.HeadLabel()),
		"base":   style.FgYellow.Sprint(pr.BaseRefName),
	}))
	ages := resolve(t.CreatedAndUpdated, map[string]string{
		"created": utils.UnixToTimeAgo(pr.CreatedAt.Unix()),
		"updated": utils.UnixToTimeAgo(pr.UpdatedAt.Unix()),
	})
	line("%s  %s", formatDiffStat(pr.Additions, pr.Deletions), githubDim.Sprint(ages))
	if opts.LocalBranch != "" {
		line("%s", resolve(t.CheckedOutAs, map[string]string{"branch": style.FgGreen.Sprint(opts.LocalBranch)}))
	}
	if len(pr.Labels) > 0 {
		line("%s%s", style.AttrBold.Sprint(t.Labels+":"), formatLabels(pr.Labels))
	}

	if opts.DetailsErr != nil {
		line("")
		line("%s", style.FgRed.Sprint(resolve(t.DetailsError, map[string]string{"error": opts.DetailsErr.Error()})))
		return b.String()
	}
	if details == nil {
		line("")
		line("%s", githubDim.Sprint(t.LoadingDetails))
		return b.String()
	}

	if details.Milestone != "" {
		line("%s %s", style.AttrBold.Sprint(t.Milestone+":"), details.Milestone)
	}
	if len(details.Assignees) > 0 {
		line("%s %s", style.AttrBold.Sprint(t.Assignees+":"), strings.Join(details.Assignees, ", "))
	}
	if reviewers := formatReviewers(details); reviewers != "" {
		line("%s %s", style.AttrBold.Sprint(t.Reviewers+":"), reviewers)
	}
	line("%s %s", style.AttrBold.Sprint(t.Checks+":"), formatCheckCounts(tr, details))
	if status := formatMergeStatus(tr, pr, details); status != "" {
		line("%s %s", style.AttrBold.Sprint(t.MergeStatus+":"), status)
	}
	if details.PendingReview != nil {
		line("%s", style.FgYellow.Sprint(resolve(t.PendingReviewNotice, map[string]string{
			"count": fmt.Sprint(details.PendingReview.CommentsCount),
		})))
	}

	line("")
	if body := FormatPullRequestBody(details.Body); body != "" {
		line("%s", body)
	} else {
		line("%s", githubDim.Sprint(t.NoDescription))
	}

	failedChecks := lo.Filter(details.Checks, func(check *models.GithubCheck, _ int) bool {
		return check.State().IsFailed()
	})
	if len(failedChecks) > 0 {
		line("")
		line("%s", style.AttrBold.Sprint(t.FailingChecks))
		for _, check := range failedChecks {
			required := lo.Ternary(check.IsRequired, githubDim.Sprint(" "+t.Required), "")
			line("  %s %s%s", GithubActionsStateIcon(check.State()), check.DisplayName(), required)
		}
	}

	if len(details.Files) > 0 {
		line("")
		line("%s", style.AttrBold.Sprint(resolve(t.FilesChanged, map[string]string{
			"count":   fmt.Sprint(len(details.Files)),
			"commits": fmt.Sprint(details.CommitsCount),
		})))
		for _, file := range details.Files {
			line("  %s %s %s", fileChangeType(file.ChangeType), file.Path, formatDiffStat(file.Additions, file.Deletions))
		}
	}

	if len(details.Timeline) > 0 {
		line("")
		line("%s", style.AttrBold.Sprint(t.Conversation))
		for _, item := range details.Timeline {
			line("")
			line("%s", formatTimelineHeading(tr, item))
			if body := FormatPullRequestBody(item.Body); body != "" {
				line("%s", indent(body, "  "))
			}
		}
	}

	if len(details.ReviewThreads) > 0 {
		line("")
		line("%s", style.AttrBold.Sprint(resolve(t.ReviewThreadsHeading, map[string]string{
			"unresolved": fmt.Sprint(len(details.UnresolvedThreads())),
			"total":      fmt.Sprint(len(details.ReviewThreads)),
		})))
		for _, thread := range details.ReviewThreads {
			line("")
			b.WriteString(FormatReviewThread(tr, thread, false))
		}
	}

	return b.String()
}

// fileChangeType abbreviates how a pull request changes a file the way git
// status does.
func fileChangeType(changeType string) string {
	switch changeType {
	case "ADDED":
		return style.FgGreen.Sprint("A")
	case "DELETED":
		return style.FgRed.Sprint("D")
	case "RENAMED":
		return style.FgCyan.Sprint("R")
	case "COPIED":
		return style.FgCyan.Sprint("C")
	default:
		return style.FgYellow.Sprint("M")
	}
}

func formatReviewers(details *models.GithubPullRequestDetails) string {
	reviewers := lo.FilterMap(details.LatestReviews, func(review *models.GithubReview, _ int) (string, bool) {
		if review.State == "COMMENTED" && details.ReviewRequestedFrom(review.Author) {
			// re-requested since; shown below as requested
			return "", false
		}
		return reviewStateIcon(review.State) + " " + review.Author, true
	})
	for _, requested := range details.ReviewRequests {
		reviewers = append(reviewers, reviewStateIcon("REQUESTED")+" "+requested)
	}
	return strings.Join(reviewers, "  ")
}

func reviewStateIcon(state string) string {
	switch state {
	case "APPROVED":
		return style.FgGreen.Sprint("✔")
	case "CHANGES_REQUESTED":
		return style.FgRed.Sprint("±")
	case "COMMENTED":
		return githubDim.Sprint("💬")
	case "DISMISSED":
		return githubDim.Sprint("✗")
	default:
		return style.FgYellow.Sprint("◷")
	}
}

func formatCheckCounts(tr *i18n.TranslationSet, details *models.GithubPullRequestDetails) string {
	t := tr.PullRequests
	if len(details.Checks) == 0 {
		return githubDim.Sprint(t.NoChecks)
	}
	counts := details.CheckCounts()
	parts := []string{}
	add := func(count int, template string, textStyle style.TextStyle) {
		if count > 0 {
			parts = append(parts, textStyle.Sprint(utils.ResolvePlaceholderString(template, map[string]string{"count": fmt.Sprint(count)})))
		}
	}
	add(counts.Failed, t.ChecksFailed, style.FgRed)
	add(counts.Pending, t.ChecksPending, style.FgYellow)
	add(counts.Passed, t.ChecksPassed, style.FgGreen)
	add(counts.Skipped, t.ChecksSkipped, githubDim)
	return strings.Join(parts, ", ")
}

func formatMergeStatus(tr *i18n.TranslationSet, pr *models.GithubPullRequest, details *models.GithubPullRequestDetails) string {
	t := tr.PullRequests
	switch pr.State {
	case "MERGED":
		return WithPrColor("MERGED", utils.ResolvePlaceholderString(t.MergedBy, map[string]string{
			"user": details.MergedBy,
			"ago":  utils.UnixToTimeAgo(details.MergedAt.Unix()),
		}), false)
	case "CLOSED":
		return WithPrColor("CLOSED", utils.ResolvePlaceholderString(t.ClosedAgo, map[string]string{
			"ago": utils.UnixToTimeAgo(details.ClosedAt.Unix()),
		}), false)
	}

	var status string
	switch {
	case details.Mergeable == "CONFLICTING" || details.MergeStateStatus == "DIRTY":
		status = style.FgRed.Sprint(t.MergeConflicts)
	case details.MergeStateStatus == "BEHIND":
		status = style.FgYellow.Sprint(t.MergeBehind)
	case details.MergeStateStatus == "BLOCKED":
		status = style.FgYellow.Sprint(t.MergeBlocked)
	case details.MergeStateStatus == "UNSTABLE":
		status = style.FgYellow.Sprint(t.MergeUnstable)
	case details.MergeStateStatus == "CLEAN" || details.MergeStateStatus == "HAS_HOOKS":
		status = style.FgGreen.Sprint(t.MergeReady)
	case details.MergeStateStatus == "DRAFT":
		status = githubDim.Sprint(t.MergeDraft)
	default:
		status = githubDim.Sprint(t.MergeUnknown)
	}
	if details.AutoMerge != nil {
		status += "  " + style.FgCyan.Sprint(utils.ResolvePlaceholderString(t.AutoMergeEnabled, map[string]string{
			"method": strings.ToLower(details.AutoMerge.MergeMethod),
			"user":   details.AutoMerge.EnabledBy,
		}))
	}
	return status
}

func formatTimelineHeading(tr *i18n.TranslationSet, item *models.GithubTimelineItem) string {
	t := tr.PullRequests
	action := t.Commented
	switch item.ReviewState {
	case "APPROVED":
		action = style.FgGreen.Sprint(t.Approved)
	case "CHANGES_REQUESTED":
		action = style.FgRed.Sprint(t.RequestedChanges)
	case "DISMISSED":
		action = githubDim.Sprint(t.ReviewDismissed)
	case "COMMENTED":
		action = t.Reviewed
	}
	return fmt.Sprintf("%s %s %s", style.FgMagenta.Sprint(item.Author), action,
		style.FgBlue.Sprint(utils.UnixToTimeAgo(item.CreatedAt.Unix())))
}

// FormatReviewThread shows a review thread: the file and line it is about,
// the end of the diff hunk leading up to that line, and its comments.
func FormatReviewThread(tr *i18n.TranslationSet, thread *models.GithubReviewThread, fullHunk bool) string {
	t := tr.PullRequests
	var b strings.Builder

	location := style.FgYellow.Sprint(thread.Path)
	if line := lo.Ternary(thread.Line != 0, thread.Line, thread.OriginalLine); line != 0 {
		location += githubDim.Sprintf(":%d", line)
	}
	tags := []string{}
	if thread.IsResolved {
		tags = append(tags, style.FgGreen.Sprint(t.Resolved))
	}
	if thread.IsOutdated {
		tags = append(tags, githubDim.Sprint(t.Outdated))
	}
	b.WriteString(location)
	if len(tags) > 0 {
		b.WriteString("  " + strings.Join(tags, " "))
	}
	b.WriteString("\n")

	if len(thread.Comments) > 0 && thread.Comments[0].DiffHunk != "" {
		hunkLines := strings.Split(strings.TrimRight(thread.Comments[0].DiffHunk, "\n"), "\n")
		if !fullHunk && len(hunkLines) > 4 {
			hunkLines = hunkLines[len(hunkLines)-4:]
		}
		for _, hunkLine := range hunkLines {
			b.WriteString("  " + colorDiffLine(hunkLine) + "\n")
		}
	}

	for _, comment := range thread.Comments {
		heading := style.FgMagenta.Sprint(comment.Author) + " " + style.FgBlue.Sprint(utils.UnixToTimeAgo(comment.CreatedAt.Unix()))
		if comment.IsPending {
			heading += " " + style.FgYellow.Sprint(t.Pending)
		}
		b.WriteString("  " + heading + "\n")
		b.WriteString(indent(FormatPullRequestBody(comment.Body), "    ") + "\n")
	}
	return b.String()
}

func colorDiffLine(line string) string {
	switch {
	case strings.HasPrefix(line, "@@"):
		return style.FgCyan.Sprint(line)
	case strings.HasPrefix(line, "+"):
		return style.FgGreen.Sprint(line)
	case strings.HasPrefix(line, "-"):
		return style.FgRed.Sprint(line)
	default:
		return githubDim.Sprint(line)
	}
}

func indent(text string, prefix string) string {
	lines := strings.Split(text, "\n")
	return strings.Join(lo.Map(lines, func(line string, _ int) string {
		if line == "" {
			return ""
		}
		return prefix + line
	}), "\n")
}
