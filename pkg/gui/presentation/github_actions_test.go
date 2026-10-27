package presentation

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/xo/terminfo"
)

func TestFormatGithubDuration(t *testing.T) {
	assert.Equal(t, "0s", FormatGithubDuration(0))
	assert.Equal(t, "59s", FormatGithubDuration(59*time.Second+900*time.Millisecond))
	assert.Equal(t, "4m12s", FormatGithubDuration(4*time.Minute+12*time.Second))
	assert.Equal(t, "2h3m", FormatGithubDuration(2*time.Hour+3*time.Minute+59*time.Second))
}

func TestFormatGithubRunSummary(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	run := &models.GithubRun{
		Name:         "CI",
		DisplayTitle: "Fix the flaky test",
		Number:       12,
		Attempt:      2,
		Event:        "push",
		Status:       "completed",
		Conclusion:   "failure",
		HeadBranch:   "main",
		HeadSha:      "0123456789abcdef",
		Path:         ".github/workflows/ci.yml",
		Url:          "https://github.com/owner/repo/actions/runs/1",
		CreatedAt:    start,
		RunStartedAt: start,
		UpdatedAt:    start.Add(3*time.Minute + 5*time.Second),
		Actor:        models.GithubUser{Login: "octocat"},
	}
	jobs := []*models.GithubJob{
		{
			JobId: 1, Name: "lint", Status: "completed", Conclusion: "success",
			StartedAt: start, CompletedAt: start.Add(time.Minute),
			Steps: []models.GithubStep{{Name: "Run linter", Status: "completed", Conclusion: "success"}},
		},
		{
			JobId: 2, Name: "test", Status: "completed", Conclusion: "failure",
			StartedAt: start, CompletedAt: start.Add(3 * time.Minute),
			Steps: []models.GithubStep{
				{Name: "Set up job", Status: "completed", Conclusion: "success"},
				{Name: "Run tests", Status: "completed", Conclusion: "failure"},
				{Name: "Upload", Status: "completed", Conclusion: "skipped"},
			},
		},
	}
	annotations := map[int64][]*models.GithubAnnotation{
		2: {{Path: "pkg/foo_test.go", StartLine: 42, Level: "failure", Message: "expected 1\ngot 2"}},
	}

	summary := FormatGithubRunSummary(i18n.EnglishTranslationSet(), GithubRunSummaryOpts{
		Run: run, Jobs: jobs, JobsLoaded: true, Annotations: annotations, Now: start.Add(time.Hour),
	})

	assert.Equal(t, `X Fix the flaky test
CI #12 · main 01234567 · .github/workflows/ci.yml
Triggered via push by octocat 1h ago · 3m5s · attempt 2
https://github.com/owner/repo/actions/runs/1

Jobs
✓ lint 1m0s
X test 3m0s
  ✓ Set up job
  X Run tests

Annotations
test
  X expected 1
    got 2
    pkg/foo_test.go#42
`, summary)
}

func TestFormatGithubRunSummaryWhileLoading(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	run := &models.GithubRun{Name: "CI", DisplayTitle: "Title", Status: "queued"}
	tr := i18n.EnglishTranslationSet()

	assert.Contains(t, FormatGithubRunSummary(tr, GithubRunSummaryOpts{Run: run}), "Jobs\nLoading jobs…\n")
	assert.Contains(t, FormatGithubRunSummary(tr, GithubRunSummaryOpts{Run: run, JobsError: errors.New("boom")}), "Jobs\nboom\n")
}

func TestFormatGithubLogLines(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	assert.Equal(t,
		"▾ Run make\n/usr/bin/make\nmake: *** Error 1\nError: Process completed with exit code 2.\n",
		FormatGithubLogLines([]string{
			"##[group]Run make",
			"[command]/usr/bin/make",
			"make: *** Error 1",
			"##[endgroup]",
			"##[error]Process completed with exit code 2.",
		}),
	)
}

