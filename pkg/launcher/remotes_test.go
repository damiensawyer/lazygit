package launcher

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRemoteURLs(t *testing.T) {
	output := "remote.upstream.url git@github.com:jesseduffield/lazygit.git\n" +
		"remote.gitlab.url https://gitlab.com/me/lazygit.git\n" +
		"remote.origin.url https://github.com/me/lazygit.git\n" +
		"remote.mirror.url git@github.com:jesseduffield/lazygit.git\n"

	assert.Equal(t, []string{"me/lazygit", "jesseduffield/lazygit"}, parseRemoteURLs("github.com", output))
	assert.Empty(t, parseRemoteURLs("github.com", ""))
}

func TestFindRemotesOfRepos(t *testing.T) {
	dir := t.TempDir()
	repoDir := filepath.Join(dir, "repo")
	for _, args := range [][]string{
		{"init", "-q", repoDir},
		{"-C", repoDir, "remote", "add", "origin", "git@github.com:me/thing.git"},
	} {
		assert.NoError(t, exec.Command("git", args...).Run())
	}

	remotes := findRemotesOfRepos(context.Background(), "github.com",
		[]string{repoDir, filepath.Join(dir, "gone"), dir})

	assert.Equal(t, map[string][]string{repoDir: {"me/thing"}}, remotes)
}
