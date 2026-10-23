package controllers

import (
	goContext "context"
	"fmt"
	"slices"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// checks lists a pull request's checks, the way `gh pr checks` does, failing
// ones first. A check that is a GitHub Actions job opens in the GitHub
// Actions view; any other opens in the browser.
func (self *PullRequestsController) checks(pr *models.GithubPullRequest) error {
	tr := self.c.Tr.PullRequests
	return self.c.Helpers().GithubPullRequests.WithDetails(pr, func(loaded *helpers.GithubPullRequestDetails) error {
		checks := slices.Clone(loaded.Details.Checks)
		if len(checks) == 0 {
			return fmt.Errorf("%s", tr.NoChecksToShow)
		}
		rank := func(check *models.GithubCheck) int {
			switch state := check.State(); {
			case state.IsFailed():
				return 0
			case state.IsActive():
				return 1
			default:
				return 2
			}
		}
		slices.SortStableFunc(checks, func(a, b *models.GithubCheck) int { return rank(a) - rank(b) })

		items := lo.Map(checks, func(check *models.GithubCheck, _ int) *types.MenuItem {
			name := check.DisplayName()
			if check.IsRequired {
				name += " " + style.FgYellow.Sprint(tr.Required)
			}
			return &types.MenuItem{
				Label: presentation.GithubActionsStateIcon(check.State()) + " " + name,
				OnPress: func() error {
					if check.RunId != 0 {
						return self.c.Helpers().GithubActions.OpenRunJobs(check.RunId, check.JobId)
					}
					if check.Url == "" {
						return nil
					}
					return self.c.OS().OpenLink(check.Url)
				},
			}
		})

		failedRuns := lo.Uniq(lo.FilterMap(checks, func(check *models.GithubCheck, _ int) (int64, bool) {
			return check.RunId, check.RunId != 0 && check.State().IsFailed()
		}))
		if len(failedRuns) > 0 {
			items = append(items, &types.MenuItem{
				Label:   tr.RerunFailedChecks,
				Tooltip: tr.RerunFailedChecksTooltip,
				OnPress: func() error {
					return self.c.Helpers().GithubPullRequests.RunAction(pr, tr.RerunningChecks,
						utils.ResolvePlaceholderString(tr.RerunRequested, map[string]string{"count": fmt.Sprint(len(failedRuns))}),
						func(ctx goContext.Context, client *github_commands.Client) error {
							for _, runId := range failedRuns {
								if err := client.RerunFailedJobs(ctx, runId, false); err != nil {
									return err
								}
							}
							return nil
						})
				},
			})
		}
		items = append(items, &types.MenuItem{
			Label: tr.OpenChecksInBrowser,
			OnPress: func() error {
				return self.c.OS().OpenLink(strings.TrimSuffix(pr.Url, "/") + "/checks")
			},
		})

		return self.c.Menu(types.CreateMenuOptions{
			Title: utils.ResolvePlaceholderString(tr.ChecksTitle, prPlaceholders(pr)),
			Items: items,
		})
	})
}
