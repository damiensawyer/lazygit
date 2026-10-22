package models

import (
	"fmt"
	"strings"
	"time"
)

// The REST API reports a workflow run, job, or step through two fields: a
// status that says how far along it is, and, once the status is "completed", a
// conclusion that says how it ended.
const (
	GithubStatusCompleted = "completed"
	GithubStatusQueued    = "queued"
	GithubStatusWaiting   = "waiting"
	GithubStatusPending   = "pending"
	GithubStatusRequested = "requested"

	GithubConclusionSuccess        = "success"
	GithubConclusionFailure        = "failure"
	GithubConclusionCancelled      = "cancelled"
	GithubConclusionSkipped        = "skipped"
	GithubConclusionNeutral        = "neutral"
	GithubConclusionTimedOut       = "timed_out"
	GithubConclusionActionRequired = "action_required"
	GithubConclusionStartupFailure = "startup_failure"
	GithubConclusionStale          = "stale"
)

// GithubActionsState collapses a status/conclusion pair into the handful of
// outcomes worth distinguishing when displaying it.
type GithubActionsState int

const (
	GithubActionsStateInProgress GithubActionsState = iota
	GithubActionsStateQueued
	GithubActionsStateSuccess
	GithubActionsStateFailure
	GithubActionsStateCancelled
	GithubActionsStateSkipped
	GithubActionsStateNeutral
)

func githubActionsState(status, conclusion string) GithubActionsState {
	switch status {
	case GithubStatusCompleted:
	case GithubStatusQueued, GithubStatusWaiting, GithubStatusPending, GithubStatusRequested:
		return GithubActionsStateQueued
	default:
		return GithubActionsStateInProgress
	}

	switch conclusion {
	case GithubConclusionSuccess:
		return GithubActionsStateSuccess
	case GithubConclusionCancelled:
		return GithubActionsStateCancelled
	case GithubConclusionSkipped:
		return GithubActionsStateSkipped
	case GithubConclusionNeutral, GithubConclusionStale:
		return GithubActionsStateNeutral
	default:
		return GithubActionsStateFailure
	}
}

// IsFailed reports whether the state is one that someone should look at:
// a failure, or a cancellation.
func (self GithubActionsState) IsFailed() bool {
	return self == GithubActionsStateFailure || self == GithubActionsStateCancelled
}

func (self GithubActionsState) IsActive() bool {
	return self == GithubActionsStateInProgress || self == GithubActionsStateQueued
}

// elapsed returns how long something that started at `start` ran, ending at
// `end`, or at `now` if it hasn't ended yet.
func elapsed(start, end, now time.Time) time.Duration {
	if start.IsZero() {
		return 0
	}
	if end.IsZero() || end.Before(start) {
		end = now
	}
	return end.Sub(start).Truncate(time.Second)
}

// GithubRun is a workflow run.
type GithubRun struct {
	RunId        int64      `json:"id"`
	Name         string     `json:"name"` // the workflow's name
	DisplayTitle string     `json:"display_title"`
	Number       int64      `json:"run_number"`
	Attempt      int        `json:"run_attempt"`
	Event        string     `json:"event"`
	Status       string     `json:"status"`
	Conclusion   string     `json:"conclusion"`
	WorkflowId   int64      `json:"workflow_id"`
	HeadBranch   string     `json:"head_branch"`
	HeadSha      string     `json:"head_sha"`
	Path         string     `json:"path"` // e.g. ".github/workflows/ci.yml"
	Url          string     `json:"html_url"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	RunStartedAt time.Time  `json:"run_started_at"`
	Actor        GithubUser `json:"actor"`
	HeadCommit   struct {
		Message string `json:"message"`
	} `json:"head_commit"`
}

type GithubUser struct {
	Login string `json:"login"`
}

func (self *GithubRun) ID() string {
	return fmt.Sprintf("%d", self.RunId)
}

func (self *GithubRun) Description() string {
	return self.Title()
}

func (self *GithubRun) State() GithubActionsState {
	return githubActionsState(self.Status, self.Conclusion)
}

// Title is what GitHub's web UI shows as the run's headline: usually the
// commit subject or the PR title.
func (self *GithubRun) Title() string {
	if self.DisplayTitle != "" {
		return self.DisplayTitle
	}
	subject, _, _ := strings.Cut(self.HeadCommit.Message, "\n")
	return subject
}

// Duration is the time from the run's (latest attempt's) start until its last
// update, or until now while it's still going.
func (self *GithubRun) Duration(now time.Time) time.Duration {
	start := self.RunStartedAt
	if start.IsZero() {
		start = self.CreatedAt
	}
	end := self.UpdatedAt
	if self.State().IsActive() {
		end = time.Time{}
	}
	return elapsed(start, end, now)
}

// GithubJob is a job within a workflow run, with its steps.
type GithubJob struct {
	JobId       int64        `json:"id"`
	RunId       int64        `json:"run_id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	Conclusion  string       `json:"conclusion"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at"`
	Url         string       `json:"html_url"`
	RunnerName  string       `json:"runner_name"`
	Labels      []string     `json:"labels"`
	Steps       []GithubStep `json:"steps"`
}

func (self *GithubJob) ID() string {
	return fmt.Sprintf("%d", self.JobId)
}

func (self *GithubJob) Description() string {
	return self.Name
}

func (self *GithubJob) State() GithubActionsState {
	return githubActionsState(self.Status, self.Conclusion)
}

func (self *GithubJob) Duration(now time.Time) time.Duration {
	return elapsed(self.StartedAt, self.CompletedAt, now)
}

// GithubStep is a step within a job.
type GithubStep struct {
	Number      int       `json:"number"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Conclusion  string    `json:"conclusion"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
}

func (self *GithubStep) ID() string {
	return fmt.Sprintf("%d", self.Number)
}

func (self *GithubStep) Description() string {
	return self.Name
}

func (self *GithubStep) State() GithubActionsState {
	return githubActionsState(self.Status, self.Conclusion)
}

func (self *GithubStep) Duration(now time.Time) time.Duration {
	return elapsed(self.StartedAt, self.CompletedAt, now)
}

// GithubWorkflow is a workflow defined in the repository.
type GithubWorkflow struct {
	WorkflowId int64  `json:"id"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	State      string `json:"state"` // "active", "disabled_manually", ...
	Url        string `json:"html_url"`
}

func (self *GithubWorkflow) IsActive() bool {
	return self.State == "active"
}

// GithubArtifact is a file archive uploaded by a workflow run.
type GithubArtifact struct {
	ArtifactId  int64  `json:"id"`
	Name        string `json:"name"`
	SizeInBytes int64  `json:"size_in_bytes"`
	Expired     bool   `json:"expired"`
}

// GithubAnnotation is an error, warning, or notice reported by a job, such as
// a compiler error or a failed test, optionally pointing at a file location.
type GithubAnnotation struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	Level     string `json:"annotation_level"` // "failure", "warning", or "notice"
	Title     string `json:"title"`
	Message   string `json:"message"`
}
