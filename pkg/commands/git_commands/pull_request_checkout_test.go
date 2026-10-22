package git_commands

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestPullRequestCheckoutCommands(t *testing.T) {
	sameRepo := func() PullRequestCheckout {
		return PullRequestCheckout{
			Number:            12,
			HeadRefName:       "feature",
			HeadRepoOwner:     "owner",
			HeadRepoName:      "repo",
			BaseRemote:        "origin",
			HeadRemote:        "origin",
			DefaultBranch:     "master",
			CurrentBranch:     "master",
			LocalBranchExists: func(string) bool { return false },
			HasMergeConfig:    func(string) bool { return false },
		}
	}
	fork := func() PullRequestCheckout {
		checkout := sameRepo()
		checkout.HeadRepoOwner = "contributor"
		checkout.HeadRepoName = "repo-fork"
		checkout.IsCrossRepository = true
		checkout.HeadRemote = ""
		checkout.HeadRepoURL = "https://github.com/contributor/repo-fork.git"
		return checkout
	}
	with := func(checkout PullRequestCheckout, change func(*PullRequestCheckout)) PullRequestCheckout {
		change(&checkout)
		return checkout
	}
	existingBranches := func(names ...string) func(string) bool {
		return func(name string) bool { return lo.Contains(names, name) }
	}

	scenarios := []struct {
		name     string
		checkout PullRequestCheckout
		expected [][]string
	}{
		{
			name:     "branch of the base repository, new local branch",
			checkout: sameRepo(),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature:refs/remotes/origin/feature", "--no-tags"},
				{"checkout", "-b", "feature", "--track", "origin/feature"},
			},
		},
		{
			name:     "branch of the base repository, existing local branch",
			checkout: with(sameRepo(), func(c *PullRequestCheckout) { c.LocalBranchExists = existingBranches("feature") }),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature:refs/remotes/origin/feature", "--no-tags"},
				{"checkout", "feature"},
				{"merge", "--ff-only", "refs/remotes/origin/feature"},
			},
		},
		{
			name: "existing local branch, forced",
			checkout: with(sameRepo(), func(c *PullRequestCheckout) {
				c.LocalBranchExists = existingBranches("feature")
				c.Force = true
			}),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature:refs/remotes/origin/feature", "--no-tags"},
				{"checkout", "feature"},
				{"reset", "--hard", "refs/remotes/origin/feature"},
			},
		},
		{
			name:     "branch of the base repository, under another name",
			checkout: with(sameRepo(), func(c *PullRequestCheckout) { c.BranchName = "review-12" }),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature:refs/remotes/origin/feature", "--no-tags"},
				{"checkout", "-b", "review-12", "--track", "origin/feature"},
			},
		},
		{
			name:     "branch of the base repository, detached",
			checkout: with(sameRepo(), func(c *PullRequestCheckout) { c.Detach = true }),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature", "--no-tags"},
				{"checkout", "--detach", "FETCH_HEAD"},
			},
		},
		{
			name:     "branch of the base repository, in a new worktree",
			checkout: with(sameRepo(), func(c *PullRequestCheckout) { c.WorktreePath = "../repo-pr-12" }),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature:refs/remotes/origin/feature", "--no-tags"},
				{"worktree", "add", "--track", "-b", "feature", "--", "../repo-pr-12", "origin/feature"},
			},
		},
		{
			name: "existing local branch, in a new worktree",
			checkout: with(sameRepo(), func(c *PullRequestCheckout) {
				c.WorktreePath = "../repo-pr-12"
				c.LocalBranchExists = existingBranches("feature")
			}),
			expected: [][]string{
				{"fetch", "origin", "+refs/heads/feature:refs/remotes/origin/feature", "--no-tags"},
				{"worktree", "add", "--", "../repo-pr-12", "feature"},
				{"-C", "../repo-pr-12", "merge", "--ff-only", "refs/remotes/origin/feature"},
			},
		},
		{
			name:     "fork with a remote of its own",
			checkout: with(fork(), func(c *PullRequestCheckout) { c.HeadRemote = "contributor" }),
			expected: [][]string{
				{"fetch", "contributor", "+refs/heads/feature:refs/remotes/contributor/feature", "--no-tags"},
				{"checkout", "-b", "feature", "--track", "contributor/feature"},
			},
		},
		{
			name:     "fork without a remote, maintainers can't push",
			checkout: fork(),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head:feature", "--no-tags"},
				{"checkout", "feature"},
				{"config", "branch.feature.remote", "origin"},
				{"config", "branch.feature.pushRemote", "origin"},
				{"config", "branch.feature.merge", "refs/pull/12/head"},
			},
		},
		{
			name:     "fork without a remote, maintainers can push",
			checkout: with(fork(), func(c *PullRequestCheckout) { c.MaintainerCanModify = true }),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head:feature", "--no-tags"},
				{"checkout", "feature"},
				{"config", "branch.feature.remote", "https://github.com/contributor/repo-fork.git"},
				{"config", "branch.feature.pushRemote", "https://github.com/contributor/repo-fork.git"},
				{"config", "branch.feature.merge", "refs/heads/feature"},
			},
		},
		{
			name: "fork whose repository was deleted",
			checkout: with(fork(), func(c *PullRequestCheckout) {
				c.MaintainerCanModify = true
				c.HeadRepoName = ""
			}),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head:feature", "--no-tags"},
				{"checkout", "feature"},
				{"config", "branch.feature.remote", "origin"},
				{"config", "branch.feature.pushRemote", "origin"},
				{"config", "branch.feature.merge", "refs/pull/12/head"},
			},
		},
		{
			name: "fork already checked out and set up",
			checkout: with(fork(), func(c *PullRequestCheckout) {
				c.CurrentBranch = "feature"
				c.HasMergeConfig = func(string) bool { return true }
			}),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head", "--no-tags"},
				{"merge", "--ff-only", "FETCH_HEAD"},
			},
		},
		{
			name: "fork already checked out, forced",
			checkout: with(fork(), func(c *PullRequestCheckout) {
				c.CurrentBranch = "feature"
				c.HasMergeConfig = func(string) bool { return true }
				c.Force = true
			}),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head", "--no-tags"},
				{"reset", "--hard", "FETCH_HEAD"},
			},
		},
		{
			name: "fork whose head branch is named like the default branch",
			checkout: with(fork(), func(c *PullRequestCheckout) {
				c.HeadRefName = "master"
				c.HasMergeConfig = func(string) bool { return true }
			}),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head:contributor/master", "--no-tags"},
				{"checkout", "contributor/master"},
			},
		},
		{
			name:     "fork, detached",
			checkout: with(fork(), func(c *PullRequestCheckout) { c.Detach = true }),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head", "--no-tags"},
				{"checkout", "--detach", "FETCH_HEAD"},
			},
		},
		{
			name: "fork, in a new worktree",
			checkout: with(fork(), func(c *PullRequestCheckout) {
				c.WorktreePath = "../repo-pr-12"
				c.HasMergeConfig = func(string) bool { return true }
			}),
			expected: [][]string{
				{"fetch", "origin", "refs/pull/12/head:feature", "--no-tags"},
				{"worktree", "add", "--", "../repo-pr-12", "feature"},
			},
		},
		{
			name:     "fork, without a remote for the base repository",
			checkout: with(fork(), func(c *PullRequestCheckout) { c.BaseRemote = "https://github.com/owner/repo.git" }),
			expected: [][]string{
				{"fetch", "https://github.com/owner/repo.git", "refs/pull/12/head:feature", "--no-tags"},
				{"checkout", "feature"},
				{"config", "branch.feature.remote", "https://github.com/owner/repo.git"},
				{"config", "branch.feature.pushRemote", "https://github.com/owner/repo.git"},
				{"config", "branch.feature.merge", "refs/pull/12/head"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			commands, err := s.checkout.Commands()
			assert.NoError(t, err)
			assert.Equal(t, s.expected, commands)
		})
	}
}

func TestPullRequestCheckoutRejectsOptionLikeBranchNames(t *testing.T) {
	_, err := PullRequestCheckout{HeadRefName: "--upload-pack=evil"}.Commands()
	assert.EqualError(t, err, `invalid branch name: "--upload-pack=evil"`)
}
