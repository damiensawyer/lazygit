# GitHub Actions

Lazygit can show the GitHub Actions workflow runs of the repository you have
open, and do most of what `gh run` and `gh workflow` do from inside it: watch
runs, read their logs, re-run or cancel them, download their artifacts, and
trigger workflows.

Press `<ctrl+a>` (configurable as `keybinding.universal.openGithubActions`) to
open the runs list. It takes the place of the commits panel; press `esc` to
get the commits back. While it is shown it refreshes itself, every few seconds
while a run is in progress and every half minute otherwise.

## Which repository, and logging in

The repository is picked from your GitHub remotes the same way as for pull
requests: the one set with `gh repo set-default` if there is one, otherwise
the only GitHub remote, otherwise a remote named `upstream`; if that still
leaves a choice, you are asked.

The token is the one the GitHub CLI uses, if you are logged in with `gh`. If
not, lazygit offers to log you in, either in your browser (the same device
login `gh auth login --web` uses) or by pasting a personal access token. The
token is stored in the GitHub CLI's `hosts.yml`, so `gh` will use it too.

## Runs

The main view summarises the selected run like `gh run view`: what triggered
it, its jobs, the steps of those that failed, and the annotations (compiler
errors, test failures) that they reported.

| Key | Action |
|-----|--------|
| `enter` | Show the run's jobs |
| `o` | Open the run in the browser |
| `r` | Re-run all jobs or only the failed ones, optionally with debug logging |
| `c` | Cancel or force-cancel the run |
| `d` | Delete the run |
| `f` | Only show runs of the current or another branch, a workflow, a status, an event, or the commit selected in the commits panel |
| `w` | Workflows: run one (asking for its inputs), enable or disable it, edit it, or only show its runs |
| `a` | Download the run's artifacts into the working tree |
| `y` | Copy the run's URL, ID, commit hash or branch |
| `R` | Refresh |
| `/` | Filter the loaded runs by text |

## Jobs and steps

`enter` on a run lists its jobs, and the main view shows the selected job's
steps and log; for a failed job, only the log of the steps that failed.
`enter` on a job lists its steps, starting at the first one that failed, and
the main view shows just that step's part of the log. Logs become available
once a job has completed.

| Key | Action |
|-----|--------|
| `enter` | On a job: show its steps |
| `o` | Open the job in the browser (on a step: at that step) |
| `r` | On a job: re-run it, optionally with debug logging |
| `e` | Open the job's log in your editor (on a step: at that step) |
| `y` | Copy the job's URL, ID or log, or the step's log |