func TestFormatGithubJobDetailsShowsOnlyFailedStepsOfFailedJob(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	job := &models.GithubJob{
		Name: "test", Status: "completed", Conclusion: "failure",
		StartedAt: start, CompletedAt: start.Add(2 * time.Minute),
		Url: "https://github.com/owner/repo/actions/runs/1/job/2",
		Steps: []models.GithubStep{
			{Number: 1, Name: "Set up job", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start.Add(time.Second)},
			{Number: 2, Name: "Run tests", Status: "completed", Conclusion: "failure", StartedAt: start.Add(time.Second), CompletedAt: start.Add(2 * time.Minute)},
		},
	}
	log := &github_commands.JobLog{
		Steps: []github_commands.StepLog{
			{StepNumber: 1, Lines: []string{"setting up"}},
			{StepNumber: 2, Lines: []string{"FAIL: TestFoo", "##[error]Process completed with exit code 1."}},
		},
	}

	details := FormatGithubJobDetails(i18n.EnglishTranslationSet(), GithubJobDetailsOpts{
		Job: job, Log: log, LogLoaded: true, Now: start.Add(time.Hour),
	})

	assert.Equal(t, `X test 2m0s
https://github.com/owner/repo/actions/runs/1/job/2

Steps
✓ Set up job 1s
X Run tests 1m59s

Log of failed steps
━━ Run tests
FAIL: TestFoo
Error: Process completed with exit code 1.
`, details)
}

func TestFormatGithubJobDetailsOfJobInProgress(t *testing.T) {
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	defer color.ForceSetColorLevel(oldColorLevel)

	job := &models.GithubJob{Name: "test", Status: "in_progress", StartedAt: time.Now()}

	details := FormatGithubJobDetails(i18n.EnglishTranslationSet(), GithubJobDetailsOpts{Job: job, Now: time.Now()})

	assert.Contains(t, details, "Log\nThe log will be available once the job has completed\n")
}

func withoutColors(t *testing.T) {
	t.Helper()
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelNone)
	t.Cleanup(func() { color.ForceSetColorLevel(oldColorLevel) })
}

func withBasicColors(t *testing.T) {
	t.Helper()
	oldColorLevel := color.ForceSetColorLevel(terminfo.ColorLevelBasic)
	t.Cleanup(func() { color.ForceSetColorLevel(oldColorLevel) })
}

func TestGithubActionsStateIcon(t *testing.T) {
	withBasicColors(t)

	scenarios := []struct {
		state    models.GithubActionsState
		expected string
	}{
		{models.GithubActionsStateSuccess, "\x1b[32m✓\x1b[0m"},
		{models.GithubActionsStateFailure, "\x1b[31mX\x1b[0m"},
		{models.GithubActionsStateCancelled, "\x1b[33mX\x1b[0m"},
		{models.GithubActionsStateSkipped, "\x1b[90m-\x1b[0m"},
		{models.GithubActionsStateNeutral, "\x1b[90m-\x1b[0m"},
		{models.GithubActionsStateQueued, "\x1b[33m•\x1b[0m"},
		{models.GithubActionsStateInProgress, "\x1b[33m*\x1b[0m"},
		{models.GithubActionsState(99), "?"},
	}

	for _, s := range scenarios {
		assert.Equal(t, s.expected, GithubActionsStateIcon(s.state), "state %d", s.state)
	}
}

func TestFormatGithubDurationBoundaries(t *testing.T) {
	scenarios := []struct {
		duration time.Duration
		expected string
	}{
		{999 * time.Millisecond, "0s"},
		{time.Second, "1s"},
		{time.Minute, "1m0s"},
		{time.Hour - time.Second, "59m59s"},
		{time.Hour, "1h0m"},
		{25*time.Hour + 30*time.Minute, "25h30m"},
	}

	for _, s := range scenarios {
		assert.Equal(t, s.expected, FormatGithubDuration(s.duration), s.duration.String())
	}
}

