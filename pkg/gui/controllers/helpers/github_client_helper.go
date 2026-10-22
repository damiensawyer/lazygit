package helpers

import (
	"errors"
	"net/http"
	"sync"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// what we authenticate with to an integration test's fake API, which doesn't
// check it
const fakeGithubToken = "fake-github-token"

// GithubTokens remembers the tokens the user logged in with during this
// session. It outlives repo switches, unlike the helpers, so that the user
// doesn't have to log in again in every repo; and it keeps a token that
// couldn't be stored usable for the rest of the session.
type GithubTokens struct {
	mutex  sync.Mutex
	byHost map[string]string
}

func (self *GithubTokens) get(host string) string {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	return self.byHost[host]
}

func (self *GithubTokens) set(host string, token string) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	if self.byHost == nil {
		self.byHost = map[string]string{}
	}
	self.byHost[host] = token
}

// GithubClientHelper works out which GitHub repository the current repo
// belongs to, and gets a token for talking to it, logging the user in if
// there is none. It hands out a client for that repository to the GitHub
// views.
type GithubClientHelper struct {
	c           *HelperCommon
	loginHelper *GithubLoginHelper
	tokens      *GithubTokens

	// nil until the repo and token have been resolved
	client *github_commands.Client
}

func NewGithubClientHelper(c *HelperCommon, loginHelper *GithubLoginHelper, tokens *GithubTokens) *GithubClientHelper {
	return &GithubClientHelper{c: c, loginHelper: loginHelper, tokens: tokens}
}

// Client returns the client for the current repository, or nil if it hasn't
// been resolved yet.
func (self *GithubClientHelper) Client() *github_commands.Client {
	return self.client
}

// WithClient makes sure we know which repository to talk to and have a token
// for it, asking the user where we can't work that out, and then calls f on
// the UI thread.
func (self *GithubClientHelper) WithClient(f func(client *github_commands.Client) error) error {
	if self.client != nil {
		return f(self.client)
	}

	remotes := getGithubRemotes(self.c.Model().Remotes, self.c.Git().HostingService)
	if len(remotes) == 0 {
		return errors.New(self.c.Tr.GithubActions.NoGithubRemote)
	}

	remote := getGithubBaseRemote(remotes, self.c.Git().GitHub.ConfiguredBaseRemoteName())
	if remote == nil {
		return self.selectRemote(remotes, func(remote githubRemoteInfo) error {
			if err := self.c.Git().GitHub.SetConfiguredBaseRemoteName(remote.remote.Name); err != nil {
				self.c.Log.Error(err)
			}
			return self.withClientForRemote(remote, f, self.login)
		})
	}

	return self.withClientForRemote(*remote, f, self.login)
}

// TryClient is WithClient for when the user hasn't asked for anything: where
// the repository or the token can't be worked out without asking them, it
// calls onUnavailable on the UI thread instead of f.
func (self *GithubClientHelper) TryClient(f func(client *github_commands.Client) error, onUnavailable func() error) error {
	if self.client != nil {
		return f(self.client)
	}

	remotes := getGithubRemotes(self.c.Model().Remotes, self.c.Git().HostingService)
	remote := getGithubBaseRemote(remotes, self.c.Git().GitHub.ConfiguredBaseRemoteName())
	if remote == nil {
		return onUnavailable()
	}

	return self.withClientForRemote(*remote, f, func(string, func(string) error) error {
		return onUnavailable()
	})
}

// BaseRemoteName returns the name of the remote that is the GitHub
// repository, or "" if that isn't known without asking the user.
func (self *GithubClientHelper) BaseRemoteName() string {
	remotes := getGithubRemotes(self.c.Model().Remotes, self.c.Git().HostingService)
	if remote := getGithubBaseRemote(remotes, self.c.Git().GitHub.ConfiguredBaseRemoteName()); remote != nil {
		return remote.remote.Name
	}
	return ""
}

// HasGithubRemote reports whether the repository has a remote on GitHub.
func (self *GithubClientHelper) HasGithubRemote() bool {
	return len(getGithubRemotes(self.c.Model().Remotes, self.c.Git().HostingService)) > 0
}

func (self *GithubClientHelper) selectRemote(remotes []githubRemoteInfo, onSelect func(githubRemoteInfo) error) error {
	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.GithubActions.SelectRepository,
		Items: lo.Map(remotes, func(remote githubRemoteInfo, _ int) *types.MenuItem {
			return &types.MenuItem{
				LabelColumns: []string{remote.remote.Name, style.FgCyan.Sprint(remote.serviceInfo.RepoName)},
				OnPress:      func() error { return onSelect(remote) },
			}
		}),
	})
}

// withClientForRemote calls f with a client for the remote's repository, or,
// if there is no token for it, calls onNoToken with the host and with what to
// do with a token once there is one.
func (self *GithubClientHelper) withClientForRemote(
	remote githubRemoteInfo,
	f func(client *github_commands.Client) error,
	onNoToken func(host string, useToken func(token string) error) error,
) error {
	host := remote.serviceInfo.WebDomain
	repo := github_commands.Repo{
		Host:  host,
		Owner: remote.serviceInfo.Owner,
		Name:  remote.serviceInfo.Repository,
	}
	if baseURL := self.c.FakeGithubAPIURL(); baseURL != "" {
		repo.Token = fakeGithubToken
		self.client = github_commands.NewClientWithBaseURL(repo, baseURL)
		return f(self.client)
	}

	useToken := func(token string) error {
		repo.Token = token
		self.client = github_commands.NewClient(repo)
		return f(self.client)
	}

	if token := self.tokens.get(host); token != "" {
		return useToken(token)
	}

	// Looking the token up may have to wait for the system keyring, so do it
	// off the UI thread.
	self.c.OnWorker(func(gocui.Task) error {
		token := self.c.Git().GitHub.GetAuthToken(host)
		self.c.OnUIThread(func() error {
			if token == "" {
				return onNoToken(host, useToken)
			}
			return useToken(token)
		})
		return nil
	})
	return nil
}

func (self *GithubClientHelper) login(host string, onSuccess func(token string) error) error {
	return self.loginHelper.Login(host, func(token string) error {
		self.tokens.set(host, token)
		return onSuccess(token)
	})
}

// HandleError reports a failed API request. If the token has stopped working,
// it logs the user in again and then calls retry. Must be called on the UI
// thread.
func (self *GithubClientHelper) HandleError(err error, retry func() error) error {
	var apiErr *github_commands.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized && self.client != nil {
		// The token has been revoked or has expired; get a new one.
		host := self.client.Repo().Host
		self.client = nil
		self.tokens.set(host, "")
		return self.login(host, func(string) error { return retry() })
	}
	if github_commands.IsNotFound(err) {
		return errors.New(utils.ResolvePlaceholderString(self.c.Tr.GithubActions.RepoNotFound, map[string]string{
			"repo": self.client.Repo().FullName(),
		}))
	}
	return err
}
