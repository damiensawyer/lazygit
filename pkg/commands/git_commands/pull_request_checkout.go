package git_commands

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// PullRequestCheckout is what checking out a pull request needs to know about
// it and about the repository.
//
// The commands it produces are the ones `gh pr checkout` runs (cli/cli's
// pkg/cmd/pr/checkout), so that a pull request checked out by lazygit ends up
// the same as one checked out by gh: on a branch named after the pull
// request's head branch, set up so that pulling it brings in new commits of
// the pull request, and, where the maintainers may push to it, pushing it
// pushes to the pull request.
type PullRequestCheckout struct {
	Number      int
	HeadRefName string
	// the owner's login, and the repository's name, of the repository the
	// head branch is in; HeadRepoName is empty if it has been deleted
	HeadRepoOwner       string
	HeadRepoName        string
	IsCrossRepository   bool
	MaintainerCanModify bool

	// The remote for the base repository, or, without one, its URL.
	BaseRemote string
	// The remote for the head repository, if there is one. For a pull request
	// from a branch of the base repository, it is the base remote.
	HeadRemote string
	// The URL of the head repository, for setting up a branch to pull from and
	// push to it when there is no remote for it.
	HeadRepoURL   string
	DefaultBranch string

	CurrentBranch     string
	LocalBranchExists func(name string) bool
	// whether a local branch already says where to pull from
	HasMergeConfig func(branch string) bool

	// What to check the pull request out as: a local branch of this name
	// rather than one named like the head branch, or no branch at all.
	BranchName string
	Detach     bool
	// Reset an existing local branch to the pull request's head rather than
	// only fast-forwarding it.
	Force bool
	// Check the pull request out in a new worktree at this path.
	WorktreePath string
}

// LocalBranch returns the name of the local branch the pull request is
// checked out as.
func (self PullRequestCheckout) LocalBranch() string {
	if self.BranchName != "" {
		return self.BranchName
	}
	if self.HeadRemote == "" && self.HeadRefName == self.DefaultBranch {
		// Avoid naming the new branch the same as the default branch.
		return self.HeadRepoOwner + "/" + self.HeadRefName
	}
	return self.HeadRefName
}

// Commands returns the git commands (without the leading "git") that check
// out the pull request.
func (self PullRequestCheckout) Commands() ([][]string, error) {
	if strings.HasPrefix(self.HeadRefName, "-") {
		return nil, fmt.Errorf("invalid branch name: %q", self.HeadRefName)
	}
	if self.HeadRemote != "" {
		return self.commandsForExistingRemote(), nil
	}
	return self.commandsForMissingRemote(), nil
}

func (self PullRequestCheckout) commandsForExistingRemote() [][]string {
	remoteBranch := self.HeadRemote + "/" + self.HeadRefName
	refSpec := "+refs/heads/" + self.HeadRefName
	if !self.Detach {
		refSpec += ":refs/remotes/" + remoteBranch
	}
	fetch := []string{"fetch", self.HeadRemote, refSpec, "--no-tags"}

	if self.Detach {
		return self.detachCommands(fetch)
	}

	localBranch := self.LocalBranch()
	remoteBranchRef := "refs/remotes/" + remoteBranch
	commands := [][]string{fetch}
	switch {
	case self.WorktreePath != "":
		if self.LocalBranchExists(localBranch) {
			commands = append(commands, []string{"worktree", "add", "--", self.WorktreePath, localBranch})
			commands = append(commands, self.syncBranchCommands(self.WorktreePath, remoteBranchRef)...)
		} else {
			commands = append(commands, []string{"worktree", "add", "--track", "-b", localBranch, "--", self.WorktreePath, remoteBranch})
		}
	case self.LocalBranchExists(localBranch):
		commands = append(commands, []string{"checkout", localBranch})
		commands = append(commands, self.syncBranchCommands("", remoteBranchRef)...)
	default:
		commands = append(commands, []string{"checkout", "-b", localBranch, "--track", remoteBranch})
	}
	return commands
}

func (self PullRequestCheckout) commandsForMissingRemote() [][]string {
	ref := fmt.Sprintf("refs/pull/%d/head", self.Number)

	if self.Detach {
		return self.detachCommands([]string{"fetch", self.BaseRemote, ref, "--no-tags"})
	}

	localBranch := self.LocalBranch()
	fetchIntoBranch := []string{"fetch", self.BaseRemote, ref + ":" + localBranch, "--no-tags"}
	if self.Force {
		fetchIntoBranch = append(fetchIntoBranch, "--force")
	}

	var commands [][]string
	switch {
	case self.WorktreePath != "":
		commands = append(commands, fetchIntoBranch, []string{"worktree", "add", "--", self.WorktreePath, localBranch})
	case localBranch == self.CurrentBranch:
		// git won't fetch into the branch that is checked out.
		commands = append(commands, []string{"fetch", self.BaseRemote, ref, "--no-tags"})
		commands = append(commands, self.syncBranchCommands("", "FETCH_HEAD")...)
	default:
		commands = append(commands, fetchIntoBranch, []string{"checkout", localBranch})
	}

	remote := self.BaseRemote
	mergeRef := ref
	if self.MaintainerCanModify && self.HeadRepoName != "" {
		remote = self.HeadRepoURL
		mergeRef = "refs/heads/" + self.HeadRefName
	}
	if !self.HasMergeConfig(localBranch) {
		// .remote is needed for `git pull` to work, and .pushRemote for
		// `git push` to work if remote.pushDefault is set.
		commands = append(commands,
			[]string{"config", fmt.Sprintf("branch.%s.remote", localBranch), remote},
			[]string{"config", fmt.Sprintf("branch.%s.pushRemote", localBranch), remote},
			[]string{"config", fmt.Sprintf("branch.%s.merge", localBranch), mergeRef},
		)
	}
	return commands
}

func (self PullRequestCheckout) detachCommands(fetch []string) [][]string {
	if self.WorktreePath != "" {
		return [][]string{fetch, {"worktree", "add", "--detach", "--", self.WorktreePath, "FETCH_HEAD"}}
	}
	return [][]string{fetch, {"checkout", "--detach", "FETCH_HEAD"}}
}

// syncBranchCommands bring a branch up to date with ref: by fast-forwarding
// it, or with Force, by resetting it. A non-empty path runs them in that
// worktree.
func (self PullRequestCheckout) syncBranchCommands(path string, ref string) [][]string {
	var prefix []string
	if path != "" {
		prefix = []string{"-C", path}
	}
	if self.Force {
		return [][]string{append(prefix, "reset", "--hard", ref)}
	}
	return [][]string{append(prefix, "merge", "--ff-only", ref)}
}

// CheckOutPullRequest runs the commands that check out a pull request. Its
// fetches may need credentials, which they ask for through task.
func (self *GitHubCommands) CheckOutPullRequest(task gocui.Task, checkout PullRequestCheckout) error {
	commands, err := checkout.Commands()
	if err != nil {
		return err
	}
	for _, args := range commands {
		cmdObj := self.cmd.New(NewGitCmd(args[0]).Arg(args[1:]...).ToArgv())
		if args[0] == "fetch" {
			cmdObj.PromptOnCredentialRequest(task)
		}
		if err := cmdObj.Run(); err != nil {
			return err
		}
	}
	return nil
}