func TestGetGithubRunListDisplayStrings(t *testing.T) {
	withoutColors(t)

	now := time.Now()
	run := &models.GithubRun{
		Name:         "A workflow with a very long name",
		HeadBranch:   "main",
		Event:        "pull_request",
		Status:       "completed",
		Conclusion:   "success",
		CreatedAt:    now.Add(-3 * time.Hour),
		RunStartedAt: now.Add(-3 * time.Hour),
		UpdatedAt:    now.Add(-3*time.Hour + 4*time.Minute + 12*time.Second),
	}
	run.HeadCommit.Message = "Fix the thing\n\nbody"

	tr := i18n.EnglishTranslationSet()

	assert.Equal(t, [][]string{
		{"✓", "Succeeded", "3h", "A workflow with a v…", "Fix the thing"},
	}, GetGithubRunListDisplayStrings(tr, []*models.GithubRun{run}, false, now))

	assert.Equal(t, [][]string{
		{"✓", "Succeeded", "3h", "A workflow with a very long name", "main", "pull_request", "4m12s", "Fix the thing"},
	}, GetGithubRunListDisplayStrings(tr, []*models.GithubRun{run}, true, now))
}

func TestGetGithubRunListDisplayStringsDoesNotTruncateShortNames(t *testing.T) {
	withoutColors(t)

	now := time.Now()
	runs := []*models.GithubRun{
		{Name: "Exactly twenty chars", DisplayTitle: "One", Status: "in_progress", CreatedAt: now.Add(-30 * time.Minute)},
		{Name: "CI", DisplayTitle: "Two", Status: "queued", CreatedAt: now.Add(-2 * 24 * time.Hour)},
	}

	assert.Equal(t, [][]string{
		{"*", "In progress", "30m", "Exactly twenty chars", "One"},
		{"•", "Queued", "2d", "CI", "Two"},
	}, GetGithubRunListDisplayStrings(i18n.EnglishTranslationSet(), runs, false, now))
}

func TestGetGithubRunListDisplayStringsMeasuresAgeToNow(t *testing.T) {
	withoutColors(t)

	created := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	run := &models.GithubRun{
		Name: "CI", DisplayTitle: "One", HeadBranch: "main", Event: "push", Status: "in_progress",
		CreatedAt: created, RunStartedAt: created,
	}

	assert.Equal(t, [][]string{
		{"*", "In progress", "5m", "CI", "main", "push", "5m0s", "One"},
	}, GetGithubRunListDisplayStrings(i18n.EnglishTranslationSet(), []*models.GithubRun{run}, true, created.Add(5*time.Minute)))
}

func TestGetGithubRunListDisplayStringsIsEmptyForNoRuns(t *testing.T) {
	assert.Empty(t, GetGithubRunListDisplayStrings(i18n.EnglishTranslationSet(), nil, false, time.Now()))
}

func TestGetGithubRunListDisplayStringsColors(t *testing.T) {
	withBasicColors(t)

	now := time.Now()
	run := &models.GithubRun{
		Name: "CI", DisplayTitle: "Title", HeadBranch: "main", Event: "push",
		Status: "completed", Conclusion: "failure",
		CreatedAt: now.Add(-time.Minute), RunStartedAt: now.Add(-time.Minute), UpdatedAt: now,
	}

	assert.Equal(t, [][]string{{
		"\x1b[31mX\x1b[0m",
		"\x1b[31mFailed\x1b[0m",
		"\x1b[34m1m\x1b[0m",
		"\x1b[36mCI\x1b[0m",
		"\x1b[35mmain\x1b[0m",
		"\x1b[90mpush\x1b[0m",
		"1m0s",
		"Title",
	}}, GetGithubRunListDisplayStrings(i18n.EnglishTranslationSet(), []*models.GithubRun{run}, true, now))
}

