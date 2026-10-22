# Pull Requests

Lazygit can show the pull requests of the repository you have open on
GitHub, and do most of what `gh pr` does from inside it, without needing
`gh` installed.

The pull requests are a tab of the branches panel, between local branches
and remotes (move it elsewhere with `gui.sidePanels`; it is called
`pullRequests` there). They load the first time the tab is shown, and while
it is shown they reload by themselves every minute, or every 20 seconds
while some of them have checks running.

## Which repository, and logging in

The repository is picked from your GitHub remotes the same way as for the
GitHub Actions view: the one set with `gh repo set-default` if there is one,
otherwise the only GitHub remote, otherwise a remote named `upstream`; if
that still leaves a choice, you are asked when you press `R`.

The token is found the way `gh auth token` finds it, so if you are logged in
with `gh` there is nothing to do. Otherwise the tab says so, and pressing `R`
offers to log you in, in your browser or with a personal access token; see
[GitHub Actions](./GitHub_Actions.md#which-repository-and-logging-in).

## The list

Each pull request shows its state (open, draft, merged or closed), its
number, a `*` if it is checked out (or `•` if you have a local branch for it
that isn't), the state of its checks, its review decision (approved, changes
requested, or review required), its author and its title. Press `+` to
enlarge the panel and also see when it was last updated, its head and base
branches, its size, how many comments it has, and its labels.

The main view shows the selected pull request in roughly the way
`gh pr view` does: who wants to merge what into what, its labels,
milestone, assignees and reviewers, how its checks are doing, whether it can
be merged, its description, its failing checks, its conversation, and its
review threads.

| Key | Action |
|-----|--------|
| `o` | Open the pull request in the browser |
| `y` | Copy its URL, number, title, head branch or description |
| `f` | Choose which pull requests to load: open, closed, merged or all of them; yours, the ones awaiting your review, or the ones assigned to you; or those by an author, with a label, against a base branch, or matching a search in GitHub's search syntax |
| `R` | Reload the pull requests, logging in first if needed |
| `/` | Filter the loaded pull requests by text |
