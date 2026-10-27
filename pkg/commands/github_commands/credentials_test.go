package github_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
)

func setUpCredentials(t *testing.T, hostsYml string) {
	t.Helper()
	keyring.MockInit()
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
		t.Setenv(name, "")
	}
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	if hostsYml != "" {
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "hosts.yml"), []byte(hostsYml), 0o600))
	}
}

func TestTokenForHost(t *testing.T) {
	scenarios := []struct {
		name     string
		hostsYml string
		env      map[string]string
		keyring  map[string]string // user -> token, for gh:<host>
		host     string
		expected string
	}{
		{
			name:     "nothing anywhere",
			host:     "github.com",
			expected: "",
		},
		{
			name:     "GH_TOKEN wins over everything for github.com",
			hostsYml: "github.com:\n    oauth_token: from_config\n",
			env:      map[string]string{"GH_TOKEN": "from_gh_token", "GITHUB_TOKEN": "from_github_token"},
			host:     "github.com",
			expected: "from_gh_token",
		},
		{
			name:     "GITHUB_TOKEN for github.com",
			env:      map[string]string{"GITHUB_TOKEN": "from_github_token"},
			host:     "GitHub.com",
			expected: "from_github_token",
		},
		{
			name:     "GITHUB_TOKEN isn't used for an enterprise host",
			env:      map[string]string{"GITHUB_TOKEN": "from_github_token"},
			host:     "git.acme.com",
			expected: "",
		},
		{
			name:     "GH_ENTERPRISE_TOKEN for an enterprise host",
			env:      map[string]string{"GH_ENTERPRISE_TOKEN": "enterprise"},
			host:     "git.acme.com",
			expected: "enterprise",
		},
		{
			name:     "token stored insecurely for the host",
			hostsYml: "github.com:\n    user: octocat\n    oauth_token: from_config\n",
			keyring:  map[string]string{"octocat": "from_keyring"},
			host:     "github.com",
			expected: "from_config",
		},
		{
			name:     "active user's token in the keyring",
			hostsYml: "github.com:\n    user: octocat\n    users:\n        octocat:\n        hubot:\n",
			keyring:  map[string]string{"octocat": "octocat_token", "hubot": "hubot_token"},
			host:     "github.com",
			expected: "octocat_token",
		},
		{
			name:     "active user's token stored insecurely under users",
			hostsYml: "github.com:\n    user: octocat\n    users:\n        octocat:\n            oauth_token: octocat_token\n",
			host:     "github.com",
			expected: "octocat_token",
		},
		{
			name:     "token in the keyring without a user",
			hostsYml: "github.com:\n    git_protocol: ssh\n",
			keyring:  map[string]string{"": "unkeyed"},
			host:     "github.com",
			expected: "unkeyed",
		},
		{
			name:     "malformed config",
			hostsYml: "github.com: [unterminated\n",
			keyring:  map[string]string{"": "unkeyed"},
			host:     "github.com",
			expected: "unkeyed",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			setUpCredentials(t, s.hostsYml)
			for name, value := range s.env {
				t.Setenv(name, value)
			}
			for user, token := range s.keyring {
				assert.NoError(t, keyring.Set(keyringServiceName("github.com"), user, token))
			}

			assert.Equal(t, s.expected, TokenForHost(s.host))
		})
	}
}

func TestTokenForHostSeesChangesToTheConfig(t *testing.T) {
	setUpCredentials(t, "github.com:\n    oauth_token: first\n")
	assert.Equal(t, "first", TokenForHost("github.com"))

	assert.NoError(t, os.WriteFile(hostsConfigPath(), []byte("github.com:\n    oauth_token: second\n"), 0o600))
	assert.Equal(t, "second", TokenForHost("github.com"))
}

func TestGitProtocolForHost(t *testing.T) {
	setUpCredentials(t, "")
	assert.Equal(t, "https", GitProtocolForHost("github.com"))

	configDir := os.Getenv("GH_CONFIG_DIR")
	assert.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yml"), []byte("git_protocol: ssh\n"), 0o600))
	assert.Equal(t, "ssh", GitProtocolForHost("github.com"))

	assert.NoError(t, os.WriteFile(filepath.Join(configDir, "hosts.yml"),
		[]byte("github.com:\n    git_protocol: https\n"), 0o600))
	assert.Equal(t, "https", GitProtocolForHost("GitHub.com"))
	assert.Equal(t, "ssh", GitProtocolForHost("git.acme.com"))
}