func TestGetGithubJobAndStepListDisplayStrings(t *testing.T) {
	withoutColors(t)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	now := start.Add(10 * time.Minute)

	jobs := []*models.GithubJob{
		{Name: "build", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start.Add(75 * time.Second)},
		{Name: "test", Status: "in_progress", StartedAt: start.Add(time.Minute)},
		{Name: "deploy", Status: "queued"},
		{Name: "docs", Status: "completed", Conclusion: "skipped"},
		{Name: "lint", Status: "completed", Conclusion: "cancelled", StartedAt: start, CompletedAt: start.Add(time.Second)},
	}
	tr := i18n.EnglishTranslationSet()

	assert.Equal(t, [][]string{
		{"✓", "Succeeded", "build", "1m15s"},
		{"*", "In progress", "test", "9m0s"},
		{"•", "Queued", "deploy", ""},
		{"-", "Skipped", "docs", ""},
		{"X", "Cancelled", "lint", "1s"},
	}, GetGithubJobListDisplayStrings(tr, jobs, now))

	steps := []*models.GithubStep{
		{Name: "Set up job", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start.Add(2 * time.Second)},
		{Name: "Run tests", Status: "in_progress", StartedAt: start.Add(2 * time.Second)},
		{Name: "Upload", Status: "pending"},
		{Name: "Notify", Status: "completed", Conclusion: "skipped"},
		{Name: "Neutral", Status: "completed", Conclusion: "neutral", StartedAt: start, CompletedAt: start},
	}
	assert.Equal(t, [][]string{
		{"✓", "Succeeded", "Set up job", "2s"},
		{"*", "In progress", "Run tests", "9m58s"},
		{"•", "Queued", "Upload", ""},
		{"-", "Skipped", "Notify", ""},
		{"-", "Neutral", "Neutral", "0s"},
	}, GetGithubStepListDisplayStrings(tr, steps, now))
}

func TestFormatGithubRunSummaryOfRunInProgress(t *testing.T) {
	withoutColors(t)

	start := time.Now().Add(-5 * time.Minute)
	run := &models.GithubRun{
		Name: "CI", DisplayTitle: "Add a feature", Number: 3, Attempt: 1,
		Event: "pull_request", Status: "in_progress", HeadBranch: "feature", HeadSha: "abc",
		Path: ".github/workflows/ci.yml", Url: "https://github.com/o/r/actions/runs/3",
		CreatedAt: start, RunStartedAt: start, UpdatedAt: start.Add(time.Minute),
		Actor: models.GithubUser{Login: "hubot"},
	}
	jobs := []*models.GithubJob{
		{
			JobId: 1, Name: "test", Status: "in_progress", StartedAt: start,
			Steps: []models.GithubStep{
				{Name: "Set up job", Status: "completed", Conclusion: "success"},
				{Name: "Run tests", Status: "in_progress"},
				{Name: "Upload", Status: "queued"},
			},
		},
		{JobId: 2, Name: "deploy", Status: "queued"},
	}

	summary := FormatGithubRunSummary(i18n.EnglishTranslationSet(), GithubRunSummaryOpts{
		Run: run, Jobs: jobs, JobsLoaded: true, Now: start.Add(5 * time.Minute),
	})

	// The first attempt isn't worth mentioning, and a run without annotations
	// has no section for them.
	assert.Equal(t, `* Add a feature
CI #3 · feature abc · .github/workflows/ci.yml
Triggered via pull_request by hubot 5m ago · 5m0s
https://github.com/o/r/actions/runs/3

Jobs
* test 5m0s
  ✓ Set up job
  * Run tests
  • Upload
• deploy
`, summary)
}

func TestFormatGithubRunSummaryWithoutJobs(t *testing.T) {
	withoutColors(t)

	run := &models.GithubRun{Name: "CI", DisplayTitle: "Title", Status: "completed", Conclusion: "startup_failure"}

	summary := FormatGithubRunSummary(i18n.EnglishTranslationSet(), GithubRunSummaryOpts{Run: run, JobsLoaded: true})

	assert.Contains(t, summary, "X Title\n")
	assert.Contains(t, summary, "Jobs\nThis run has no jobs\n")
}

