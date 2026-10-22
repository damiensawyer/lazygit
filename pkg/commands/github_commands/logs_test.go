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

func TestSplitJobLogEdgeCases(t *testing.T) {
	scenarios := []struct {
		name     string
		log      string
		steps    []models.GithubStep
		expected []StepLog
	}{
		{
			name:     "no steps",
			log:      "2026-10-21T00:00:01.5Z one\n",
			steps:    nil,
			expected: nil,
		},
		{
			name: "only steps that never ran",
			log:  "2026-10-21T00:00:01.5Z one\n",
			steps: []models.GithubStep{
				{Number: 1, Name: "Skipped", Status: "completed", Conclusion: "skipped", StartedAt: at("00:00:01"), CompletedAt: at("00:00:01")},
				{Number: 2, Name: "Queued", Status: "queued"},
			},
			expected: nil,
		},
		{
			name:     "single step without a trailing newline",
			log:      "2026-10-21T00:00:01.5Z one\n2026-10-21T00:00:01.6Z two",
			steps:    []models.GithubStep{step(1, "One", "00:00:01", "00:00:02")},
			expected: []StepLog{{StepNumber: 1, Lines: []string{"one", "two"}}},
		},
		{
			name: "CRLF line endings and byte order mark",
			log: byteOrderMark + "2026-10-21T00:00:01.5Z one\r\n" +
				"2026-10-21T00:00:03.5Z two\r\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one"}},
				{StepNumber: 2, Lines: []string{"two"}},
			},
		},
		{
			name: "unicode",
			log: "2026-10-21T00:00:01.5Z ✓ tests passed 🎉\n" +
				"2026-10-21T00:00:03.5Z 日本語のログ\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"✓ tests passed 🎉"}},
				{StepNumber: 2, Lines: []string{"日本語のログ"}},
			},
		},
		{
			name: "lines without a timestamp stay with the step before them",
			log: "2026-10-21T00:00:01.5Z one\n" +
				"continued\n" +
				"2026-10-21T00:00:03.5Z two\n" +
				"also continued\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one", "continued"}},
				{StepNumber: 2, Lines: []string{"two", "also continued"}},
			},
		},
		{
			name: "invalid timestamps are part of the text",
			log: "2026-10-21T00:00:01.5Z one\n" +
				"2026-13-45T00:00:03.5Z not a date\n" +
				"2026-10-21T00:00:03.5Z two\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one", "2026-13-45T00:00:03.5Z not a date"}},
				{StepNumber: 2, Lines: []string{"two"}},
			},
		},
		{
			name: "groups other than a step's Run don't open a step",
			// The set-up step logs groups of its own, in the second the next
			// step starts in.
			log: "2026-10-21T00:00:01.1Z ##[group]Operating System\n" +
				"2026-10-21T00:00:01.2Z Ubuntu\n" +
				"2026-10-21T00:00:01.3Z ##[endgroup]\n" +
				"2026-10-21T00:00:01.4Z ##[group]Run make\n" +
				"2026-10-21T00:00:01.5Z ##[endgroup]\n" +
				"2026-10-21T00:00:01.6Z make: Nothing to be done\n",
			steps: []models.GithubStep{
				step(1, "Set up job", "00:00:01", "00:00:01"),
				step(2, "Build", "00:00:01", "00:00:02"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"##[group]Operating System", "Ubuntu", "##[endgroup]"}},
				{StepNumber: 2, Lines: []string{"##[group]Run make", "##[endgroup]", "make: Nothing to be done"}},
			},
		},
		{
			name: "lines logged before the first step started belong to it",
			log:  "2026-10-21T00:00:00.5Z early\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
			},
			expected: []StepLog{{StepNumber: 1, Lines: []string{"early"}}},
		},
		{
			name: "lines logged after the last step completed belong to it",
			log: "2026-10-21T00:00:01.5Z one\n" +
				"2026-10-21T00:00:09.5Z late\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one"}},
				{StepNumber: 2, Lines: []string{"late"}},
			},
		},
		{
			name: "several steps without output in a row",
			log: "2026-10-21T00:00:01.5Z one\n" +
				"2026-10-21T00:00:09.5Z four\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
				step(3, "Three", "00:00:05", "00:00:06"),
				step(4, "Four", "00:00:09", "00:00:10"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one"}},
				{StepNumber: 2},
				{StepNumber: 3},
				{StepNumber: 4, Lines: []string{"four"}},
			},
		},
		{
			name: "empty lines within a step are kept",
			log: "2026-10-21T00:00:01.5Z one\n" +
				"2026-10-21T00:00:01.6Z \n" +
				"2026-10-21T00:00:01.7Z three\n",
			steps:    []models.GithubStep{step(1, "One", "00:00:01", "00:00:02")},
			expected: []StepLog{{StepNumber: 1, Lines: []string{"one", "", "three"}}},
		},
		{
			name: "log ending in an earlier step",
			log:  "2026-10-21T00:00:01.5Z one\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:05", "00:00:06"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one"}},
				{StepNumber: 2},
			},
		},
		{
			name: "log without timestamps",
			log:  "one\ntwo\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:05", "00:00:06"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one", "two"}},
				{StepNumber: 2},
			},
		},
		{
			name: "timestamps without fractional seconds",
			log: "2026-10-21T00:00:01Z one\n" +
				"2026-10-21T00:00:03Z two\n",
			steps: []models.GithubStep{
				step(1, "One", "00:00:01", "00:00:02"),
				step(2, "Two", "00:00:03", "00:00:04"),
			},
			expected: []StepLog{
				{StepNumber: 1, Lines: []string{"one"}},
				{StepNumber: 2, Lines: []string{"two"}},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, SplitJobLog(s.log, s.steps))
		})
	}
}

