package presentation

import (
	"fmt"
	"strings"
	"time"

	"github.com/gookit/color"
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

func GetGithubRunListDisplayStrings(runs []*models.GithubRun, fullDescription bool, now time.Time) [][]string {
	return lo.Map(runs, func(run *models.GithubRun, _ int) []string {
		age := style.FgBlue.Sprint(utils.UnixToTimeAgo(run.CreatedAt.Unix()))
		// The title comes last because it's the one column that can be long.
		if !fullDescription {
			return []string{
				GithubActionsStateIcon(run.State()),
				age,
				style.FgCyan.Sprint(utils.TruncateWithEllipsis(run.Name, 20)),
				run.Title(),
			}
		}

		return []string{
			GithubActionsStateIcon(run.State()),
			age,
			style.FgCyan.Sprint(run.Name),
			style.FgMagenta.Sprint(run.HeadBranch),
			githubDim.Sprint(run.Event),
			FormatGithubDuration(run.Duration(now)),
			run.Title(),
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
			b.WriteString(formatGithubJobLine(job, opts.Now))
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
		"ago":   utils.UnixToTimeAgo(run.CreatedAt.Unix()),
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

func formatGithubJobLine(job *models.GithubJob, now time.Time) string {
	line := GithubActionsStateIcon(job.State()) + " " + job.Name
	if duration := githubDurationColumn(job.State(), job.Duration(now)); duration != "" {
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
