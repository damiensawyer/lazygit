package github_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStoreToken(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GH_CONFIG_DIR", dir)
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

func TestSupportsDeviceLogin(t *testing.T) {
	assert.True(t, SupportsDeviceLogin("github.com"))
	assert.False(t, SupportsDeviceLogin("git.acme.com"))
}
