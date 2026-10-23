package helpers

import (
	goContext "context"
	"errors"
	"fmt"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// AddReviewComment asks for a comment on lines of a pull request's diff and
// adds it to the viewer's pending review, starting one if there is none.
// diffHead is the newest commit of the diff the lines were picked from, which
// has to be the pull request's head for the lines to be the ones GitHub has.
func (self *GithubPullRequestsHelper) AddReviewComment(number int, diffHead string, location github_commands.ReviewComment, lines string) error {
	tr := self.c.Tr.PullRequests
	return self.withDetailsByNumber(number, func(loaded *GithubPullRequestDetails) error {
		pr := loaded.PullRequest
		if pr.HeadRefOid != diffHead {
			return errors.New(utils.ResolvePlaceholderString(tr.CommentNeedsLatestCommit, map[string]string{
				"number": fmt.Sprint(number), "head": utils.ShortHash(pr.HeadRefOid),
			}))
		}
		pendingId := ""
		if loaded.Details.PendingReview != nil {
			pendingId = loaded.Details.PendingReview.NodeId
		}

		return self.EditText(EditTextOpts{
			Title: utils.ResolvePlaceholderString(tr.ReviewCommentTitle, map[string]string{
				"path": location.Path, "lines": lines,
			}),
		}, func(body string) error {
			location.Body = body
			return self.RunAction(pr, tr.AddingReviewComment, tr.ReviewCommentAdded,
				func(ctx goContext.Context, client *github_commands.Client) error {
					reviewId := pendingId
					if reviewId == "" {
						var err error
						if reviewId, err = client.StartPendingReview(ctx, pr.NodeId); err != nil {
							return err
						}
					}
					return client.AddReviewComment(ctx, reviewId, location)
				})
		})
	})
}

// withDetailsByNumber loads a pull request and its details, which needn't be
// one of the listed ones, and passes them to f on the UI thread.
func (self *GithubPullRequestsHelper) withDetailsByNumber(number int, f func(*GithubPullRequestDetails) error) error {
	return self.clientHelper.WithClient(func(client *github_commands.Client) error {
		generation := self.c.State().GetRepoGeneration()
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.LoadingDetails, func(gocui.Task) error {
			details := loadPullRequestDetails(client, &models.GithubPullRequest{Number: number})
			self.c.OnUIThread(func() error {
				if self.c.State().GetRepoGeneration() != generation {
					return nil
				}
				if details.Err != nil {
					return self.clientHelper.HandleError(details.Err, self.Refresh)
				}
				return f(details)
			})
			return nil
		})
	})
}
