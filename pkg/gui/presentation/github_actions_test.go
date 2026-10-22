package presentation

import (
	"errors"
	"testing"
	"time"

	"github.com/gookit/color"
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

	start := time.Now().Add(-10 * time.Minute)
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
Triggered via push by octocat 10m ago · 3m5s · attempt 2
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
