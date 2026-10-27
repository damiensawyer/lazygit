package launcher

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// findRemotesOfRepos returns, for each of the repos at paths that has any,
// the full names of the GitHub repositories on host that its remotes point
// to. Recent repos may be on a network drive that has gone away, so each one
// gets only a little while to answer.
func findRemotesOfRepos(ctx context.Context, host string, paths []string) map[string][]string {
	var mutex sync.Mutex
	result := map[string][]string{}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 8)
	for _, path := range paths {
		wg.Add(1)
		go utils.Safe(func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			names := findRemotesOfRepo(ctx, host, path)
			if len(names) > 0 {
				mutex.Lock()
				result[path] = names
				mutex.Unlock()
			}
		})
	}
	wg.Wait()
	return result
}

func findRemotesOfRepo(ctx context.Context, host string, path string) []string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if _, err := os.Stat(path); err != nil {
		return nil
	}
	output, err := exec.CommandContext(ctx, "git", "-C", path, "config", "--get-regexp", `^remote\..*\.url$`).Output()
	if err != nil {
		return nil
	}
	return parseRemoteURLs(host, string(output))
}

// parseRemoteURLs parses the output of `git config --get-regexp
// remote\..*\.url`, returning the full names of the GitHub repositories on
// host among the remote URLs, the origin's first.
func parseRemoteURLs(host string, output string) []string {
	names := []string{}
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		key, url, found := strings.Cut(line, " ")
		if !found {
			continue
		}
		owner, name, ok := github_commands.ParseRepositoryName(host, url)
		if !ok || !strings.Contains(url, host) {
			continue
		}
		fullName := owner + "/" + name
		if key == "remote.origin.url" {
			names = append([]string{fullName}, names...)
		} else {
			names = append(names, fullName)
		}
	}
	return lo.Uniq(names)
}
