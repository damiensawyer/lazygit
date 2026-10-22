package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGithubActionsState(t *testing.T) {
	scenarios := []struct {
		status     string
		conclusion string
		expected   GithubActionsState
	}{
		{"in_progress", "", GithubActionsStateInProgress},
		{"queued", "", GithubActionsStateQueued},
		{"waiting", "", GithubActionsStateQueued},
		{"completed", "success", GithubActionsStateSuccess},
		{"completed", "failure", GithubActionsStateFailure},
		{"completed", "timed_out", GithubActionsStateFailure},
		{"completed", "startup_failure", GithubActionsStateFailure},
		{"completed", "cancelled", GithubActionsStateCancelled},
		{"completed", "skipped", GithubActionsStateSkipped},
		{"completed", "neutral", GithubActionsStateNeutral},
	}

	for _, s := range scenarios {
		t.Run(s.status+"/"+s.conclusion, func(t *testing.T) {
			assert.Equal(t, s.expected, githubActionsState(s.status, s.conclusion))
		})
	}
}

func TestGithubRunDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := start.Add(time.Hour)

	completed := &GithubRun{
		Status:       "completed",
		Conclusion:   "success",
		RunStartedAt: start,
		UpdatedAt:    start.Add(90 * time.Second),
	}
	assert.Equal(t, 90*time.Second, completed.Duration(now))

	running := &GithubRun{
		Status:       "in_progress",
		RunStartedAt: start,
		UpdatedAt:    start.Add(time.Second),
	}
	assert.Equal(t, time.Hour, running.Duration(now))
}

func TestGithubRunTitle(t *testing.T) {
	run := &GithubRun{}
	run.HeadCommit.Message = "Fix the thing\n\nWith a body"
	assert.Equal(t, "Fix the thing", run.Title())

	run.DisplayTitle = "Pull request title"
	assert.Equal(t, "Pull request title", run.Title())
}