func TestParseJobLog(t *testing.T) {
	log := "2026-10-21T00:00:01.5Z one\n" +
		"2026-10-21T00:00:03.5Z ##[group]Run two\n"
	steps := []models.GithubStep{
		step(1, "One", "00:00:01", "00:00:03"),
		step(2, "Two", "00:00:03", "00:00:04"),
	}

	assert.Equal(t, &JobLog{
		Lines: []string{"one", "##[group]Run two"},
		Steps: []StepLog{
			{StepNumber: 1, Lines: []string{"one"}},
			{StepNumber: 2, Lines: []string{"##[group]Run two"}},
		},
	}, ParseJobLog(log, steps))

	assert.Equal(t, &JobLog{Lines: []string{"one", "##[group]Run two"}}, ParseJobLog(log, nil))
}

func TestStripLogTimestampsEdgeCases(t *testing.T) {
	scenarios := []struct {
		name     string
		log      string
		expected []string
	}{
		{
			name:     "no trailing newline",
			log:      "2026-10-21T00:00:01.5Z one\n2026-10-21T00:00:01.5Z two",
			expected: []string{"one", "two"},
		},
		{
			name:     "blank lines at the end are kept but the final newline isn't",
			log:      "2026-10-21T00:00:01.5Z one\n\n",
			expected: []string{"one", ""},
		},
		{
			name:     "unicode",
			log:      "2026-10-21T00:00:01.5Z ✓ héllo 🎉\n",
			expected: []string{"✓ héllo 🎉"},
		},
		{
			name:     "only the first timestamp of a line is removed",
			log:      "2026-10-21T00:00:01.5Z 2026-10-21T00:00:01.5Z quoted\n",
			expected: []string{"2026-10-21T00:00:01.5Z quoted"},
		},
		{
			name:     "first word that isn't a UTC timestamp",
			log:      "2026-10-21T00:00:01+00:00 offset\nhello world\nREADME.Z file\n",
			expected: []string{"2026-10-21T00:00:01+00:00 offset", "hello world", "README.Z file"},
		},
		{
			name:     "CR in the middle of a line is kept",
			log:      "2026-10-21T00:00:01.5Z progress 10%\rprogress 100%\r\n",
			expected: []string{"progress 10%\rprogress 100%"},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, StripLogTimestamps(s.log))
		})
	}
}

func TestStripLogTimestamps(t *testing.T) {
	log := "\ufeff2026-10-21T00:05:22.6292870Z first\r\n" +
		"no timestamp\n" +
		"2026-10-21T00:05:22.6292870Z \n"

	assert.Equal(t, []string{"first", "no timestamp", ""}, StripLogTimestamps(log))
}
