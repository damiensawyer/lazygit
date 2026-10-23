package git_commands

import (
	"fmt"
	"strings"
)

type CommitMessage struct {
	Subject string
	Body    string
}

// CommitMessagesBetween returns the messages of the commits on head that
// aren't on base, newest first.
func (self *GitHubCommands) CommitMessagesBetween(base string, head string) ([]CommitMessage, error) {
	cmdArgs := NewGitCmd("log").
		Arg("--format=%x1e%s%x00%b", base+".."+head, "--").
		ToArgv()
	output, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}

	messages := []CommitMessage{}
	for _, entry := range strings.Split(output, "\x1e") {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		subject, body, _ := strings.Cut(entry, "\x00")
		messages = append(messages, CommitMessage{Subject: subject, Body: strings.TrimSpace(body)})
	}
	return messages, nil
}

// DefaultPullRequestTitleAndBody suggests a title and description for a pull
// request of the given commits (newest first) the way `gh pr create` does:
// those of the commit if there is only one, and otherwise the branch's name
// and a list of the commits' subjects.
func DefaultPullRequestTitleAndBody(branchName string, commits []CommitMessage) (string, string) {
	if len(commits) == 1 {
		return commits[0].Subject, commits[0].Body
	}

	title := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return ' '
		}
		return r
	}, branchName)

	var body strings.Builder
	for i := len(commits) - 1; i >= 0; i-- {
		fmt.Fprintf(&body, "- **%s**\n", commits[i].Subject)
	}
	return title, body.String()
}
