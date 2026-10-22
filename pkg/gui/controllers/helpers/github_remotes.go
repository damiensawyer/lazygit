package helpers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/git_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/hosting_service"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

type githubRemoteInfo struct {
	remote      *models.Remote
	serviceInfo hosting_service.ServiceInfo
	authToken   string
}

// getGithubRemotes returns the remotes that point at a GitHub instance.
func getGithubRemotes(remotes []*models.Remote, hostingService *git_commands.HostingService) []githubRemoteInfo {
	return lo.FilterMap(remotes, func(remote *models.Remote, _ int) (githubRemoteInfo, bool) {
		if len(remote.Urls) == 0 {
			return githubRemoteInfo{}, false
		}
		serviceInfo, err := hostingService.GetServiceInfo(remote.Urls[0])
		if err != nil || serviceInfo.Provider != "github" {
			return githubRemoteInfo{}, false
		}
		return githubRemoteInfo{remote: remote, serviceInfo: serviceInfo}, true
	})
}

// getAuthenticatedGithubRemotes drops remotes for which no auth token is
// available and attaches the resolved token to the rest. Token lookups are
// cached by host so that multiple remotes pointing at the same instance
// (e.g. origin + a fork on github.com) only trigger one lookup.
func getAuthenticatedGithubRemotes(githubRemotes []githubRemoteInfo, getAuthToken func(host string) string) []githubRemoteInfo {
	tokensByHost := map[string]string{}
	return lo.FilterMap(githubRemotes, func(info githubRemoteInfo, _ int) (githubRemoteInfo, bool) {
		host := info.serviceInfo.WebDomain
		token, cached := tokensByHost[host]
		if !cached {
			token = getAuthToken(host)
			tokensByHost[host] = token
		}
		if token == "" {
			return githubRemoteInfo{}, false
		}
		info.authToken = token
		return info, true
	})
}

func getGithubBaseRemote(githubRemotes []githubRemoteInfo, configuredRemoteName string) *githubRemoteInfo {
	findRemoteByName := func(name string) *githubRemoteInfo {
		info, ok := lo.Find(githubRemotes, func(info githubRemoteInfo) bool {
			return info.remote.Name == name
		})
		if !ok {
			return nil
		}
		return &info
	}

	if configuredRemoteName != "" {
		return findRemoteByName(configuredRemoteName)
	}

	if len(githubRemotes) == 1 {
		return &githubRemotes[0]
	}

	// Not sure if "upstream" is really a common convention for the name of the remote that PRs are
	// made against, but if it exists it's pretty likely to be the one we want.
	if info := findRemoteByName("upstream"); info != nil {
		return info
	}

	return nil
}
