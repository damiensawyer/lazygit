package helpers

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestGetGithubBaseRemote(t *testing.T) {
	cases := []struct {
		name             string
		githubRemotes    []githubRemoteInfo
		configuredRemote string
		expected         string
	}{
		{
			name:             "configured remote wins",
			githubRemotes:    makeGithubRemoteInfoList("origin", "upstream", "fork"),
			configuredRemote: "fork",
			expected:         "fork",
		},
		{
			name:             "configured remote not in github remotes returns nil",
			githubRemotes:    makeGithubRemoteInfoList("origin"),
			configuredRemote: "missing",
			expected:         "",
		},
		{
			name:             "single github remote is auto-picked",
			githubRemotes:    makeGithubRemoteInfoList("myremote"),
			configuredRemote: "",
			expected:         "myremote",
		},
		{
			name:             "upstream is preferred when multiple github remotes exist",
			githubRemotes:    makeGithubRemoteInfoList("origin", "upstream", "fork"),
			configuredRemote: "",
			expected:         "upstream",
		},
		{
			name:             "no upstream and multiple remotes returns nil",
			githubRemotes:    makeGithubRemoteInfoList("origin", "fork"),
			configuredRemote: "",
			expected:         "",
		},
		{
			name:             "empty list returns nil",
			githubRemotes:    nil,
			configuredRemote: "",
			expected:         "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := getGithubBaseRemote(c.githubRemotes, c.configuredRemote)
			if c.expected == "" {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, c.expected, result.remote.Name)
			}
		})
	}
}

func TestGetAuthenticatedGithubRemotes(t *testing.T) {
	githubRemotes := []githubRemoteInfo{
		makeGithubRemoteInfo("origin", "github.com"),
		makeGithubRemoteInfo("fork", "github.com"),
		makeGithubRemoteInfo("enterprise", "ghe.example.com"),
		makeGithubRemoteInfo("missing-auth", "no-token.example.com"),
	}

	callsByHost := map[string]int{}
	result := getAuthenticatedGithubRemotes(githubRemotes, func(host string) string {
		callsByHost[host]++
		switch host {
		case "github.com":
			return "github-token"
		case "ghe.example.com":
			return "ghe-token"
		default:
			return ""
		}
	})

	assert.Equal(t, []githubRemoteInfo{
		makeAuthenticatedGithubRemoteInfo("origin", "github.com", "github-token"),
		makeAuthenticatedGithubRemoteInfo("fork", "github.com", "github-token"),
		makeAuthenticatedGithubRemoteInfo("enterprise", "ghe.example.com", "ghe-token"),
	}, result)
	// Two remotes share github.com; the lookup runs only once.
	assert.Equal(t, map[string]int{
		"github.com":           1,
		"ghe.example.com":      1,
		"no-token.example.com": 1,
	}, callsByHost)
}

