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

func TestGithubActionsStateOfEveryStatusAndConclusion(t *testing.T) {
	scenarios := []struct {
		status     string
		conclusion string
		expected   GithubActionsState
	}{
		{"pending", "", GithubActionsStateQueued},
		{"requested", "", GithubActionsStateQueued},
		// statuses we don't know about are assumed to be making progress
		{"", "", GithubActionsStateInProgress},
		{"some_new_status", "", GithubActionsStateInProgress},
		// the conclusion only counts once the status is completed
		{"in_progress", "success", GithubActionsStateInProgress},
		{"queued", "failure", GithubActionsStateQueued},
		{"completed", "action_required", GithubActionsStateFailure},
		{"completed", "stale", GithubActionsStateNeutral},
		{"completed", "", GithubActionsStateFailure},
		{"completed", "some_new_conclusion", GithubActionsStateFailure},
	}

	for _, s := range scenarios {
		t.Run(s.status+"/"+s.conclusion, func(t *testing.T) {
			assert.Equal(t, s.expected, githubActionsState(s.status, s.conclusion))
			assert.Equal(t, s.expected, (&GithubRun{Status: s.status, Conclusion: s.conclusion}).State())
			assert.Equal(t, s.expected, (&GithubJob{Status: s.status, Conclusion: s.conclusion}).State())
			assert.Equal(t, s.expected, (&GithubStep{Status: s.status, Conclusion: s.conclusion}).State())
		})
	}
}

func TestGithubActionsStatePredicates(t *testing.T) {
	scenarios := []struct {
		state    GithubActionsState
		isFailed bool
		isActive bool
	}{
		{GithubActionsStateInProgress, false, true},
		{GithubActionsStateQueued, false, true},
		{GithubActionsStateSuccess, false, false},
		{GithubActionsStateFailure, true, false},
		{GithubActionsStateCancelled, true, false},
		{GithubActionsStateSkipped, false, false},
		{GithubActionsStateNeutral, false, false},
	}

	for _, s := range scenarios {
		assert.Equal(t, s.isFailed, s.state.IsFailed(), "IsFailed of %d", s.state)
		assert.Equal(t, s.isActive, s.state.IsActive(), "IsActive of %d", s.state)
	}
}

func TestElapsed(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := start.Add(time.Hour)

	scenarios := []struct {
		name     string
		start    time.Time
		end      time.Time
		expected time.Duration
	}{
		{"not started", time.Time{}, start, 0},
		{"not started or ended", time.Time{}, time.Time{}, 0},
		{"ended", start, start.Add(90 * time.Second), 90 * time.Second},
		{"still going", start, time.Time{}, time.Hour},
		{"truncated to seconds", start, start.Add(1999 * time.Millisecond), time.Second},
		{"zero length", start, start, 0},
		{
			"ended before it started", start, start.Add(-time.Second),
			/* EXPECTED:
			0,
			ACTUAL: */
			time.Hour,
		},
		{
			"started after now", now.Add(time.Second),
			time.Time{},
			/* EXPECTED:
			0,
			ACTUAL: */
			-time.Second,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, elapsed(s.start, s.end, now))
		})
	}
}

func TestGithubRunDurationFallsBackToCreation(t *testing.T) {
	created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	run := &GithubRun{
		Status:     "completed",
		Conclusion: "failure",
		CreatedAt:  created,
		UpdatedAt:  created.Add(2 * time.Minute),
	}
	assert.Equal(t, 2*time.Minute, run.Duration(created.Add(time.Hour)))
}

func TestGithubRunDurationWhileQueued(t *testing.T) {
	created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	run := &GithubRun{
		Status:    "queued",
		CreatedAt: created,
		UpdatedAt: created,
	}
	assert.Equal(t, 5*time.Minute, run.Duration(created.Add(5*time.Minute)))
}

func TestGithubJobAndStepDuration(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := start.Add(time.Hour)

	assert.Equal(t, 3*time.Minute, (&GithubJob{StartedAt: start, CompletedAt: start.Add(3 * time.Minute)}).Duration(now))
	assert.Equal(t, time.Hour, (&GithubJob{StartedAt: start}).Duration(now))
	assert.Equal(t, time.Duration(0), (&GithubJob{}).Duration(now))

	assert.Equal(t, 4*time.Second, (&GithubStep{StartedAt: start, CompletedAt: start.Add(4 * time.Second)}).Duration(now))
	assert.Equal(t, time.Hour, (&GithubStep{StartedAt: start}).Duration(now))
	assert.Equal(t, time.Duration(0), (&GithubStep{}).Duration(now))
}

func TestGithubItemIdsAndDescriptions(t *testing.T) {
	run := &GithubRun{RunId: 1234567890123, DisplayTitle: "Run title"}
	assert.Equal(t, "1234567890123", run.ID())
	assert.Equal(t, "Run title", run.Description())

	job := &GithubJob{JobId: 42, Name: "test (ubuntu-latest)"}
	assert.Equal(t, "42", job.ID())
	assert.Equal(t, "test (ubuntu-latest)", job.Description())

	step := &GithubStep{Number: 3, Name: "Run tests"}
	assert.Equal(t, "3", step.ID())
	assert.Equal(t, "Run tests", step.Description())
}

func TestGithubRunTitleWithoutCommitMessage(t *testing.T) {
	assert.Equal(t, "", (&GithubRun{}).Title())

	run := &GithubRun{}
	run.HeadCommit.Message = "Subject only"
	assert.Equal(t, "Subject only", run.Title())

	run.HeadCommit.Message = "\nBody after an empty subject"
	assert.Equal(t, "", run.Title())
}

func TestGithubWorkflowIsActive(t *testing.T) {
	scenarios := []struct {
		state    string
		expected bool
	}{
		{"active", true},
		{"disabled_manually", false},
		{"disabled_inactivity", false},
		{"disabled_fork", false},
		{"deleted", false},
		{"", false},
	}

	for _, s := range scenarios {
		assert.Equal(t, s.expected, (&GithubWorkflow{State: s.state}).IsActive(), s.state)
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
