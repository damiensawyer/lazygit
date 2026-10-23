package helpers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// PullRequestCheckoutOpts are the choices `gh pr checkout` offers as flags.
type PullRequestCheckoutOpts struct {
	BranchName string
	Detach     bool
	Force      bool
}

// CheckOut checks a pull request out the way `gh pr checkout` does.
func (self *GithubPullRequestsHelper) CheckOut(pr *models.GithubPullRequest, opts PullRequestCheckoutOpts) error {
	return self.withCheckout(pr, opts, func(checkout git_commands.PullRequestCheckout) error {
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.CheckingOut, func(task gocui.Task) error {
			self.c.LogAction(self.c.Tr.Actions.CheckOutPullRequest)
			if err := self.c.Git().GitHub.CheckOutPullRequest(task, checkout); err != nil {
				return err
			}
			self.c.Contexts().LocalCommits.SetLimitCommits(true)
			self.c.RefreshFromWorker(types.RefreshOptions{
				BatchUIUpdates: true,
				Scope: []types.RefreshableView{
					types.COMMITS, types.BRANCHES, types.REMOTES, types.FILES, types.REFLOG, types.WORKTREES, types.BISECT_INFO,
				},
				BranchSelection:       types.SelectCheckedOutBranch,
				CommitSelection:       types.SelectHeadCommit,
				SelectTopReflogCommit: true,
				Then: func() error {
					self.c.Toast(self.checkedOutMessage(pr, checkout))
					return nil
				},
			})
			return nil
		})
	})
}

// CheckOutInWorktree checks a pull request out in a new worktree, asking
// where, and switches to it.
func (self *GithubPullRequestsHelper) CheckOutInWorktree(pr *models.GithubPullRequest, worktreeHelper *WorktreeHelper, opts PullRequestCheckoutOpts) error {
	return self.withCheckout(pr, opts, func(checkout git_commands.PullRequestCheckout) error {
		repoName := filepath.Base(self.c.Git().RepoPaths.RepoPath())
		dirName := fmt.Sprintf("%s-pr-%d", repoName, pr.Number)
		prompt := utils.ResolvePlaceholderString(self.c.Tr.PullRequests.WorktreeLocationPrompt, map[string]string{
			"number": fmt.Sprint(pr.Number),
		})
		return worktreeHelper.PromptForWorktreeLocation(dirName, prompt, func(path string) error {
			checkout.WorktreePath = path
			return worktreeHelper.CreateWorktreeAndSwitch(path, self.c.Context().Current().GetKey(), func(task gocui.Task) error {
				self.c.LogAction(self.c.Tr.Actions.CheckOutPullRequest)
				return self.c.Git().GitHub.CheckOutPullRequest(task, checkout)
			})
		})
	})
}

func (self *GithubPullRequestsHelper) checkedOutMessage(pr *models.GithubPullRequest, checkout git_commands.PullRequestCheckout) string {
	if checkout.Detach {
		return utils.ResolvePlaceholderString(self.c.Tr.PullRequests.CheckedOutDetached, map[string]string{
			"number": fmt.Sprint(pr.Number),
		})
	}
	return utils.ResolvePlaceholderString(self.c.Tr.PullRequests.CheckedOut, map[string]string{
		"number": fmt.Sprint(pr.Number),
		"branch": checkout.LocalBranch(),
	})
}

// withCheckout works out how to check out a pull request, and passes that to
// f on the UI thread.
func (self *GithubPullRequestsHelper) withCheckout(pr *models.GithubPullRequest, opts PullRequestCheckoutOpts, f func(git_commands.PullRequestCheckout) error) error {
	return self.WithDetails(pr, func(details *GithubPullRequestDetails) error {
		return f(self.pullRequestCheckout(details.PullRequest, details.Details, opts))
	})
}

