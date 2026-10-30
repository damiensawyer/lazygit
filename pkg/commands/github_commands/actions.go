package github_commands

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/samber/lo"
)

// ListRunsOptions narrows down which runs ListRuns returns. Zero values mean
// "don't filter by this".
type ListRunsOptions struct {
	Branch     string
	WorkflowId int64
	// One of the statuses or conclusions the API accepts, e.g. "failure" or
	// "in_progress".
	Status  string
	Event   string
	HeadSha string
	Limit   int
}

// ListRuns returns the repository's most recent workflow runs, newest first.
func (self *Client) ListRuns(ctx context.Context, opts ListRunsOptions) ([]*models.GithubRun, error) {
	path := self.repoPath("actions/runs")
	if opts.WorkflowId != 0 {
		path = self.repoPath("actions/workflows/%d/runs", opts.WorkflowId)
	}

	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	query := url.Values{"per_page": {itoa(int64(limit))}}
	if opts.Branch != "" {
		query.Set("branch", opts.Branch)
	}
	if opts.Status != "" {
		query.Set("status", opts.Status)
	}
	if opts.Event != "" {
		query.Set("event", opts.Event)
	}
	if opts.HeadSha != "" {
		query.Set("head_sha", opts.HeadSha)
	}

	var result struct {
		WorkflowRuns []*models.GithubRun `json:"workflow_runs"`
	}
	if err := self.getJSON(ctx, path, query, &result); err != nil {
		return nil, err
	}
	for _, run := range result.WorkflowRuns {
		self.correctQueuedStatus(ctx, run)
	}
	return result.WorkflowRuns, nil
}

// correctQueuedStatus looks at a queued run's jobs to tell whether it has
// really started; see GithubRun.CorrectQueuedStatus. If the jobs can't be
// loaded, GitHub's status stands.
func (self *Client) correctQueuedStatus(ctx context.Context, run *models.GithubRun) {
	if run.State() != models.GithubActionsStateQueued {
		return
	}
	if jobs, err := self.ListJobs(ctx, run.RunId); err == nil {
		run.CorrectQueuedStatus(jobs)
	}
}

func (self *Client) GetRun(ctx context.Context, runId int64) (*models.GithubRun, error) {
	var run models.GithubRun
	if err := self.getJSON(ctx, self.repoPath("actions/runs/%d", runId), nil, &run); err != nil {
		return nil, err
	}
	self.correctQueuedStatus(ctx, &run)
	return &run, nil
}

// ListJobs returns the jobs of a run's latest attempt.
func (self *Client) ListJobs(ctx context.Context, runId int64) ([]*models.GithubJob, error) {
	var jobs []*models.GithubJob
	// Matrix builds can easily exceed one page.
	for page := 1; ; page++ {
		var result struct {
			TotalCount int                 `json:"total_count"`
			Jobs       []*models.GithubJob `json:"jobs"`
		}
		query := url.Values{"per_page": {"100"}, "page": {itoa(int64(page))}}
		if err := self.getJSON(ctx, self.repoPath("actions/runs/%d/jobs", runId), query, &result); err != nil {
			return nil, err
		}
		jobs = append(jobs, result.Jobs...)
		if len(result.Jobs) == 0 || len(jobs) >= result.TotalCount {
			return jobs, nil
		}
	}
}

