package github_commands

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/cli/go-gh/v2/pkg/auth"
	ghconfig "github.com/cli/go-gh/v2/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

// The credential lookup below follows the GitHub CLI's (cli/cli's
// internal/config AuthConfig.ActiveToken), so that we find whatever token
// `gh auth token` would print without needing gh to be installed.

const keyringTimeout = 60 * time.Second

func keyringServiceName(host string) string {
	return "gh:" + host
}

// TokenForHost returns the token to authenticate against host with, or an
// empty string if there is none. It looks in the environment variables gh
// reads, then in gh's hosts.yml, and then in the system keyring, where gh
// keeps its tokens unless it was told to store them insecurely.
//
// hosts.yml is read afresh on every call, because gh rewrites it whenever the
// active account changes, and lazygit may have been running since before.
func TokenForHost(host string) string {
	host = auth.NormalizeHostname(host)

	if token := tokenFromEnv(host); token != "" {
		return token
	}

	hostEntry := readHostsConfig()[host]
	if token := hostEntry.OauthToken; token != "" {
		return token
	}

	if user := hostEntry.User; user != "" {
		if token, err := keyringGet(keyringServiceName(host), user); err == nil && token != "" {
			return token
		}
		if token := hostEntry.Users[user].OauthToken; token != "" {
			return token
		}
	}

	// Very old versions of gh keep the token in the keyring without a user.
	if token, err := keyringGet(keyringServiceName(host), ""); err == nil {
		return token
	}

	return ""
}

func tokenFromEnv(host string) string {
	var names []string
	if host == "github.com" || auth.IsTenancy(host) || host == "github.localhost" {
		names = []string{"GH_TOKEN", "GITHUB_TOKEN"}
	} else {
		names = []string{"GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}
	}
	for _, name := range names {
		if token := os.Getenv(name); token != "" {
			return token
		}
	}
	return ""
}

type hostsConfigEntry struct {
	OauthToken string `yaml:"oauth_token"`
	User       string `yaml:"user"`
	Users      map[string]struct {
		OauthToken string `yaml:"oauth_token"`
	} `yaml:"users"`
}

func hostsConfigPath() string {
	return filepath.Join(ghconfig.ConfigDir(), "hosts.yml")
}

// readHostsConfig reads gh's hosts.yml. A missing or malformed file reads as
// having no hosts in it, the way it does to gh.
func readHostsConfig() map[string]hostsConfigEntry {
	content, err := os.ReadFile(hostsConfigPath())
	if err != nil {
		return nil
	}
	var hosts map[string]hostsConfigEntry
	if yaml.Unmarshal(content, &hosts) != nil {
		return nil
	}
	return hosts
}

// keyringGet reads a secret from the system keyring, giving up after a while:
// on some systems the keyring daemon can hang rather than answer.
func keyringGet(service string, user string) (string, error) {
	type result struct {
		value string
		err   error
	}
	ch := make(chan result, 1)
	go utils.Safe(func() {
		value, err := keyring.Get(service, user)
		ch <- result{value, err}
	})
	select {
	case res := <-ch:
		return res.value, res.err
	case <-time.After(keyringTimeout):
		return "", errors.New("timeout while trying to get secret from keyring")
	}
}