func TestGetGithubRemotes(t *testing.T) {
	userConfig := config.GetDefaultConfig()
	userConfig.Services = map[string]string{
		"git.acme.com":   "github:git.acme.com",
		"ssh.acme.com":   "github:web.acme.com:8443",
		"gitlab.acme.io": "gitlab:gitlab.acme.io",
	}
	cmn := common.NewDummyCommonWithUserConfigAndAppState(userConfig, &config.AppState{})
	hostingService := git_commands.NewHostingServiceCommand(git_commands.NewGitCommon(cmn, nil, nil, nil, nil, nil, nil))

	remotes := []*models.Remote{
		{Name: "https", Urls: []string{"https://github.com/owner/https-repo.git"}},
		{Name: "https-without-suffix", Urls: []string{"https://github.com/owner/plain-repo"}},
		{Name: "scp-like", Urls: []string{"git@github.com:owner/scp-repo.git"}},
		{Name: "ssh-with-port", Urls: []string{"ssh://git@github.com:22/owner/ssh-repo.git"}},
		{Name: "enterprise", Urls: []string{"git@git.acme.com:team/enterprise-repo.git"}},
		{Name: "enterprise-other-web-host", Urls: []string{"ssh://git@ssh.acme.com/team/web-repo.git"}},
		{Name: "gitlab", Urls: []string{"git@gitlab.com:owner/gitlab-repo.git"}},
		{Name: "configured-gitlab", Urls: []string{"https://gitlab.acme.io/owner/repo.git"}},
		{Name: "unknown-host", Urls: []string{"https://example.com/owner/repo.git"}},
		{Name: "invalid", Urls: []string{"not a url"}},
		{Name: "no-urls", Urls: nil},
		// only the first URL counts, as it's the one git fetches from
		{Name: "github-second", Urls: []string{"git@gitlab.com:owner/repo.git", "git@github.com:owner/repo.git"}},
		{Name: "github-first", Urls: []string{"git@github.com:owner/first-repo.git", "git@gitlab.com:owner/repo.git"}},
	}

	result := getGithubRemotes(remotes, hostingService)

	assert.Equal(t, []githubRemoteInfo{
		{remote: remotes[0], serviceInfo: githubServiceInfo("github.com", "owner", "https-repo")},
		{remote: remotes[1], serviceInfo: githubServiceInfo("github.com", "owner", "plain-repo")},
		{remote: remotes[2], serviceInfo: githubServiceInfo("github.com", "owner", "scp-repo")},
		{remote: remotes[3], serviceInfo: githubServiceInfo("github.com", "owner", "ssh-repo")},
		{remote: remotes[4], serviceInfo: githubServiceInfo("git.acme.com", "team", "enterprise-repo")},
		{remote: remotes[5], serviceInfo: githubServiceInfo("web.acme.com:8443", "team", "web-repo")},
		{remote: remotes[12], serviceInfo: githubServiceInfo("github.com", "owner", "first-repo")},
	}, result)
}

func TestGetGithubRemotesWithoutRemotes(t *testing.T) {
	hostingService := git_commands.NewHostingServiceCommand(git_commands.NewGitCommon(common.NewDummyCommon(), nil, nil, nil, nil, nil, nil))

	assert.Empty(t, getGithubRemotes(nil, hostingService))
}

func TestGetAuthenticatedGithubRemotesWithoutTokens(t *testing.T) {
	githubRemotes := []githubRemoteInfo{
		makeGithubRemoteInfo("origin", "github.com"),
		makeGithubRemoteInfo("fork", "github.com"),
	}

	calls := 0
	result := getAuthenticatedGithubRemotes(githubRemotes, func(string) string {
		calls++
		return ""
	})

	assert.Empty(t, result)
	// A failed lookup is cached too.
	assert.Equal(t, 1, calls)
}

func TestGetGithubBaseRemoteConfiguredNameTakesPrecedenceOverUpstream(t *testing.T) {
	// A configured remote that isn't a GitHub remote means there is none to
	// use, even if one is called upstream.
	assert.Nil(t, getGithubBaseRemote(makeGithubRemoteInfoList("origin", "upstream"), "gitlab"))
}

func githubServiceInfo(webDomain string, owner string, repo string) hosting_service.ServiceInfo {
	return hosting_service.ServiceInfo{
		Provider:   "github",
		WebDomain:  webDomain,
		Owner:      owner,
		Repository: repo,
		RepoName:   owner + "/" + repo,
	}
}

func makeGithubRemoteInfoList(names ...string) []githubRemoteInfo {
	return lo.Map(names, func(name string, _ int) githubRemoteInfo {
		return makeGithubRemoteInfo(name, name)
	})
}

func makeGithubRemoteInfo(name string, webDomain string) githubRemoteInfo {
	return githubRemoteInfo{
		remote: &models.Remote{Name: name},
		serviceInfo: hosting_service.ServiceInfo{
			RepoName:  name,
			WebDomain: webDomain,
		},
	}
}

func makeAuthenticatedGithubRemoteInfo(name string, webDomain string, authToken string) githubRemoteInfo {
	info := makeGithubRemoteInfo(name, webDomain)
	info.authToken = authToken
	return info
}
