package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/stretchr/testify/assert"
)

func TestDefaultPullRequestTitleAndBody(t *testing.T) {
	title, body := DefaultPullRequestTitleAndBody("add-widget", []CommitMessage{{Subject: "Add a widget", Body: "It's useful."}})
	assert.Equal(t, "Add a widget", title)
	assert.Equal(t, "It's useful.", body)

	title, body = DefaultPullRequestTitleAndBody("feature/add-a_widget", []CommitMessage{
		{Subject: "Test the widget"},
		{Subject: "Add a widget", Body: "It's useful."},
	})
	assert.Equal(t, "feature/add a widget", title)
	assert.Equal(t, "- **Add a widget**\n- **Test the widget**\n", body)
}

func TestCommitMessagesBetween(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"log", "--format=%x1e%s%x00%b", "origin/master..widget", "--"},
			"\x1eTest the widget\x00\n\x1eAdd a widget\x00It's useful.\n\nReally.\n\n", nil)
	instance := NewGitHubCommands(buildGitCommon(commonDeps{runner: runner}))

	messages, err := instance.CommitMessagesBetween("origin/master", "widget")
	assert.NoError(t, err)
	assert.Equal(t, []CommitMessage{
		{Subject: "Test the widget", Body: ""},
		{Subject: "Add a widget", Body: "It's useful.\n\nReally."},
	}, messages)
	runner.CheckForMissingCalls()
}
