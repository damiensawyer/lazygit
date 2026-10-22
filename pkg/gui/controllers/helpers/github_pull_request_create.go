package helpers

import (
	goContext "context"
	"fmt"
	"os"
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

// CreatePullRequest writes a pull request for a local branch and opens it on
// GitHub, the way `gh pr create` does: asking for the base branch, starting
// the title and description off from the branch's commits or the
// repository's pull request template, and pushing the branch first if it
// isn't pushed yet.
func (self *GithubPullRequestsHelper) CreatePullRequest(branch *models.Branch) error {
	tr := self.c.Tr.PullRequests
	return self.clientHelper.WithClient(func(client *github_commands.Client) error {
		return self.c.WithWaitingStatus(tr.Loading, func(gocui.Task) error {
			defaultBranch, err := client.GetDefaultBranch(goContext.Background())
			self.c.OnUIThread(func() error {
				if err != nil {
					return self.clientHelper.HandleError(err, func() error { return self.CreatePullRequest(branch) })
				}
				if branch.Name == defaultBranch {
					return fmt.Errorf("%s", utils.ResolvePlaceholderString(tr.CannotCreateFromDefaultBranch, map[string]string{
						"branch": branch.Name,
					}))
				}
				return self.promptForBase(branch, defaultBranch)
			})
			return nil
		})
	})
}

func (self *GithubPullRequestsHelper) promptForBase(branch *models.Branch, defaultBranch string) error {
	tr := self.c.Tr.PullRequests
	baseRemote, _ := self.baseRemote()
	self.c.Prompt(types.PromptOpts{
		Title:               utils.ResolvePlaceholderString(tr.CreateBaseBranchPrompt, map[string]string{"branch": branch.Name}),
		InitialContent:      defaultBranch,
		FindSuggestionsFunc: self.suggestionsHelper.GetRemoteBranchesForRemoteSuggestionsFunc(baseRemote),
		HandleConfirm: func(base string) error {
			base = strings.TrimSpace(base)
			return self.withTemplate(func(template string) error {
				return self.writePullRequest(branch, base, template)
			})
		},
	})
	return nil
}

// withTemplate lets the user choose one of the repository's pull request
// templates if it has several, and passes the chosen one's body to f.
func (self *GithubPullRequestsHelper) withTemplate(f func(template string) error) error {
	tr := self.c.Tr.PullRequests
	templates := github_commands.FindPullRequestTemplates(self.c.Git().RepoPaths.WorktreePath())
	switch len(templates) {
	case 0:
		return f("")
	case 1:
		return f(templates[0].Body)
	}

	items := lo.Map(templates, func(template github_commands.PullRequestTemplate, _ int) *types.MenuItem {
		return &types.MenuItem{
			Label:   template.Name,
			OnPress: func() error { return f(template.Body) },
		}
	})
	items = append(items, &types.MenuItem{
		Label:   tr.NoTemplate,
		OnPress: func() error { return f("") },
	})
	return self.c.Menu(types.CreateMenuOptions{Title: tr.ChooseTemplate, Items: items})
}

func (self *GithubPullRequestsHelper) writePullRequest(branch *models.Branch, base string, template string) error {
	tr := self.c.Tr.PullRequests
	baseRemote, _ := self.baseRemote()

	baseRef := base
	if !strings.Contains(baseRemote, "/") {
		baseRef = baseRemote + "/" + base
	}
	commits, err := self.c.Git().GitHub.CommitMessagesBetween(baseRef, branch.Name)
	if err != nil {
		self.c.Log.Warnf("Couldn't list the commits of %s: %v", branch.Name, err)
	}
	if len(commits) == 0 && err == nil {
		return fmt.Errorf("%s", utils.ResolvePlaceholderString(tr.NoCommitsToCreatePullRequest, map[string]string{
			"branch": branch.Name, "base": base,
		}))
	}

	title, body := git_commands.DefaultPullRequestTitleAndBody(branch.Name, commits)
	if template != "" {
		body = template
	}

	return self.EditTitleAndBody(EditTitleAndBodyOpts{
		Title:        utils.ResolvePlaceholderString(tr.CreateTitle, map[string]string{"branch": branch.Name, "base": base}),
		InitialTitle: title,
		InitialBody:  body,
	}, func(title string, body string) error {
		return self.chooseDraftAndCreate(branch, base, title, body)
	})
}

type EditTitleAndBodyOpts struct {
	Title        string
	InitialTitle string
	InitialBody  string
}

// EditTitleAndBody lets the user write a pull request's title and
// description, in the commit message panel or, from there, in their editor,
// and passes them to onConfirm.
func (self *GithubPullRequestsHelper) EditTitleAndBody(opts EditTitleAndBodyOpts, onConfirm func(title string, body string) error) error {
	submit := func(title string, body string) error {
		return onConfirm(strings.TrimSpace(title), strings.TrimSpace(body))
	}
	self.commitsHelper.OpenCommitMessagePanel(&OpenCommitMessagePanelOpts{
		CommitIndex:       context.NoCommitIndex,
		SummaryTitle:      opts.Title,
		DescriptionTitle:  self.c.Tr.PullRequests.DescriptionTitle,
		InitialMessage:    joinFirstLineAndRest(opts.InitialTitle, opts.InitialBody),
		UnwrapDescription: true,
		OnConfirm:         submit,
		OnSwitchToEditor: func(path string) error {
			if err := self.filesHelper.EditFileAtLineAndWait(path, 1); err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			title, body, _ := strings.Cut(strings.TrimSpace(string(content)), "\n")
			if strings.TrimSpace(title) == "" {
				return nil
			}
			return submit(title, body)
		},
	})
	return nil
}

func (self *GithubPullRequestsHelper) chooseDraftAndCreate(branch *models.Branch, base string, title string, body string) error {
	tr := self.c.Tr.PullRequests
	item := func(label string, draft bool) *types.MenuItem {
		return &types.MenuItem{
			Label:   label,
			OnPress: func() error { return self.pushAndCreate(branch, base, title, body, draft) },
		}
	}
	prompt := ""
	if needsPush(branch) {
		prompt = utils.ResolvePlaceholderString(tr.CreateWillPush, map[string]string{
			"branch": branch.Name, "remote": self.pushRemote(branch),
		})
	}
	return self.c.Menu(types.CreateMenuOptions{
		Title:  title,
		Prompt: prompt,
		Items: []*types.MenuItem{
			item(tr.CreatePullRequest, false),
			item(tr.CreateDraftPullRequest, true),
		},
	})
}

func needsPush(branch *models.Branch) bool {
	return !branch.IsTrackingRemote() || branch.RemoteBranchNotStoredLocally() || branch.AheadForPush != "0"
}

// pushRemote returns the remote a branch is pushed to: the one it tracks, or
// for a branch that isn't pushed yet, origin, which is where gh pushes too
// unless told otherwise, or else the base repository's remote.
func (self *GithubPullRequestsHelper) pushRemote(branch *models.Branch) string {
	if branch.IsTrackingRemote() {
		return branch.UpstreamRemote
	}
	if lo.SomeBy(self.c.Model().Remotes, func(remote *models.Remote) bool { return remote.Name == "origin" }) {
		return "origin"
	}
	baseRemote, _ := self.baseRemote()
	return baseRemote
}

func (self *GithubPullRequestsHelper) pushAndCreate(branch *models.Branch, base string, title string, body string, draft bool) error {
	tr := self.c.Tr.PullRequests
	client := self.clientHelper.Client()
	remote := self.pushRemote(branch)
	remoteBranch := lo.Ternary(branch.IsTrackingRemote(), branch.UpstreamBranch, branch.Name)
	push := needsPush(branch)

	head := remoteBranch
	if remoteModel, found := lo.Find(self.c.Model().Remotes, func(r *models.Remote) bool { return r.Name == remote }); found && len(remoteModel.Urls) > 0 {
		if info, err := hosting_service.GetRepoInfoFromURL(remoteModel.Urls[0]); err == nil &&
			!strings.EqualFold(info.Owner, client.Repo().Owner) {
			head = info.Owner + ":" + remoteBranch
		}
	}

	return self.c.WithWaitingStatus(tr.Creating, func(task gocui.Task) error {
		if push {
			self.c.LogAction(self.c.Tr.Actions.Push)
			if err := self.c.Git().Sync.Push(task, git_commands.PushOpts{
				Remote:      remote,
				Refspecs:    []string{"refs/heads/" + branch.Name + ":refs/heads/" + remoteBranch},
				SetUpstream: !branch.IsTrackingRemote(),
			}); err != nil {
				return err
			}
		}

		self.c.LogAction(self.c.Tr.Actions.CreatePullRequest)
		created, err := client.CreatePullRequest(goContext.Background(), github_commands.CreatePullRequestOptions{
			Title: title,
			Body:  body,
			Head:  head,
			Base:  base,
			Draft: draft,
		})
		if err != nil {
			self.c.OnUIThread(func() error { return self.clientHelper.HandleError(err, self.Refresh) })
			return nil
		}

		self.c.RefreshFromWorker(types.RefreshOptions{
			Scope: []types.RefreshableView{types.BRANCHES, types.COMMITS, types.PULL_REQUESTS},
		})
		self.c.OnUIThread(func() error {
			self.c.Toast(utils.ResolvePlaceholderString(tr.Created, map[string]string{
				"number": fmt.Sprint(created.Number),
			}))
			return nil
		})
		return self.loadPullRequests(false, func() { self.selectPullRequest(created.Number) })
	})
}

// selectPullRequest selects the pull request with the given number, if it is
// listed.
func (self *GithubPullRequestsHelper) selectPullRequest(number int) {
	prsContext := self.c.Contexts().PullRequests
	if _, idx, found := lo.FindIndexOf(prsContext.GetItems(), func(pr *models.GithubPullRequest) bool {
		return pr.Number == number
	}); found {
		prsContext.SetSelection(idx)
		self.c.PostRefreshUpdate(prsContext)
	}
}
