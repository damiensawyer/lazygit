package helpers

import (
	goContext "context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

const (
	// How often the runs are reloaded while the GitHub Actions view is shown.
	// Something still running is worth watching closely; otherwise we only
	// look out for new runs.
	githubActiveRefreshInterval = 5 * time.Second
	githubIdleRefreshInterval   = 30 * time.Second

	githubRunsLimit = 50
)

// GithubRunDetails is what we load about a run in addition to what the runs
// list says about it, for showing it in the main view.
type GithubRunDetails struct {
	Jobs        []*models.GithubJob
	Annotations map[int64][]*models.GithubAnnotation
	Err         error

	// the run's UpdatedAt when the details were loaded
	updatedAt time.Time
	// set for a run that is still going, whose jobs may have progressed
	// without the run's UpdatedAt changing
	stale bool
}

// GithubActionsHelper loads the current repository's GitHub Actions data and
// keeps it fresh while it is being shown.
type GithubActionsHelper struct {
	c            *HelperCommon
	windowHelper *WindowHelper
	clientHelper *GithubClientHelper

	// bumped whenever the runs are loaded, so that an auto-refresh scheduled
	// before then knows it has been superseded
	refreshGeneration int

	// set when a run has been requested that GitHub hasn't created yet, so
	// that the next refresh comes soon
	expectingNewRun bool

	// keyed by run ID; only touched on the UI thread
	runDetails        map[int64]*GithubRunDetails
	runDetailsLoading map[int64]bool

	// keyed by job ID; only touched on the UI thread
	jobLogs        map[int64]*GithubJobLog
	jobLogsLoading map[int64]bool
}

// GithubJobLog is a completed job's log, along with its annotations.
type GithubJobLog struct {
	Log         *github_commands.JobLog
	Annotations []*models.GithubAnnotation
	Err         error
}

func NewGithubActionsHelper(
	c *HelperCommon,
	windowHelper *WindowHelper,
	clientHelper *GithubClientHelper,
) *GithubActionsHelper {
	return &GithubActionsHelper{
		c:                 c,
		windowHelper:      windowHelper,
		clientHelper:      clientHelper,
		runDetails:        map[int64]*GithubRunDetails{},
		runDetailsLoading: map[int64]bool{},
		jobLogs:           map[int64]*GithubJobLog{},
		jobLogsLoading:    map[int64]bool{},
	}
}

// Open shows the runs of the current repository, logging in first if needed.
func (self *GithubActionsHelper) Open() error {
	return self.withClient(func() error {
		// The runs take over the commits window, so escaping from them shows the
		// commits again.
		runsContext := self.c.Contexts().GithubRuns
		runsContext.SetParentContext(self.c.Contexts().LocalCommits)
		runsContext.GetView().TitlePrefix = self.c.Contexts().LocalCommits.GetView().TitlePrefix

		return self.c.WithWaitingStatus(self.c.Tr.GithubActions.LoadingRuns, func(gocui.Task) error {
			return self.loadRuns(false, func() {
				runsContext.SetSelection(0)
				self.c.Context().Push(runsContext, types.OnFocusOpts{})
			})
		})
	})
}

// RefreshRuns reloads the runs, keeping the selected one selected, along
// with the jobs being shown.
func (self *GithubActionsHelper) RefreshRuns() error {
	return self.withClient(func() error {
		return self.c.WithWaitingStatus(self.c.Tr.GithubActions.LoadingRuns, func(gocui.Task) error {
			return self.loadRuns(false, self.reloadChangedDetails)
		})
	})
}

// withClient makes sure we know which repository to talk to and have a token
// for it, and then calls f on the UI thread.
func (self *GithubActionsHelper) withClient(f func() error) error {
	return self.clientHelper.WithClient(func(*github_commands.Client) error {
		self.updateTitle()
		return f()
	})
}

// HandleError reports a failed API request, logging the user in again if
// the token has stopped working. Must be called on the UI thread.
func (self *GithubActionsHelper) HandleError(err error) error {
	return self.handleError(err)
}

// handleError reports a failed API request. Must be called on the UI thread.
func (self *GithubActionsHelper) handleError(err error) error {
	return self.clientHelper.HandleError(err, self.Open)
}

func (self *GithubActionsHelper) updateTitle() {
	title := utils.ResolvePlaceholderString(self.c.Tr.GithubActions.RunsTitleForRepo, map[string]string{
		"repo": self.clientHelper.Client().Repo().FullName(),
	})
	if description := self.describeRunsFilter(); description != "" {
		title += " · " + description
	}
	self.c.Views().GithubRuns.Title = title
}

func (self *GithubActionsHelper) describeRunsFilter() string {
	tr := self.c.Tr.GithubActions
	filter := self.c.Contexts().GithubRuns.GetRunsFilter()

	parts := []string{}
	add := func(template string, value string) {
		if value != "" {
			parts = append(parts, utils.ResolvePlaceholderString(template, map[string]string{"value": value}))
		}
	}
	if filter.Workflow != nil {
		add(tr.FilterDescriptionWorkflow, filter.Workflow.Name)
	}
	add(tr.FilterDescriptionBranch, filter.Branch)
	if status, found := lo.Find(self.RunStatusFilters(), func(status GithubRunStatusFilter) bool {
		return status.Value == filter.Status
	}); found && filter.Status != "" {
		add(tr.FilterDescriptionStatus, status.Label)
	}
	add(tr.FilterDescriptionEvent, filter.Event)
	add(tr.FilterDescriptionCommit, utils.ShortHash(filter.HeadSha))
	return strings.Join(parts, ", ")
}

// GithubRunStatusFilter is a value of the runs API's status parameter, which
// takes conclusions as well as statuses.
type GithubRunStatusFilter struct {
	Value string
	Label string
}

func (self *GithubActionsHelper) RunStatusFilters() []GithubRunStatusFilter {
	tr := self.c.Tr.GithubActions
	return []GithubRunStatusFilter{
		{models.GithubConclusionFailure, tr.StatusFailure},
		{"in_progress", tr.StatusInProgress},
		{models.GithubStatusQueued, tr.StatusQueued},
		{models.GithubConclusionActionRequired, tr.StatusActionRequired},
		{models.GithubConclusionSuccess, tr.StatusSuccess},
		{models.GithubConclusionCancelled, tr.StatusCancelled},
	}
}

// SetRunsFilter changes which runs are shown, and reloads them.
func (self *GithubActionsHelper) SetRunsFilter(filter context.GithubRunsFilter) error {
	runsContext := self.c.Contexts().GithubRuns
	runsContext.SetRunsFilter(filter)
	self.updateTitle()
	return self.c.WithWaitingStatus(self.c.Tr.GithubActions.LoadingRuns, func(gocui.Task) error {
		return self.loadRuns(false, func() {
			runsContext.SetSelection(0)
			self.c.PostRefreshUpdate(runsContext)
		})
	})
}

// WithWorkflows loads the repository's workflows and passes them to f on the
// UI thread.
func (self *GithubActionsHelper) WithWorkflows(f func(workflows []*models.GithubWorkflow) error) error {
	return self.withClient(func() error {
		client := self.clientHelper.Client()
		return self.c.WithWaitingStatus(self.c.Tr.GithubActions.LoadingWorkflows, func(gocui.Task) error {
			workflows, err := client.ListWorkflows(goContext.Background())
			self.c.OnUIThread(func() error {
				if err != nil {
					return self.handleError(err)
				}
				return f(workflows)
			})
			return nil
		})
	})
}

// loadRuns runs on a worker. It fetches the runs and, on the UI thread, shows
// them and then calls then. A background load is one the user didn't ask
// for, so it doesn't report errors; it just tries again later.
func (self *GithubActionsHelper) loadRuns(background bool, then func()) error {
	client := self.clientHelper.Client()
	generation := self.c.State().GetRepoGeneration()

	filter := self.c.Contexts().GithubRuns.GetRunsFilter()
	opts := github_commands.ListRunsOptions{
		Branch:  filter.Branch,
		Status:  filter.Status,
		Event:   filter.Event,
		HeadSha: filter.HeadSha,
		Limit:   githubRunsLimit,
	}
	if filter.Workflow != nil {
		opts.WorkflowId = filter.Workflow.WorkflowId
	}
	runs, err := client.ListRuns(goContext.Background(), opts)

	onUIThread := self.c.OnUIThread
	if background {
		onUIThread = self.c.OnUIThreadBackground
	}
	onUIThread(func() error {
		if self.c.State().GetRepoGeneration() != generation {
			return nil
		}
		if err != nil {
			if background {
				self.c.Log.Warnf("Failed to refresh GitHub Actions runs: %v", err)
				self.scheduleAutoRefresh()
				return nil
			}
			return self.handleError(err)
		}
		self.setRuns(runs)
		then()
		self.scheduleAutoRefresh()
		return nil
	})
	return nil
}

func (self *GithubActionsHelper) setRuns(runs []*models.GithubRun) {
	runsContext := self.c.Contexts().GithubRuns
	selectedId := runsContext.GetSelectedItemId()

	self.c.Model().GithubRuns = runs
	runsContext.ReApplyFilter(self.c.UserConfig().Gui.UseFuzzySearch())

	// New runs show up at the top, so the selected one may have moved down.
	if selectedId != "" {
		if _, idx, found := lo.FindIndexOf(runsContext.GetItems(), func(run *models.GithubRun) bool {
			return run.ID() == selectedId
		}); found {
			runsContext.SetSelection(idx)
		}
	}

	self.c.PostRefreshUpdateWithOptions(runsContext,
		types.OnFocusOpts{KeepScrollPosition: true})
}

// IsShowing reports whether any of the GitHub Actions views is visible.
func (self *GithubActionsHelper) IsShowing() bool {
	return lo.SomeBy(self.contexts(), self.isShowing)
}

func (self *GithubActionsHelper) isShowing(context types.Context) bool {
	return self.windowHelper.GetViewNameForWindow(context.GetWindowName()) == context.GetViewName()
}

func (self *GithubActionsHelper) contexts() []types.Context {
	return []types.Context{self.c.Contexts().GithubRuns, self.c.Contexts().GithubJobs, self.c.Contexts().GithubSteps}
}

func (self *GithubActionsHelper) scheduleAutoRefresh() {
	self.refreshGeneration++
	refreshGeneration := self.refreshGeneration
	repoGeneration := self.c.State().GetRepoGeneration()

	interval := githubIdleRefreshInterval
	if self.expectingNewRun || lo.SomeBy(self.c.Model().GithubRuns, func(run *models.GithubRun) bool { return run.State().IsActive() }) {
		interval = githubActiveRefreshInterval
	}
	self.expectingNewRun = false

	time.AfterFunc(interval, utils.SafeFunc(func() {
		self.c.OnUIThreadBackground(func() error {
			if refreshGeneration != self.refreshGeneration ||
				repoGeneration != self.c.State().GetRepoGeneration() ||
				self.clientHelper.Client() == nil {
				return nil
			}
			if !self.IsShowing() {
				// Look again later; the view may be shown again without being
				// reopened, e.g. by switching back to its window.
				self.scheduleAutoRefresh()
				return nil
			}
			self.c.OnWorkerBackground(func(gocui.Task) error {
				return self.loadRuns(true, self.reloadChangedDetails)
			})
			return nil
		})
	}))
}

// reloadChangedDetails brings what is shown about the runs' jobs up to date
// with freshly loaded runs.
func (self *GithubActionsHelper) reloadChangedDetails() {
	self.markActiveRunDetailsStale()
	self.reloadJobsIfRunChanged()
}

// ExpectNewRun makes the runs refresh soon, for after triggering a workflow:
// GitHub takes a moment to create the run, so it isn't there to be shown
// straight away.
func (self *GithubActionsHelper) ExpectNewRun() {
	self.expectingNewRun = true
}

// RunAction performs an action on the repository's runs, like cancelling one,
// and then reloads the runs to show its effect.
func (self *GithubActionsHelper) RunAction(waitingStatus string, doneToast string, action func(ctx goContext.Context, client *github_commands.Client) error) error {
	return self.withClient(func() error {
		client := self.clientHelper.Client()
		return self.c.WithWaitingStatus(waitingStatus, func(gocui.Task) error {
			if err := action(goContext.Background(), client); err != nil {
				self.c.OnUIThread(func() error { return self.handleError(err) })
				return nil
			}
			self.c.OnUIThread(func() error {
				self.c.Toast(doneToast)
				return nil
			})
			return self.loadRuns(false, self.reloadChangedDetails)
		})
	})
}

// markActiveRunDetailsStale makes the details of runs that are still going
// reload the next time they are shown, so that the main view shows their
// progress.
func (self *GithubActionsHelper) markActiveRunDetailsStale() {
	for _, run := range self.c.Model().GithubRuns {
		if details, ok := self.runDetails[run.RunId]; ok && run.State().IsActive() {
			details.stale = true
		}
	}
	if self.c.Context().IsCurrent(self.c.Contexts().GithubRuns) {
		self.c.Contexts().GithubRuns.HandleRenderToMain()
	}
}

// reloadJobsIfRunChanged reloads the jobs being shown if their run has
// progressed since they were loaded.
func (self *GithubActionsHelper) reloadJobsIfRunChanged() {
	jobsContext := self.c.Contexts().GithubJobs
	shownRun := jobsContext.GetRun()
	if shownRun == nil || !(self.isShowing(jobsContext) || self.isShowing(self.c.Contexts().GithubSteps)) {
		return
	}
	run, found := lo.Find(self.c.Model().GithubRuns, func(run *models.GithubRun) bool {
		return run.RunId == shownRun.RunId
	})
	if !found || (!run.State().IsActive() && run.UpdatedAt.Equal(shownRun.UpdatedAt)) {
		return
	}

	self.loadRunDetailsInBackground(run, func(details *GithubRunDetails) {
		if details.Err != nil || jobsContext.GetRun() == nil || jobsContext.GetRun().RunId != run.RunId {
			return
		}
		self.setJobs(run, details.Jobs)
	})
}

// OpenJobs shows the jobs of a run in place of the runs.
func (self *GithubActionsHelper) OpenJobs(run *models.GithubRun) error {
	show := func(details *GithubRunDetails) error {
		if details.Err != nil {
			return self.handleError(details.Err)
		}
		jobsContext := self.c.Contexts().GithubJobs
		jobsContext.SetParentContext(self.c.Contexts().GithubRuns)
		jobsContext.GetView().TitlePrefix = self.c.Contexts().GithubRuns.GetView().TitlePrefix
		jobsContext.SetSelection(0)
		jobsContext.ClearFilter()
		self.setJobs(run, details.Jobs)
		self.c.Context().Push(jobsContext, types.OnFocusOpts{})
		return nil
	}

	if details, ok := self.runDetails[run.RunId]; ok && details.updatedAt.Equal(run.UpdatedAt) {
		return show(details)
	}

	client := self.clientHelper.Client()
	generation := self.c.State().GetRepoGeneration()
	return self.c.WithWaitingStatus(self.c.Tr.GithubActions.LoadingJobs, func(gocui.Task) error {
		details := loadRunDetails(client, run)
		self.c.OnUIThread(func() error {
			if self.c.State().GetRepoGeneration() != generation {
				return nil
			}
			self.runDetails[run.RunId] = details
			return show(details)
		})
		return nil
	})
}

func (self *GithubActionsHelper) setJobs(run *models.GithubRun, jobs []*models.GithubJob) {
	jobsContext := self.c.Contexts().GithubJobs
	selectedId := jobsContext.GetSelectedItemId()

	jobsContext.SetJobs(run, jobs)
	jobsContext.ReApplyFilter(self.c.UserConfig().Gui.UseFuzzySearch())
	if _, idx, found := lo.FindIndexOf(jobsContext.GetItems(), func(job *models.GithubJob) bool {
		return job.ID() == selectedId
	}); found {
		jobsContext.SetSelection(idx)
	}

	self.c.PostRefreshUpdateWithOptions(jobsContext,
		types.OnFocusOpts{KeepScrollPosition: true})

	stepsContext := self.c.Contexts().GithubSteps
	if shownJob := stepsContext.GetJob(); shownJob != nil {
		if job, found := lo.Find(jobs, func(job *models.GithubJob) bool { return job.JobId == shownJob.JobId }); found {
			stepsContext.SetJob(job)
			self.c.PostRefreshUpdateWithOptions(stepsContext,
				types.OnFocusOpts{KeepScrollPosition: true})
		}
	}
}

// OpenSteps shows the steps of a job in place of the jobs.
func (self *GithubActionsHelper) OpenSteps(job *models.GithubJob) error {
	stepsContext := self.c.Contexts().GithubSteps
	stepsContext.SetParentContext(self.c.Contexts().GithubJobs)
	stepsContext.GetView().TitlePrefix = self.c.Contexts().GithubJobs.GetView().TitlePrefix
	stepsContext.ClearFilter()
	stepsContext.SetJob(job)

	// Most of the time a step is looked at because it failed.
	_, idx, found := lo.FindIndexOf(stepsContext.GetItems(), func(step *models.GithubStep) bool {
		return step.State().IsFailed()
	})
	if !found {
		idx = 0
	}
	stepsContext.SetSelection(idx)

	self.c.PostRefreshUpdate(stepsContext)
	self.c.Context().Push(stepsContext, types.OnFocusOpts{})
	return nil
}

// GetRunDetails returns the details of a run if they have been loaded, or nil
// if they haven't. Unless they are up to date, it starts loading them, and
// re-renders the main view once they have been, if the run is still selected
// by then. Outdated details are returned in the meantime, so that the main
// view doesn't flash a loading message on every refresh of a run in progress.
// Must be called on the UI thread.
func (self *GithubActionsHelper) GetRunDetails(run *models.GithubRun) *GithubRunDetails {
	details, loaded := self.runDetails[run.RunId]
	upToDate := loaded && !details.stale && details.updatedAt.Equal(run.UpdatedAt)
	if upToDate || self.runDetailsLoading[run.RunId] || self.clientHelper.Client() == nil {
		return details
	}
	self.loadRunDetailsInBackground(run, func(*GithubRunDetails) {
		runsContext := self.c.Contexts().GithubRuns
		if self.c.Context().IsCurrent(runsContext) && runsContext.GetSelectedItemId() == run.ID() {
			runsContext.HandleRenderToMain()
		}
	})
	return details
}

// loadRunDetailsInBackground loads a run's details, stores them, and passes
// them to then on the UI thread.
func (self *GithubActionsHelper) loadRunDetailsInBackground(run *models.GithubRun, then func(*GithubRunDetails)) {
	self.runDetailsLoading[run.RunId] = true

	client := self.clientHelper.Client()
	generation := self.c.State().GetRepoGeneration()
	self.c.OnWorkerBackground(func(gocui.Task) error {
		details := loadRunDetails(client, run)
		self.c.OnUIThreadBackground(func() error {
			if self.c.State().GetRepoGeneration() != generation {
				return nil
			}
			delete(self.runDetailsLoading, run.RunId)
			self.runDetails[run.RunId] = details
			then(details)
			return nil
		})
		return nil
	})
}

func loadRunDetails(client *github_commands.Client, run *models.GithubRun) *GithubRunDetails {
	details := &GithubRunDetails{updatedAt: run.UpdatedAt, Annotations: map[int64][]*models.GithubAnnotation{}}

	ctx := goContext.Background()
	details.Jobs, details.Err = client.ListJobs(ctx, run.RunId)
	if details.Err != nil {
		return details
	}

	// Annotations explain failures (a compiler error, a failed assertion),
	// so they are worth a request per failed job.
	for _, job := range details.Jobs {
		if !job.State().IsFailed() {
			continue
		}
		annotations, err := client.ListAnnotations(ctx, job.JobId)
		if err == nil {
			details.Annotations[job.JobId] = annotations
		}
	}
	return details
}

// GetJobLog returns the log of a completed job if it has been loaded.
// Otherwise it starts loading it, and re-renders the main view once it has
// been, if the job is still selected by then. Must be called on the UI
// thread.
func (self *GithubActionsHelper) GetJobLog(job *models.GithubJob) *GithubJobLog {
	if jobLog, ok := self.jobLogs[job.JobId]; ok {
		return jobLog
	}
	if self.jobLogsLoading[job.JobId] || self.clientHelper.Client() == nil {
		return nil
	}
	self.jobLogsLoading[job.JobId] = true

	client := self.clientHelper.Client()
	generation := self.c.State().GetRepoGeneration()
	self.c.OnWorkerBackground(func(gocui.Task) error {
		jobLog := loadJobLog(client, job)
		self.c.OnUIThreadBackground(func() error {
			if self.c.State().GetRepoGeneration() != generation {
				return nil
			}
			delete(self.jobLogsLoading, job.JobId)
			self.jobLogs[job.JobId] = jobLog

			jobsContext := self.c.Contexts().GithubJobs
			stepsContext := self.c.Contexts().GithubSteps
			if self.c.Context().IsCurrent(jobsContext) && jobsContext.GetSelectedItemId() == job.ID() {
				jobsContext.HandleRenderToMain()
			}
			if self.c.Context().IsCurrent(stepsContext) && stepsContext.GetJob() != nil && stepsContext.GetJob().JobId == job.JobId {
				stepsContext.HandleRenderToMain()
			}
			return nil
		})
		return nil
	})
	return nil
}

// WithJobLog passes the log of a completed job to f on the UI thread, loading
// it first if it hasn't been yet.
func (self *GithubActionsHelper) WithJobLog(job *models.GithubJob, f func(*github_commands.JobLog) error) error {
	if job.State().IsActive() {
		return errors.New(self.c.Tr.GithubActions.LogNotAvailableYet)
	}
	if jobLog, ok := self.jobLogs[job.JobId]; ok && jobLog.Err == nil {
		return f(jobLog.Log)
	}

	client := self.clientHelper.Client()
	generation := self.c.State().GetRepoGeneration()
	return self.c.WithWaitingStatus(self.c.Tr.GithubActions.LoadingLog, func(gocui.Task) error {
		jobLog := loadJobLog(client, job)
		self.c.OnUIThread(func() error {
			if self.c.State().GetRepoGeneration() != generation {
				return nil
			}
			if jobLog.Err != nil {
				return self.handleError(jobLog.Err)
			}
			self.jobLogs[job.JobId] = jobLog
			return f(jobLog.Log)
		})
		return nil
	})
}

// WithJobLogFile writes a job's log to a file, for opening in the user's
// editor, which is better at searching a long log than the main view is. It
// passes f the file's path, and the line that the given step's part of the
// log starts at, or 1 if stepNumber is 0.
func (self *GithubActionsHelper) WithJobLogFile(job *models.GithubJob, stepNumber int, f func(path string, line int) error) error {
	return self.WithJobLog(job, func(log *github_commands.JobLog) error {
		dir := filepath.Join(self.c.OS().GetTempDir(), "github-actions")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		path := filepath.Join(dir, fmt.Sprintf("%s-%s-job-%d.log",
			self.clientHelper.Client().Repo().Owner, self.clientHelper.Client().Repo().Name, job.JobId))
		content := utils.Decolorise(strings.Join(log.Lines, "\n")) + "\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}

		line := 1
		for _, stepLog := range log.Steps {
			if stepLog.StepNumber == stepNumber {
				return f(path, line)
			}
			line += len(stepLog.Lines)
		}
		return f(path, 1)
	})
}

func loadJobLog(client *github_commands.Client, job *models.GithubJob) *GithubJobLog {
	ctx := goContext.Background()
	log, err := client.GetJobLog(ctx, job.JobId)
	if err != nil {
		return &GithubJobLog{Err: err}
	}

	jobLog := &GithubJobLog{Log: github_commands.ParseJobLog(log, job.Steps)}
	if job.State().IsFailed() {
		jobLog.Annotations, _ = client.ListAnnotations(ctx, job.JobId)
	}
	return jobLog
}
