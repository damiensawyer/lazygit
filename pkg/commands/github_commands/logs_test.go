package github_commands

import (
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func at(clock string) time.Time {
	t, err := time.Parse(time.RFC3339, "2026-10-21T"+clock+"Z")
	if err != nil {
		panic(err)
	}
	return t
}

func step(number int, name string, started, completed string) models.GithubStep {
	return models.GithubStep{
		Number:      number,
		Name:        name,
		Status:      "completed",
		Conclusion:  "success",
		StartedAt:   at(started),
		CompletedAt: at(completed),
	}
}

func TestSplitJobLog(t *testing.T) {
	// Modelled on a real log: steps share the second one ends and the next
	// starts in, and the post-run steps all start in the same second.
	log := "\ufeff2026-10-21T00:05:22.6292870Z Current runner version: '2.338.0'\n" +
		"2026-10-21T00:05:24.0112311Z Complete job name: lint\n" +
		"2026-10-21T00:05:24.1043776Z ##[group]Run actions/checkout@v7\n" +
		"2026-10-21T00:05:24.1043999Z with:\n" +
		"2026-10-21T00:05:25.0101010Z Checked out\n" +
		"2026-10-21T00:05:25.2329153Z ##[group]Run ./build.sh\n" +
		"2026-10-21T00:05:31.0000000Z ##[error]Process completed with exit code 1.\n" +
		"2026-10-21T00:06:02.7045705Z Post job cleanup.\n" +
		"2026-10-21T00:06:02.8045705Z Removing credentials\n" +
		"2026-10-21T00:06:03.0641344Z Cleaning up orphan processes\n"

	steps := []models.GithubStep{
		step(1, "Set up job", "00:05:22", "00:05:24"),
		step(2, "Checkout code", "00:05:24", "00:05:25"),
		step(3, "Build", "00:05:25", "00:05:31"),
		{Number: 4, Name: "Upload", Status: "completed", Conclusion: "skipped"},
		step(5, "Post Checkout code", "00:06:02", "00:06:03"),
		step(6, "Complete job", "00:06:03", "00:06:03"),
	}

	assert.Equal(t, []StepLog{
		{StepNumber: 1, Lines: []string{"Current runner version: '2.338.0'", "Complete job name: lint"}},
		{StepNumber: 2, Lines: []string{"##[group]Run actions/checkout@v7", "with:", "Checked out"}},
		{StepNumber: 3, Lines: []string{"##[group]Run ./build.sh", "##[error]Process completed with exit code 1."}},
		{StepNumber: 5, Lines: []string{"Post job cleanup.", "Removing credentials"}},
		{StepNumber: 6, Lines: []string{"Cleaning up orphan processes"}},
	}, SplitJobLog(log, steps))
}

func TestSplitJobLogMovesOnOnceAStepHasCompleted(t *testing.T) {
	// A step without any of the lines that usually open one still gets the
	// lines logged after the previous step completed.
	log := "2026-10-21T00:00:01.5Z one\n" +
		"2026-10-21T00:00:05.5Z three\n"

	steps := []models.GithubStep{
		step(1, "One", "00:00:01", "00:00:02"),
		step(2, "Two", "00:00:02", "00:00:03"),
		step(3, "Three", "00:00:04", "00:00:06"),
	}

	assert.Equal(t, []StepLog{
		{StepNumber: 1, Lines: []string{"one"}},
		{StepNumber: 2},
		{StepNumber: 3, Lines: []string{"three"}},
	}, SplitJobLog(log, steps))
}

func TestStripLogTimestamps(t *testing.T) {
	log := "\ufeff2026-10-21T00:05:22.6292870Z first\r\n" +
		"no timestamp\n" +
		"2026-10-21T00:05:22.6292870Z \n"

	assert.Equal(t, []string{"first", "no timestamp", ""}, StripLogTimestamps(log))
}
