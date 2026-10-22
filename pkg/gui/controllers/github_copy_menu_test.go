package controllers

import (
	"errors"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestJoinLogLines(t *testing.T) {
	scenarios := []struct {
		name     string
		lines    []string
		expected string
	}{
		{"one line", []string{"one"}, "one\n"},
		{"several lines", []string{"one", "", "three"}, "one\n\nthree\n"},
		{"unicode", []string{"✓ héllo", "🎉"}, "✓ héllo\n🎉\n"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, joinLogLines(s.lines))
		})
	}
}

func TestGithubCopyValue(t *testing.T) {
	item := githubCopyValue("Run URL", "https://github.com/owner/repo/actions/runs/1")
	assert.Equal(t, "Run URL", item.label)

	var copied []string
	assert.NoError(t, item.get(func(value string) error {
		copied = append(copied, value)
		return nil
	}))
	assert.Equal(t, []string{"https://github.com/owner/repo/actions/runs/1"}, copied)

	// The value can be produced more than once, e.g. when the menu is opened
	// again.
	assert.NoError(t, item.get(func(value string) error {
		copied = append(copied, value)
		return nil
	}))
	assert.Len(t, copied, 2)
}

func TestGithubCopyValuePassesOnErrors(t *testing.T) {
	item := githubCopyValue("Run ID", "42")
	err := errors.New("no clipboard")

	assert.Equal(t, err, item.get(func(string) error { return err }))
}

func TestRunPlaceholders(t *testing.T) {
	run := &models.GithubRun{Name: "CI", Number: 128}
	run.HeadCommit.Message = "Fix the thing\n\nbody"

	assert.Equal(t, map[string]string{
		"workflow": "CI",
		"number":   "128",
		"title":    "Fix the thing",
	}, runPlaceholders(run))

	run.DisplayTitle = "PR title"
	assert.Equal(t, "PR title", runPlaceholders(run)["title"])
}
