package presentation

import (
	"fmt"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// for text that is there for reference rather than to be read
var githubDim = style.FromBasicFg(color.FgDarkGray)

// GithubActionsStateIcon returns the coloured symbol gh uses for a run, job,
// or step in this state.
func GithubActionsStateIcon(state models.GithubActionsState) string {
	switch state {
	case models.GithubActionsStateSuccess:
		return style.FgGreen.Sprint("✓")
	case models.GithubActionsStateFailure:
		return style.FgRed.Sprint("X")
	case models.GithubActionsStateCancelled:
		return style.FgYellow.Sprint("X")
	case models.GithubActionsStateSkipped, models.GithubActionsStateNeutral:
		return githubDim.Sprint("-")
	case models.GithubActionsStateQueued:
		return style.FgYellow.Sprint("•")
	case models.GithubActionsStateInProgress:
		return style.FgYellow.Sprint("*")
	}
	return "?"
}

// GithubActionsStatusLabel spells out a run, job, or step's state, in the
// colour of its icon.
func GithubActionsStatusLabel(tr *i18n.TranslationSet, state models.GithubActionsState) string {
	switch state {
	case models.GithubActionsStateSuccess:
		return style.FgGreen.Sprint(tr.GithubActions.StatusSuccess)
	case models.GithubActionsStateFailure:
		return style.FgRed.Sprint(tr.GithubActions.StatusFailure)
	case models.GithubActionsStateCancelled:
		return style.FgYellow.Sprint(tr.GithubActions.StatusCancelled)
	case models.GithubActionsStateSkipped:
		return githubDim.Sprint(tr.GithubActions.StatusSkipped)
	case models.GithubActionsStateNeutral:
		return githubDim.Sprint(tr.GithubActions.StatusNeutral)
	case models.GithubActionsStateQueued:
		return style.FgYellow.Sprint(tr.GithubActions.StatusQueued)
	case models.GithubActionsStateInProgress:
		return style.FgYellow.Sprint(tr.GithubActions.StatusInProgress)
	}
	return "?"
}

// FormatGithubDuration formats a duration the way gh does, e.g. "4m12s".
func FormatGithubDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

func GetGithubRunListDisplayStrings(tr *i18n.TranslationSet, runs []*models.GithubRun, fullDescription bool, now time.Time) [][]string {
	return lo.Map(runs, func(run *models.GithubRun, _ int) []string {
		age := style.FgBlue.Sprint(utils.TimeAgo(run.CreatedAt, now))
		state := run.State()
		// The title comes last because it's the one column that can be long.
		if !fullDescription {
			return []string{
				GithubActionsStateIcon(state),
				GithubActionsStatusLabel(tr, state),
				age,
				style.FgCyan.Sprint(utils.TruncateWithEllipsis(run.Name, 20)),
				run.Title(),
			}
		}

		return []string{
			GithubActionsStateIcon(state),
			GithubActionsStatusLabel(tr, state),
			age,
			style.FgCyan.Sprint(run.Name),
			style.FgMagenta.Sprint(run.HeadBranch),
			githubDim.Sprint(run.Event),
			FormatGithubDuration(run.Duration(now)),
			run.Title(),
		}
	})
}

func GetGithubJobListDisplayStrings(tr *i18n.TranslationSet, jobs []*models.GithubJob, now time.Time) [][]string {
	return lo.Map(jobs, func(job *models.GithubJob, _ int) []string {
		state := job.State()
		return []string{
			GithubActionsStateIcon(state),
			GithubActionsStatusLabel(tr, state),
			job.Name,
			githubDurationColumn(state, job.Duration(now)),
		}
	})
}

func GetGithubStepListDisplayStrings(tr *i18n.TranslationSet, steps []*models.GithubStep, now time.Time) [][]string {
	return lo.Map(steps, func(step *models.GithubStep, _ int) []string {
		state := step.State()
		return []string{
			GithubActionsStateIcon(state),
			GithubActionsStatusLabel(tr, state),
			step.Name,
			githubDurationColumn(state, step.Duration(now)),
		}
	})
}

func githubDurationColumn(state models.GithubActionsState, d time.Duration) string {
	if state == models.GithubActionsStateQueued || state == models.GithubActionsStateSkipped {
		return ""
	}
	return githubDim.Sprint(FormatGithubDuration(d))
}

// GithubRunSummaryOpts is what FormatGithubRunSummary shows: a run, and,
// once they have been loaded, its jobs and the annotations of the jobs that
// reported problems.
type GithubRunSummaryOpts struct {
	Run         *models.GithubRun
	Jobs        []*models.GithubJob
	JobsLoaded  bool
	JobsError   error
	Annotations map[int64][]*models.GithubAnnotation
	Now         time.Time
}

// FormatGithubRunSummary describes a run in roughly the way `gh run view`
// does.
func FormatGithubRunSummary(tr *i18n.TranslationSet, opts GithubRunSummaryOpts) string {
	run := opts.Run
	var b strings.Builder

	fmt.Fprintf(&b, "%s %s\n", GithubActionsStateIcon(run.State()), style.AttrBold.Sprint(run.Title()))
	b.WriteString(formatGithubRunMetadata(tr, run, opts.Now))

	b.WriteString("\n" + style.AttrBold.Sprint(tr.GithubActions.JobsSectionTitle) + "\n")
	switch {
	case opts.JobsError != nil:
		b.WriteString(style.FgRed.Sprint(opts.JobsError.Error()) + "\n")
	case !opts.JobsLoaded:
		b.WriteString(githubDim.Sprint(tr.GithubActions.LoadingJobs) + "\n")
	case len(opts.Jobs) == 0:
		b.WriteString(githubDim.Sprint(tr.GithubActions.NoJobs) + "\n")
	default:
		for _, job := range opts.Jobs {
			b.WriteString(formatGithubItemLine(job.State(), job.Name, job.Duration(opts.Now)))
			// Like gh, only spell out the steps of jobs that need attention.
			if job.State().IsFailed() || job.State().IsActive() {
				for i := range job.Steps {
					step := &job.Steps[i]
					if step.State() == models.GithubActionsStateSkipped {
						continue
					}
					fmt.Fprintf(&b, "  %s %s\n", GithubActionsStateIcon(step.State()), step.Name)
				}
			}
		}
	}

	if annotations := formatGithubAnnotations(tr, opts.Jobs, opts.Annotations); annotations != "" {
		b.WriteString("\n" + annotations)
	}

	return b.String()
}

func formatGithubRunMetadata(tr *i18n.TranslationSet, run *models.GithubRun, now time.Time) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%s %s · %s %s · %s\n",
		style.FgCyan.Sprint(run.Name),
		githubDim.Sprintf("#%d", run.Number),
		style.FgMagenta.Sprint(run.HeadBranch),
		style.FgYellow.Sprint(utils.ShortHash(run.HeadSha)),
		githubDim.Sprint(run.Path),
	)

	triggered := utils.ResolvePlaceholderString(tr.GithubActions.RunTriggered, map[string]string{
		"event": run.Event,
		"actor": run.Actor.Login,
		"ago":   utils.TimeAgo(run.CreatedAt, now),
	})
	fmt.Fprintf(&b, "%s · %s", triggered, FormatGithubDuration(run.Duration(now)))
	if run.Attempt > 1 {
		b.WriteString(" · " + utils.ResolvePlaceholderString(tr.GithubActions.RunAttempt, map[string]string{
			"attempt": fmt.Sprint(run.Attempt),
		}))
	}
	b.WriteString("\n")
	b.WriteString(githubDim.Sprint(run.Url) + "\n")

	return b.String()
}

