package launcher

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
)

func git(t *testing.T, args ...string) string {
	t.Helper()
	output, err := exec.Command("git", args...).CombinedOutput()
	assert.NoError(t, err, string(output))
	return strings.TrimSpace(string(output))
}

// setUpRemotes creates bare repos standing in for a repository on GitHub and
// a fork of it, returning their paths.
func setUpRemotes(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	work := filepath.Join(dir, "work")
	parent := filepath.Join(dir, "parent.git")
	fork := filepath.Join(dir, "fork.git")
	git(t, "init", "-q", "--initial-branch=main", work)
	git(t, "-C", work, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "--allow-empty", "-m", "initial")
	git(t, "clone", "-q", "--bare", work, parent)
	git(t, "clone", "-q", "--bare", parent, fork)
	return parent, fork
}

func newTestExecutor() (*Executor, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &Executor{Tr: i18n.EnglishTranslationSet(), Out: out, GitProtocol: "https"}, out
}

func TestExecuteCloneOfFork(t *testing.T) {
	parentPath, forkPath := setUpRemotes(t)
	dest := filepath.Join(t.TempDir(), "clone")
	fork := &models.GithubRepository{
		FullName: "me/thing", Owner: "me", Name: "thing", IsFork: true, CloneURL: forkPath,
		Parent: &models.GithubRepository{FullName: "them/thing", CloneURL: parentPath},
	}

	executor, out := newTestExecutor()
	repoDir, err := executor.Execute(context.Background(), Choice{Kind: ChoiceClone, Path: dest, Repo: fork})
	assert.NoError(t, err, out.String())

	assert.Equal(t, dest, repoDir)
	assert.Equal(t, forkPath, git(t, "-C", dest, "remote", "get-url", "origin"))
	assert.Equal(t, parentPath, git(t, "-C", dest, "remote", "get-url", "upstream"))
	assert.Equal(t, "base", git(t, "-C", dest, "config", "remote.upstream.gh-resolved"))
	assert.NotEmpty(t, git(t, "-C", dest, "rev-parse", "upstream/main"))
	assert.Contains(t, out.String(), "Cloning me/thing into "+dest)
	assert.Contains(t, out.String(), "Adding them/thing as remote 'upstream'")
}

func TestExecuteCloneOfNonFork(t *testing.T) {
	parentPath, _ := setUpRemotes(t)
	dest := filepath.Join(t.TempDir(), "clone")

	executor, out := newTestExecutor()
	_, err := executor.Execute(context.Background(), Choice{
		Kind: ChoiceClone,
		Path: dest,
		Repo: &models.GithubRepository{FullName: "them/thing", CloneURL: parentPath, SSHURL: "git@github.com:them/thing.git"},
	})
	assert.NoError(t, err, out.String())

	assert.Equal(t, "origin", git(t, "-C", dest, "remote"))
}

func TestExecuteForkAndClone(t *testing.T) {
	parentPath, forkPath := setUpRemotes(t)
	dest := filepath.Join(t.TempDir(), "clone")

	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"full_name": "me/thing", "name": "thing", "owner": {"login": "me"}, "fork": true, "clone_url": %q}`, forkPath)
	}))
	t.Cleanup(server.Close)

	executor, out := newTestExecutor()
	_, err := executor.Execute(context.Background(), Choice{
		Kind:   ChoiceClone,
		Path:   dest,
		Repo:   &models.GithubRepository{FullName: "them/thing", Owner: "them", Name: "thing", CloneURL: parentPath},
		Fork:   true,
		client: github_commands.NewClientWithBaseURL(github_commands.Repo{Host: "github.com", Token: "secret"}, server.URL),
	})
	assert.NoError(t, err, out.String())

	assert.Equal(t, []string{"POST /repos/them/thing/forks"}, requests)
	assert.Equal(t, forkPath, git(t, "-C", dest, "remote", "get-url", "origin"))
	assert.Equal(t, parentPath, git(t, "-C", dest, "remote", "get-url", "upstream"))
	assert.Contains(t, out.String(), "Forking them/thing")
}

func TestExecuteForkAndCloneRetriesUntilForkIsReady(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-there-yet.git")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"full_name": "me/thing", "clone_url": %q}`, missing)
	}))
	t.Cleanup(server.Close)

	executor, out := newTestExecutor()
	_, err := executor.Execute(context.Background(), Choice{
		Kind:   ChoiceClone,
		Path:   filepath.Join(t.TempDir(), "clone"),
		Repo:   &models.GithubRepository{FullName: "them/thing", Owner: "them", Name: "thing"},
		Fork:   true,
		client: github_commands.NewClientWithBaseURL(github_commands.Repo{Host: "github.com"}, server.URL),
	})

	assert.Error(t, err)
	assert.Equal(t, 2, strings.Count(out.String(), "The fork isn't ready yet"))
}

func TestExecuteInit(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	executor, out := newTestExecutor()
	repoDir, err := executor.Execute(context.Background(), Choice{Kind: ChoiceInit, InitialBranch: "trunk"})
	assert.NoError(t, err, out.String())

	assert.Equal(t, dir, repoDir)
	assert.Equal(t, "refs/heads/trunk", git(t, "-C", dir, "symbolic-ref", "HEAD"))
}

func TestCloneURL(t *testing.T) {
	repo := &models.GithubRepository{CloneURL: "https://github.com/o/r.git", SSHURL: "git@github.com:o/r.git"}
	assert.Equal(t, "https://github.com/o/r.git", (&Executor{GitProtocol: "https"}).cloneURL(repo))
	assert.Equal(t, "git@github.com:o/r.git", (&Executor{GitProtocol: "ssh"}).cloneURL(repo))
}