func TestFormatGithubRunSummaryShowsAnnotationsInJobOrder(t *testing.T) {
	withoutColors(t)

	run := &models.GithubRun{Name: "CI", DisplayTitle: "Title", Status: "completed", Conclusion: "failure"}
	jobs := []*models.GithubJob{
		{JobId: 1, Name: "first", Status: "completed", Conclusion: "failure"},
		{JobId: 2, Name: "second", Status: "completed", Conclusion: "success"},
		{JobId: 3, Name: "third", Status: "completed", Conclusion: "failure"},
	}
	annotations := map[int64][]*models.GithubAnnotation{
		3: {{Level: "warning", Message: "deprecated"}},
		1: {{Level: "failure", Message: "boom"}},
		2: {},
		// annotations of a job that isn't in the list aren't shown
		4: {{Level: "failure", Message: "orphan"}},
	}

	summary := FormatGithubRunSummary(i18n.EnglishTranslationSet(), GithubRunSummaryOpts{
		Run: run, Jobs: jobs, JobsLoaded: true, Annotations: annotations,
	})

	assert.Contains(t, summary, "\nAnnotations\nfirst\n  X boom\nthird\n  ! deprecated\n")
	assert.NotContains(t, summary, "orphan")
}

func TestFormatGithubAnnotation(t *testing.T) {
	withoutColors(t)

	scenarios := []struct {
		name       string
		annotation models.GithubAnnotation
		expected   string
	}{
		{
			name:       "failure with location",
			annotation: models.GithubAnnotation{Level: "failure", Message: "boom", Path: "main.go", StartLine: 12},
			expected:   "  X boom\n    main.go#12\n",
		},
		{
			name:       "warning without a line",
			annotation: models.GithubAnnotation{Level: "warning", Message: "careful", Path: "main.go"},
			expected:   "  ! careful\n    main.go\n",
		},
		{
			name:       "notice",
			annotation: models.GithubAnnotation{Level: "notice", Message: "fyi"},
			expected:   "  i fyi\n",
		},
		{
			name:       "unknown level",
			annotation: models.GithubAnnotation{Level: "something", Message: "hmm"},
			expected:   "  i hmm\n",
		},
		{
			name:       "the workflow file as location isn't shown",
			annotation: models.GithubAnnotation{Level: "failure", Message: "Process completed with exit code 1.", Path: ".github", StartLine: 1},
			expected:   "  X Process completed with exit code 1.\n",
		},
		{
			name:       "title is prepended",
			annotation: models.GithubAnnotation{Level: "failure", Title: "TestFoo", Message: "expected 1"},
			expected:   "  X TestFoo: expected 1\n",
		},
		{
			name:       "title that the message already contains",
			annotation: models.GithubAnnotation{Level: "failure", Title: "TestFoo", Message: "TestFoo failed"},
			expected:   "  X TestFoo failed\n",
		},
		{
			name:       "multi-line message with trailing newlines",
			annotation: models.GithubAnnotation{Level: "failure", Message: "line 1\nline 2\n\n"},
			expected:   "  X line 1\n    line 2\n",
		},
		{
			name:       "empty message",
			annotation: models.GithubAnnotation{Level: "failure"},
			expected:   "  X \n",
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, FormatGithubAnnotation(&s.annotation))
		})
	}
}

func TestFormatGithubAnnotationColors(t *testing.T) {
	withBasicColors(t)

	assert.Equal(t,
		"  \x1b[33m!\x1b[0m careful\n    \x1b[90mmain.go#3\x1b[0m\n",
		FormatGithubAnnotation(&models.GithubAnnotation{Level: "warning", Message: "careful", Path: "main.go", StartLine: 3}),
	)
}

func TestFormatGithubLogLinesCommands(t *testing.T) {
	withoutColors(t)

	scenarios := []struct {
		line     string
		expected string
	}{
		{"plain", "plain\n"},
		{"", "\n"},
		{"##[group]Run make", "▾ Run make\n"},
		{"##[endgroup]", ""},
		{"##[error]boom", "Error: boom\n"},
		{"##[warning]careful", "Warning: careful\n"},
		{"##[notice]fyi", "Notice: fyi\n"},
		{"##[debug]details", "details\n"},
		{"[command]/usr/bin/git fetch", "/usr/bin/git fetch\n"},
		{"##[unknown]text", "##[unknown]text\n"},
		{"##[error without closing bracket", "##[error without closing bracket\n"},
		{"  ##[error]indented", "  ##[error]indented\n"},
		{"##[error]", "Error: \n"},
		{"##[error]✗ ünïcödé", "Error: ✗ ünïcödé\n"},
	}

	for _, s := range scenarios {
		t.Run(s.line, func(t *testing.T) {
			assert.Equal(t, s.expected, FormatGithubLogLines([]string{s.line}))
		})
	}

	assert.Equal(t, "", FormatGithubLogLines(nil))
}