// formatGithubItemLine shows a job or a step on a line of its own: its
// state, its name, and how long it took, where that is known.
func formatGithubItemLine(state models.GithubActionsState, name string, d time.Duration) string {
	line := GithubActionsStateIcon(state) + " " + name
	if duration := githubDurationColumn(state, d); duration != "" {
		line += " " + duration
	}
	return line + "\n"
}

func formatGithubAnnotations(tr *i18n.TranslationSet, jobs []*models.GithubJob, annotationsByJob map[int64][]*models.GithubAnnotation) string {
	var b strings.Builder
	for _, job := range jobs {
		annotations := annotationsByJob[job.JobId]
		if len(annotations) == 0 {
			continue
		}
		if b.Len() == 0 {
			b.WriteString(style.AttrBold.Sprint(tr.GithubActions.AnnotationsSectionTitle) + "\n")
		}
		b.WriteString(job.Name + "\n")
		for _, annotation := range annotations {
			b.WriteString(FormatGithubAnnotation(annotation))
		}
	}
	return b.String()
}

// FormatGithubAnnotation formats an annotation as an indented, coloured
// message, followed by where it points to.
func FormatGithubAnnotation(annotation *models.GithubAnnotation) string {
	var icon string
	switch annotation.Level {
	case "failure":
		icon = style.FgRed.Sprint("X")
	case "warning":
		icon = style.FgYellow.Sprint("!")
	default:
		icon = style.FgBlue.Sprint("i")
	}

	message := annotation.Message
	if annotation.Title != "" && !strings.Contains(message, annotation.Title) {
		message = annotation.Title + ": " + message
	}
	lines := strings.Split(strings.TrimRight(message, "\n"), "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "  %s %s\n", icon, lines[0])
	for _, line := range lines[1:] {
		b.WriteString("    " + line + "\n")
	}
	if annotation.Path != "" && annotation.Path != ".github" {
		location := annotation.Path
		if annotation.StartLine > 0 {
			location += fmt.Sprintf("#%d", annotation.StartLine)
		}
		b.WriteString("    " + githubDim.Sprint(location) + "\n")
	}
	return b.String()
}

// FormatGithubLogLines renders log lines with the runner's workflow commands
// (e.g. "##[error]") turned into colours, the way GitHub's web UI shows them.
func FormatGithubLogLines(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		if line == "##[endgroup]" {
			continue
		}
		b.WriteString(formatGithubLogLine(line))
		b.WriteString("\n")
	}
	return b.String()
}

