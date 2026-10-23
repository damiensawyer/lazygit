package github_commands

import (
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

// GitHub serves logs with one at the start.
const byteOrderMark = "\ufeff"

// StepLog is the part of a job's log that one step produced.
type StepLog struct {
	StepNumber int
	Lines      []string
}

// SplitJobLog divides a job's log into the parts produced by each of its
// steps, with the timestamp each line starts with removed.
//
// The API serves a job's log as one text file, so the step boundaries have
// to be inferred. Each line is timestamped, and each step
// knows when it started and completed, but only to the second, and
// consecutive steps usually share the second one ends and the next begins
// in. So a line moves us on to a later step when it was logged after the
// current step completed, or when it is one of the lines the runner opens a
// step with.
func SplitJobLog(log string, steps []models.GithubStep) []StepLog {
	steps = stepsWithLogs(steps)
	if len(steps) == 0 {
		return nil
	}

	result := make([]StepLog, len(steps))
	for i, step := range steps {
		result[i].StepNumber = step.Number
	}

	current := 0
	for _, line := range logLines(log) {
		timestamp, text := splitLogTimestamp(line)
		if !timestamp.IsZero() {
			second := timestamp.Truncate(time.Second)
			for next := current + 1; next < len(steps); next++ {
				if steps[next].StartedAt.After(second) {
					break
				}
				if second.After(steps[current].CompletedAt) {
					current = next
					continue
				}
				if opensStep(text) {
					current = next
				}
				break
			}
		}
		result[current].Lines = append(result[current].Lines, text)
	}

	return result
}

// stepsWithLogs drops the steps that never ran, and so have nothing in the log.
func stepsWithLogs(steps []models.GithubStep) []models.GithubStep {
	result := make([]models.GithubStep, 0, len(steps))
	for _, step := range steps {
		if step.StartedAt.IsZero() || step.Conclusion == models.GithubConclusionSkipped {
			continue
		}
		result = append(result, step)
	}
	return result
}

// opensStep reports whether a log line is one the runner starts a step's
// output with.
func opensStep(text string) bool {
	return strings.HasPrefix(text, "##[group]Run ") ||
		text == "Post job cleanup." ||
		text == "Cleaning up orphan processes"
}

// splitLogTimestamp splits a log line into its leading RFC 3339 timestamp and
// the rest. Lines without one come back with a zero time.
func splitLogTimestamp(line string) (time.Time, string) {
	stamp, rest, found := strings.Cut(line, " ")
	if !found || !strings.HasSuffix(stamp, "Z") {
		return time.Time{}, line
	}
	t, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil {
		return time.Time{}, line
	}
	return t, rest
}

// StripLogTimestamps removes the timestamp from the start of every line of a
// job log.
func StripLogTimestamps(log string) []string {
	lines := logLines(log)
	for i, line := range lines {
		_, lines[i] = splitLogTimestamp(line)
	}
	return lines
}

// logLines splits a job log into its lines, without the line endings.
func logLines(log string) []string {
	log = strings.TrimPrefix(log, byteOrderMark)
	if log == "" {
		return nil
	}
	lines := strings.Split(strings.TrimSuffix(log, "\n"), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSuffix(line, "\r")
	}
	return lines
}

// JobLog is a job's log with the timestamps removed, and divided up into the
// parts each step produced where the job's steps are known.
type JobLog struct {
	Lines []string
	Steps []StepLog
}

func ParseJobLog(log string, steps []models.GithubStep) *JobLog {
	return &JobLog{
		Lines: StripLogTimestamps(log),
		Steps: SplitJobLog(log, steps),
	}
}