// GetJobLog returns the plain-text log of a job. GitHub only serves it once
// the job has completed.
func (self *Client) GetJobLog(ctx context.Context, jobId int64) (string, error) {
	body, err := self.do(ctx, http.MethodGet, self.repoPath("actions/jobs/%d/logs", jobId), nil, nil, "")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ListAnnotations returns the errors, warnings, and notices a job reported.
func (self *Client) ListAnnotations(ctx context.Context, jobId int64) ([]*models.GithubAnnotation, error) {
	var annotations []*models.GithubAnnotation
	// A job is a check run as far as this endpoint is concerned.
	err := self.getJSON(ctx, self.repoPath("check-runs/%d/annotations", jobId), url.Values{"per_page": {"100"}}, &annotations)
	return annotations, err
}

func (self *Client) ListWorkflows(ctx context.Context) ([]*models.GithubWorkflow, error) {
	var result struct {
		Workflows []*models.GithubWorkflow `json:"workflows"`
	}
	if err := self.getJSON(ctx, self.repoPath("actions/workflows"), url.Values{"per_page": {"100"}}, &result); err != nil {
		return nil, err
	}
	return result.Workflows, nil
}

func (self *Client) ListArtifacts(ctx context.Context, runId int64) ([]*models.GithubArtifact, error) {
	var result struct {
		Artifacts []*models.GithubArtifact `json:"artifacts"`
	}
	if err := self.getJSON(ctx, self.repoPath("actions/runs/%d/artifacts", runId), url.Values{"per_page": {"100"}}, &result); err != nil {
		return nil, err
	}
	return result.Artifacts, nil
}

// DownloadArtifact returns the artifact's zip archive.
func (self *Client) DownloadArtifact(ctx context.Context, artifactId int64) ([]byte, error) {
	return self.do(ctx, http.MethodGet, self.repoPath("actions/artifacts/%d/zip", artifactId), nil, nil, "")
}

func debugLoggingBody(enableDebugLogging bool) any {
	if !enableDebugLogging {
		return nil
	}
	return map[string]bool{"enable_debug_logging": true}
}

// RerunRun re-runs all jobs of a run.
func (self *Client) RerunRun(ctx context.Context, runId int64, enableDebugLogging bool) error {
	return self.post(ctx, self.repoPath("actions/runs/%d/rerun", runId), debugLoggingBody(enableDebugLogging))
}

// RerunFailedJobs re-runs the failed jobs of a run, plus the jobs that depend
// on them.
func (self *Client) RerunFailedJobs(ctx context.Context, runId int64, enableDebugLogging bool) error {
	return self.post(ctx, self.repoPath("actions/runs/%d/rerun-failed-jobs", runId), debugLoggingBody(enableDebugLogging))
}

// RerunJob re-runs a single job, plus the jobs that depend on it.
func (self *Client) RerunJob(ctx context.Context, jobId int64, enableDebugLogging bool) error {
	return self.post(ctx, self.repoPath("actions/jobs/%d/rerun", jobId), debugLoggingBody(enableDebugLogging))
}

func (self *Client) CancelRun(ctx context.Context, runId int64) error {
	return self.post(ctx, self.repoPath("actions/runs/%d/cancel", runId), nil)
}

// ForceCancelRun cancels a run even if it is stuck, bypassing conditions like
// `always()` that would otherwise keep a job running.
func (self *Client) ForceCancelRun(ctx context.Context, runId int64) error {
	return self.post(ctx, self.repoPath("actions/runs/%d/force-cancel", runId), nil)
}

func (self *Client) DeleteRun(ctx context.Context, runId int64) error {
	return self.delete(ctx, self.repoPath("actions/runs/%d", runId))
}

// DispatchWorkflow triggers a workflow that has a workflow_dispatch trigger.
func (self *Client) DispatchWorkflow(ctx context.Context, workflowId int64, ref string, inputs map[string]string) error {
	body := map[string]any{"ref": ref}
	if len(inputs) > 0 {
		body["inputs"] = inputs
	}
	return self.post(ctx, self.repoPath("actions/workflows/%d/dispatches", workflowId), body)
}

func (self *Client) EnableWorkflow(ctx context.Context, workflowId int64) error {
	return self.put(ctx, self.repoPath("actions/workflows/%d/enable", workflowId))
}

func (self *Client) DisableWorkflow(ctx context.Context, workflowId int64) error {
	return self.put(ctx, self.repoPath("actions/workflows/%d/disable", workflowId))
}

// GetFileContent returns the content of a file in the repository at the given
// ref, e.g. a workflow definition.
func (self *Client) GetFileContent(ctx context.Context, path string, ref string) (string, error) {
	var query url.Values
	if ref != "" {
		query = url.Values{"ref": {ref}}
	}
	escapedPath := lo.Map(strings.Split(path, "/"), func(segment string, _ int) string {
		return url.PathEscape(segment)
	})
	body, err := self.do(ctx, http.MethodGet, self.repoPath("contents/%s", strings.Join(escapedPath, "/")), query, nil, "application/vnd.github.raw+json")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ExtractArtifact unpacks a downloaded artifact's zip archive into dir, the
// way `gh run download` does.
func ExtractArtifact(archive []byte, dir string) error {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		// Refuse paths that would escape dir, like "../../.bashrc".
		target := filepath.Join(dir, filepath.FromSlash(file.Name))
		if !strings.HasPrefix(target, filepath.Clean(dir)+string(filepath.Separator)) {
			return fmt.Errorf("artifact contains an invalid path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := extractFile(file, target); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(file *zip.File, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}
