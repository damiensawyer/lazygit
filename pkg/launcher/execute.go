package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// Executor carries out what the user chose in the picker, once it has closed.
// It runs git in the terminal the picker was in, so that git shows its
// progress, and can ask for an ssh passphrase or a password, as it would
// anywhere else.
type Executor struct {
	Tr *i18n.TranslationSet
	// Where the progress messages and git's output go.
	Out io.Writer
	// The protocol, "ssh" or "https", to clone with.
	GitProtocol string
	// How long to wait before retrying to clone a fork that isn't ready yet.
	ForkRetryDelay time.Duration
}

// Execute carries out choice, returning the directory of the repo to open.
func (self *Executor) Execute(ctx context.Context, choice Choice) (string, error) {
	switch choice.Kind {
	case ChoiceInit:
		args := []string{"init"}
		if choice.InitialBranch != "" {
			args = append(args, "--initial-branch="+choice.InitialBranch)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		return cwd, self.git(ctx, "", args...)
	case ChoiceOpen:
		return choice.Path, nil
	case ChoiceClone:
		return choice.Path, self.clone(ctx, choice)
	case ChoiceQuit:
	}
	return "", errors.New("nothing to do")
}

func (self *Executor) clone(ctx context.Context, choice Choice) error {
	repo := choice.Repo
	if choice.Fork {
		self.say(self.Tr.LauncherForking, map[string]string{"repo": repo.FullName})
		fork, err := choice.client.CreateFork(ctx, repo.Owner, repo.Name)
		if err != nil {
			return err
		}
		if fork.Parent == nil {
			fork.Parent = repo
		}
		repo = fork
	} else if repo.IsFork && repo.Parent == nil {
		// Searches don't say what a repository is a fork of.
		if detailed, err := choice.client.GetRepository(ctx, repo.Owner, repo.Name); err == nil {
			repo = detailed
		}
	}

	self.say(self.Tr.LauncherCloning, map[string]string{"repo": repo.FullName, "path": choice.Path})
	// A new fork's git data takes a few seconds to become available after
	// GitHub has said it created it, during which git fails with exit code
	// 128; gh retries it the same way.
	for attempt := 1; ; attempt++ {
		err := self.git(ctx, "", "clone", self.cloneURL(repo), choice.Path)
		var exitErr *exec.ExitError
		if err == nil || !choice.Fork || attempt == 3 || !errors.As(err, &exitErr) || exitErr.ExitCode() != 128 {
			if err != nil {
				return err
			}
			break
		}
		self.say(self.Tr.LauncherRetryingClone, map[string]string{"seconds": fmt.Sprint(int(self.ForkRetryDelay.Seconds()))})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(self.ForkRetryDelay):
		}
	}

	// Like `gh repo clone` does with a fork, add what it's a fork of as the
	// upstream remote, and tell gh, and our own pull request features, that
	// that's where pull requests go.
	if repo.Parent != nil {
		self.say(self.Tr.LauncherAddingUpstream, map[string]string{"repo": repo.Parent.FullName})
		if err := self.git(ctx, choice.Path, "remote", "add", "upstream", self.cloneURL(repo.Parent)); err != nil {
			return err
		}
		if err := self.git(ctx, choice.Path, "fetch", "upstream"); err != nil {
			return err
		}
		if err := self.git(ctx, choice.Path, "config", "--add", "remote.upstream.gh-resolved", "base"); err != nil {
			return err
		}
	}
	return nil
}

func (self *Executor) cloneURL(repo *models.GithubRepository) string {
	if self.GitProtocol == "ssh" && repo.SSHURL != "" {
		return repo.SSHURL
	}
	return repo.CloneURL
}

func (self *Executor) git(ctx context.Context, dir string, args ...string) error {
	if dir != "" {
		args = append([]string{"-C", dir}, args...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = self.Out
	cmd.Stderr = self.Out
	return cmd.Run()
}

func (self *Executor) say(message string, args map[string]string) {
	fmt.Fprintln(self.Out, utils.ResolvePlaceholderString(message, args))
}