func formatGithubLogLine(line string) string {
	// how actions log the commands they run
	if command, found := strings.CutPrefix(line, "[command]"); found {
		return style.FgCyan.Sprint(command)
	}
	if !strings.HasPrefix(line, "##[") {
		return line
	}
	command, rest, found := strings.Cut(line[len("##["):], "]")
	if !found {
		return line
	}
	switch command {
	case "group":
		return style.AttrBold.Sprint("▾ " + rest)
	case "error":
		return style.FgRed.Sprint("Error: " + rest)
	case "warning":
		return style.FgYellow.Sprint("Warning: " + rest)
	case "notice":
		return style.FgBlue.Sprint("Notice: " + rest)
	case "debug":
		return githubDim.Sprint(rest)
	}
	return line
}

// GithubJobDetailsOpts is what FormatGithubJobDetails shows: a job, and once
// they have been loaded, its log and annotations.
type GithubJobDetailsOpts struct {
	Job         *models.GithubJob
	Log         *github_commands.JobLog
	LogLoaded   bool
	LogError    error
	Annotations []*models.GithubAnnotation
	Now         time.Time
}

// FormatGithubJobDetails describes a job: its steps, its annotations, and its
// log, or, for a job that failed, only the log of the steps that failed.
func FormatGithubJobDetails(tr *i18n.TranslationSet, opts GithubJobDetailsOpts) string {
	job := opts.Job
	var b strings.Builder

	b.WriteString(formatGithubItemLine(job.State(), style.AttrBold.Sprint(job.Name), job.Duration(opts.Now)))
	if job.RunnerName != "" {
		b.WriteString(githubDim.Sprint(utils.ResolvePlaceholderString(tr.GithubActions.RanOn, map[string]string{
			"runner": job.RunnerName,
			"labels": strings.Join(job.Labels, ", "),
		})) + "\n")
	}
	b.WriteString(githubDim.Sprint(job.Url) + "\n")

	b.WriteString("\n" + style.AttrBold.Sprint(tr.GithubActions.StepsSectionTitle) + "\n")
	for i := range job.Steps {
		step := &job.Steps[i]
		b.WriteString(formatGithubItemLine(step.State(), step.Name, step.Duration(opts.Now)))
	}

	if len(opts.Annotations) > 0 {
		b.WriteString("\n" + style.AttrBold.Sprint(tr.GithubActions.AnnotationsSectionTitle) + "\n")
		for _, annotation := range opts.Annotations {
			b.WriteString(FormatGithubAnnotation(annotation))
		}
	}

	onlyFailedSteps := job.State().IsFailed()
	title := tr.GithubActions.LogSectionTitle
	if onlyFailedSteps {
		title = tr.GithubActions.FailedStepsLogSectionTitle
	}
	b.WriteString("\n" + style.AttrBold.Sprint(title) + "\n")
	switch {
	case job.State().IsActive():
		b.WriteString(githubDim.Sprint(tr.GithubActions.LogNotAvailableYet) + "\n")
	case opts.LogError != nil:
		b.WriteString(style.FgRed.Sprint(opts.LogError.Error()) + "\n")
	case !opts.LogLoaded:
		b.WriteString(githubDim.Sprint(tr.GithubActions.LoadingLog) + "\n")
	default:
		b.WriteString(formatGithubJobLog(job, opts.Log, onlyFailedSteps))
	}

	return b.String()
}

func formatGithubJobLog(job *models.GithubJob, log *github_commands.JobLog, onlyFailedSteps bool) string {
	if len(log.Steps) == 0 {
		// We couldn't tell which line belongs to which step, e.g. because the
		// job has none with timestamps.
		return FormatGithubLogLines(log.Lines)
	}

	steps := lo.SliceToMap(job.Steps, func(step models.GithubStep) (int, models.GithubStep) {
		return step.Number, step
	})

	var b strings.Builder
	for _, stepLog := range log.Steps {
		step := steps[stepLog.StepNumber]
		if onlyFailedSteps && !step.State().IsFailed() {
			continue
		}
		b.WriteString(style.FgCyan.SetBold().Sprint("━━ "+step.Name) + "\n")
		b.WriteString(FormatGithubLogLines(stepLog.Lines))
	}
	return b.String()
}

// FormatGithubStepLog shows the part of a job's log that one of its steps
// produced.
func FormatGithubStepLog(tr *i18n.TranslationSet, job *models.GithubJob, step *models.GithubStep, log *github_commands.JobLog, now time.Time) string {
	var b strings.Builder
	b.WriteString(formatGithubItemLine(step.State(), style.AttrBold.Sprint(step.Name), step.Duration(now)))
	b.WriteString(githubDim.Sprint(job.Name) + "\n\n")

	stepLog, found := lo.Find(log.Steps, func(stepLog github_commands.StepLog) bool {
		return stepLog.StepNumber == step.Number
	})
	if !found || len(stepLog.Lines) == 0 {
		b.WriteString(githubDim.Sprint(tr.GithubActions.NoLogForStep) + "\n")
		return b.String()
	}
	b.WriteString(FormatGithubLogLines(stepLog.Lines))
	return b.String()
}
