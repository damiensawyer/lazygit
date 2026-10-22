package github_commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zalando/go-keyring"
	"gopkg.in/yaml.v3"
)

func TestStoreToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))
	path := filepath.Join(dir, "hosts.yml")

	existing := "# kept\n" +
		"github.com:\n" +
		"    git_protocol: ssh\n" +
		"    user: old\n" +
		"git.acme.com:\n" +
		"    oauth_token: other\n"
	assert.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "# kept\n"+
		"github.com:\n"+
		"    git_protocol: ssh\n"+
		"    user: octocat\n"+
		"    oauth_token: gho_new\n"+
		"    users:\n"+
		"        octocat:\n"+
		"            oauth_token: gho_new\n"+
		"git.acme.com:\n"+
		"    oauth_token: other\n", string(content))
}

func TestStoreTokenWithoutExistingConfig(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "gh")
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))

	assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

	content, err := os.ReadFile(filepath.Join(dir, "hosts.yml"))
	assert.NoError(t, err)
	assert.Equal(t, "github.com:\n"+
		"    oauth_token: gho_new\n"+
		"    user: octocat\n"+
		"    git_protocol: https\n"+
		"    users:\n"+
		"        octocat:\n"+
		"            oauth_token: gho_new\n", string(content))

	info, err := os.Stat(filepath.Join(dir, "hosts.yml"))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestStoreTokenUpdatesExistingEntries(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))
	path := filepath.Join(dir, "hosts.yml")

	existing := "github.com:\n" +
		"    oauth_token: gho_old\n" +
		"    user: octocat\n" +
		"    git_protocol: ssh\n" +
		"    users:\n" +
		"        octocat:\n" +
		"            oauth_token: gho_old\n" +
		"        hubot:\n" +
		"            oauth_token: gho_hubot\n"
	assert.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "github.com:\n"+
		"    oauth_token: gho_new\n"+
		"    user: octocat\n"+
		"    git_protocol: ssh\n"+
		"    users:\n"+
		"        octocat:\n"+
		"            oauth_token: gho_new\n"+
		"        hubot:\n"+
		"            oauth_token: gho_hubot\n", string(content))
}

func TestStoreTokenReplacesEntriesThatAreNotMappings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))
	path := filepath.Join(dir, "hosts.yml")

	assert.NoError(t, os.WriteFile(path, []byte("github.com:\n    users: [octocat]\n"), 0o600))

	assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "github.com:\n"+
		"    users:\n"+
		"        octocat:\n"+
		"            oauth_token: gho_new\n"+
		"    oauth_token: gho_new\n"+
		"    user: octocat\n"+
		"    git_protocol: https\n", string(content))
}

func TestStoreTokenInEmptyConfig(t *testing.T) {
	for name, existing := range map[string]string{"empty": "", "whitespace": "\n  \n"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("GH_CONFIG_DIR", dir)
			keyring.MockInitWithError(errors.New("no keyring"))
			keyring.MockInitWithError(errors.New("no keyring"))
			path := filepath.Join(dir, "hosts.yml")
			assert.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

			assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

			content, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Equal(t, "github.com:\n"+
				"    oauth_token: gho_new\n"+
				"    user: octocat\n"+
				"    git_protocol: https\n"+
				"    users:\n"+
				"        octocat:\n"+
				"            oauth_token: gho_new\n", string(content))
		})
	}
}

func TestStoreTokenInFlowStyleConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))
	path := filepath.Join(dir, "hosts.yml")
	assert.NoError(t, os.WriteFile(path, []byte("{}\n"), 0o600))

	assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	var hosts map[string]map[string]any
	assert.NoError(t, yaml.Unmarshal(content, &hosts))
	assert.Equal(t, map[string]map[string]any{
		"github.com": {
			"oauth_token":  "gho_new",
			"user":         "octocat",
			"git_protocol": "https",
			"users":        map[string]any{"octocat": map[string]any{"oauth_token": "gho_new"}},
		},
	}, hosts)
}

func TestStoreTokenKeepsMalformedConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))
	path := filepath.Join(dir, "hosts.yml")
	existing := "github.com:\n  oauth_token: [unterminated\n"
	assert.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	err := StoreToken("github.com", "octocat", "gho_new")

	assert.ErrorContains(t, err, "couldn't parse "+path)
	content, readErr := os.ReadFile(path)
	assert.NoError(t, readErr)
	assert.Equal(t, existing, string(content))
}

func TestStoreTokenFailsWhenConfigCannotBeRead(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInitWithError(errors.New("no keyring"))
	assert.NoError(t, os.Mkdir(filepath.Join(dir, "hosts.yml"), 0o700))

	assert.Error(t, StoreToken("github.com", "octocat", "gho_new"))
}

func TestStoreTokenInKeyring(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
	keyring.MockInit()
	path := filepath.Join(dir, "hosts.yml")

	existing := "github.com:\n" +
		"    oauth_token: gho_old\n" +
		"    user: hubot\n" +
		"    git_protocol: ssh\n" +
		"    users:\n" +
		"        octocat:\n" +
		"            oauth_token: gho_old\n"
	assert.NoError(t, os.WriteFile(path, []byte(existing), 0o600))

	assert.NoError(t, StoreToken("github.com", "octocat", "gho_new"))

	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "github.com:\n"+
		"    user: octocat\n"+
		"    git_protocol: ssh\n"+
		"    users:\n"+
		"        octocat: {}\n", string(content))

	for _, user := range []string{"octocat", ""} {
		token, err := keyring.Get("gh:github.com", user)
		assert.NoError(t, err)
		assert.Equal(t, "gho_new", token)
	}
	assert.Equal(t, "gho_new", TokenForHost("github.com"))
}

func TestSupportsDeviceLogin(t *testing.T) {
	assert.True(t, SupportsDeviceLogin("github.com"))
	assert.True(t, SupportsDeviceLogin("GitHub.com"))
	assert.True(t, SupportsDeviceLogin("acme.ghe.com"))
	assert.False(t, SupportsDeviceLogin("git.acme.com"))
	assert.False(t, SupportsDeviceLogin("git.acme.com:8443"))
}