func (self *GithubPullRequestsHelper) pullRequestCheckout(pr *models.GithubPullRequest, details *models.GithubPullRequestDetails, opts PullRequestCheckoutOpts) git_commands.PullRequestCheckout {
	model := self.c.Model()
	repo := self.clientHelper.Client().Repo()
	baseRemote, baseRemoteURL := self.baseRemote()

	headRemote := baseRemote
	if pr.HeadRepositoryName == "" {
		headRemote = ""
	} else if pr.IsCrossRepository {
		headRemote = findRemoteForRepo(model.Remotes, pr.HeadRepositoryOwner.Login, pr.HeadRepositoryName)
	}

	branchName := opts.BranchName
	if branchName == "" {
		// A local branch that already tracks the pull request is the one to
		// bring up to date, whatever it is called.
		branchName = context.LocalBranchesOfPullRequests(model)[pr.Number]
	}

	branchesByName := lo.SliceToMap(model.Branches, func(branch *models.Branch) (string, *models.Branch) {
		return branch.Name, branch
	})

	return git_commands.PullRequestCheckout{
		Number:              pr.Number,
		HeadRefName:         pr.HeadRefName,
		HeadRepoOwner:       pr.HeadRepositoryOwner.Login,
		HeadRepoName:        pr.HeadRepositoryName,
		IsCrossRepository:   pr.IsCrossRepository,
		MaintainerCanModify: pr.MaintainerCanModify,
		BaseRemote:          baseRemote,
		HeadRemote:          headRemote,
		HeadRepoURL:         githubRepoURL(repo.Host, pr.HeadRepositoryOwner.Login, pr.HeadRepositoryName, baseRemoteURL),
		DefaultBranch:       details.Repository.DefaultBranch,
		CurrentBranch:       model.CheckedOutBranch,
		LocalBranchExists: func(name string) bool {
			_, exists := branchesByName[name]
			return exists
		},
		HasMergeConfig: func(name string) bool {
			branch, exists := branchesByName[name]
			return exists && branch.UpstreamBranch != ""
		},
		BranchName: branchName,
		Detach:     opts.Detach,
		Force:      opts.Force,
	}
}

// baseRemote returns the name of the remote for the base repository, or its
// URL if there is no remote for it, along with the remote's URL.
func (self *GithubPullRequestsHelper) baseRemote() (string, string) {
	repo := self.clientHelper.Client().Repo()
	name := self.clientHelper.BaseRemoteName()
	url := ""
	if remote, found := lo.Find(self.c.Model().Remotes, func(remote *models.Remote) bool { return remote.Name == name }); found && len(remote.Urls) > 0 {
		url = remote.Urls[0]
	}
	if name == "" {
		return githubRepoURL(repo.Host, repo.Owner, repo.Name, url), url
	}
	return name, url
}

// findRemoteForRepo returns the name of the remote whose URL points at the
// given repository, if there is one.
func findRemoteForRepo(remotes []*models.Remote, owner string, name string) string {
	remote, found := lo.Find(remotes, func(remote *models.Remote) bool {
		return lo.SomeBy(remote.Urls, func(url string) bool {
			info, err := hosting_service.GetRepoInfoFromURL(url)
			return err == nil && strings.EqualFold(info.Owner, owner) && strings.EqualFold(info.Repository, name)
		})
	})
	if !found {
		return ""
	}
	return remote.Name
}

// githubRepoURL returns the URL to clone a repository from, using SSH if the
// remote URL it is modelled on does; gh uses its configured git protocol for
// this, which is what that remote's URL tells us.
func githubRepoURL(host string, owner string, name string, modelURL string) string {
	if strings.HasPrefix(modelURL, "git@") || strings.HasPrefix(modelURL, "ssh://") {
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, name)
	}
	return fmt.Sprintf("https://%s/%s/%s.git", host, owner, name)
}

// ViewCommits fetches a pull request's head and shows its commits in place of
// the pull requests, the way entering a branch shows the branch's.
func (self *GithubPullRequestsHelper) ViewCommits(pr *models.GithubPullRequest, subCommitsHelper *SubCommitsHelper) error {
	return self.clientHelper.WithClient(func(*github_commands.Client) error {
		remote, _ := self.baseRemote()
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.FetchingPullRequest, func(task gocui.Task) error {
			if err := self.c.Git().GitHub.FetchPullRequestHead(task, remote, pr.Number); err != nil {
				return err
			}
			self.c.OnUIThread(func() error {
				return subCommitsHelper.ViewSubCommits(ViewSubCommitsOpts{
					Ref:      models.GithubPullRequestRef{Number: pr.Number, Title: pr.Title},
					TitleRef: fmt.Sprintf("#%d %s", pr.Number, pr.Title),
					Context:  self.c.Contexts().PullRequests,
				})
			})
			return nil
		})
	})
}