func TestFormatGithubLogLinesColors(t *testing.T) {
	withBasicColors(t)

	assert.Equal(t,
		"\x1b[1m▾ Run make\x1b[0m\n"+
			"\x1b[36mmake\x1b[0m\n"+
			"\x1b[31mError: boom\x1b[0m\n"+
			"\x1b[33mWarning: careful\x1b[0m\n"+
			"\x1b[34mNotice: fyi\x1b[0m\n"+
			"\x1b[90mdetails\x1b[0m\n",
		FormatGithubLogLines([]string{
			"##[group]Run make",
			"[command]make",
			"##[error]boom",
			"##[warning]careful",
			"##[notice]fyi",
			"##[debug]details",
		}),
	)
}

func newSuccessfulJob(start time.Time) *models.GithubJob {
	return &models.GithubJob{
		Name: "build", Status: "completed", Conclusion: "success",
		StartedAt: start, CompletedAt: start.Add(time.Minute),
		Url:        "https://github.com/o/r/actions/runs/1/job/2",
		RunnerName: "GitHub Actions 7",
		Labels:     []string{"ubuntu-latest", "x64"},
		Steps: []models.GithubStep{
			{Number: 1, Name: "Set up job", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start.Add(time.Second)},
			{Number: 2, Name: "Upload", Status: "completed", Conclusion: "skipped"},
			{Number: 3, Name: "Build", Status: "completed", Conclusion: "success", StartedAt: start.Add(time.Second), CompletedAt: start.Add(time.Minute)},
		},
	}
}

func TestFormatGithubJobDetailsOfSuccessfulJob(t *testing.T) {
	withoutColors(t)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	log := &github_commands.JobLog{
		Steps: []github_commands.StepLog{
			{StepNumber: 1, Lines: []string{"setting up"}},
			{StepNumber: 3, Lines: []string{"##[group]Run make", "##[endgroup]", "done"}},
		},
	}
	annotations := []*models.GithubAnnotation{{Level: "warning", Message: "Node 16 is deprecated"}}

	details := FormatGithubJobDetails(i18n.EnglishTranslationSet(), GithubJobDetailsOpts{
		Job: newSuccessfulJob(start), Log: log, LogLoaded: true, Annotations: annotations, Now: start.Add(time.Hour),
	})

	assert.Equal(t, `✓ build 1m0s
Ran on GitHub Actions 7 (ubuntu-latest, x64)
https://github.com/o/r/actions/runs/1/job/2

Steps
✓ Set up job 1s
- Upload
✓ Build 59s

Annotations
  ! Node 16 is deprecated

Log
━━ Set up job
setting up
━━ Build
▾ Run make
done
`, details)
}

func TestFormatGithubJobDetailsWithLogOfUnknownSteps(t *testing.T) {
	withoutColors(t)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	log := &github_commands.JobLog{Lines: []string{"first", "##[error]second"}}

	details := FormatGithubJobDetails(i18n.EnglishTranslationSet(), GithubJobDetailsOpts{
		Job: newSuccessfulJob(start), Log: log, LogLoaded: true, Now: start,
	})

	assert.Contains(t, details, "\nLog\nfirst\nError: second\n")
}

func TestFormatGithubJobDetailsWhileLoadingOrFailing(t *testing.T) {
	withoutColors(t)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tr := i18n.EnglishTranslationSet()

	details := FormatGithubJobDetails(tr, GithubJobDetailsOpts{Job: newSuccessfulJob(start), Now: start})
	assert.Contains(t, details, "\nLog\nLoading log…\n")

	details = FormatGithubJobDetails(tr, GithubJobDetailsOpts{Job: newSuccessfulJob(start), LogError: errors.New("HTTP 410"), Now: start})
	assert.Contains(t, details, "\nLog\nHTTP 410\n")

	// A queued job has no log yet, even if loading one failed.
	queued := &models.GithubJob{Name: "build", Status: "queued"}
	details = FormatGithubJobDetails(tr, GithubJobDetailsOpts{Job: queued, LogError: errors.New("HTTP 404"), Now: start})
	assert.Contains(t, details, "\nLog\nThe log will be available once the job has completed\n")
	assert.NotContains(t, details, "HTTP 404")
}

func TestFormatGithubJobDetailsOfCancelledJob(t *testing.T) {
	withoutColors(t)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	job := &models.GithubJob{
		Name: "test", Status: "completed", Conclusion: "cancelled",
		StartedAt: start, CompletedAt: start.Add(time.Minute),
		Steps: []models.GithubStep{
			{Number: 1, Name: "Set up job", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start},
			{Number: 2, Name: "Run tests", Status: "completed", Conclusion: "cancelled", StartedAt: start, CompletedAt: start.Add(time.Minute)},
		},
	}
	log := &github_commands.JobLog{
		Steps: []github_commands.StepLog{
			{StepNumber: 1, Lines: []string{"setting up"}},
			{StepNumber: 2, Lines: []string{"##[error]The operation was canceled."}},
		},
	}

	details := FormatGithubJobDetails(i18n.EnglishTranslationSet(), GithubJobDetailsOpts{Job: job, Log: log, LogLoaded: true, Now: start})

	assert.Contains(t, details, "\nLog of failed steps\n━━ Run tests\nError: The operation was canceled.\n")
	assert.NotContains(t, details, "setting up")
}

func TestFormatGithubStepLog(t *testing.T) {
	withoutColors(t)

	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	job := &models.GithubJob{Name: "build"}
	log := &github_commands.JobLog{
		Steps: []github_commands.StepLog{
			{StepNumber: 1, Lines: []string{"setting up"}},
			{StepNumber: 2},
			{StepNumber: 3, Lines: []string{"##[group]Run make", "##[endgroup]", "##[warning]careful"}},
		},
	}
	tr := i18n.EnglishTranslationSet()

	step := &models.GithubStep{Number: 3, Name: "Build", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start.Add(2 * time.Second)}
	assert.Equal(t, "✓ Build 2s\nbuild\n\n▾ Run make\nWarning: careful\n", FormatGithubStepLog(tr, job, step, log, start))

	silent := &models.GithubStep{Number: 2, Name: "Silent", Status: "completed", Conclusion: "success", StartedAt: start, CompletedAt: start}
	assert.Equal(t, "✓ Silent 0s\nbuild\n\nThis step logged nothing\n", FormatGithubStepLog(tr, job, silent, log, start))

	skipped := &models.GithubStep{Number: 4, Name: "Skipped", Status: "completed", Conclusion: "skipped"}
	assert.Contains(t, FormatGithubStepLog(tr, job, skipped, log, start), "\nbuild\n\nThis step logged nothing\n")
}

func TestFormatGithubJobDetailsAndStepLogHeadersWithoutDuration(t *testing.T) {
	withoutColors(t)

	tr := i18n.EnglishTranslationSet()
	job := &models.GithubJob{Name: "build", Status: "queued"}
	step := &models.GithubStep{Number: 1, Name: "Skipped", Status: "completed", Conclusion: "skipped"}

	details := FormatGithubJobDetails(tr, GithubJobDetailsOpts{Job: job, Now: time.Now()})
	stepLog := FormatGithubStepLog(tr, job, step, &github_commands.JobLog{}, time.Now())

	assert.True(t, strings.HasPrefix(details, "• build\n"))
	assert.True(t, strings.HasPrefix(stepLog, "- Skipped\n"))
}
