# lazygit (bots are going to kill us all edition)

A fork of [jesseduffield/lazygit](https://github.com/jesseduffield/lazygit) with extra features that aren't in upstream. Upstream stopped accepting pull requests, so these live here instead. The fork tracks upstream's `master`, so everything upstream lazygit does still works the same way. For the basics (installation, configuration, all the standard keybindings) see the [upstream README](https://github.com/jesseduffield/lazygit#readme).

It should be pretty obvious I didn't write any of this. I'm an old coder who misses the old ways of vim key map coffee mugs, water cooler chats about the pros and cons of monads, multiple viewings of Silicon Valley and actually having to think about something slowly and carefully before the computer would do what you wanted.

But hey - the times, they be changing.

So, with that, here's *Lazygit - crazy bot edition*.
Go wild my brothers and sisters in digital arms. If we can dream it, we can have it.

This page covers only what this fork adds.

- [GitHub Actions](#github-actions)
- [Browse all files at a commit (killer feature that git extensions on Windows had!)](#browse-all-files-at-a-commit)
- [Commit overview mode](#commit-overview-mode)
- [Show the final version of changed files](#show-the-final-version-of-changed-files)
- [Full-file diff toggle](#full-file-diff-toggle)
- [Searchable keybindings menu](#searchable-keybindings-menu)
- [Resizable side panels](#resizable-side-panels)
- [Installation](#installation)

## GitHub Actions

You can watch your repository's workflow runs, read their logs, and do most of what `gh run` and `gh workflow` do without leaving lazygit.

![GitHub Actions](demo/fork/gifs/github_actions.gif)

Press `<ctrl+a>` in a repo that has a GitHub remote. The runs list replaces the commits panel; press `<esc>` to get the commits back. While the list is open it refreshes itself.

**Runs**

| Key | Action |
|-----|--------|
| `<enter>` | Show the run's jobs |
| `o` | Open the run in the browser |
| `r` | Re-run all jobs or only the failed ones, optionally with debug logging |
| `c` | Cancel or force-cancel the run |
| `d` | Delete the run |
| `f` | Filter runs by branch, workflow, status, triggering event, or commit |
| `w` | Workflows: run one (it asks for the inputs), enable or disable one, edit one, or show only its runs |
| `a` | Download the run's artifacts |
| `y` | Copy the run's URL, ID, commit hash or branch |
| `R` | Refresh |
| `/` | Filter the loaded runs by text |

**Jobs and steps**

| Key | Action |
|-----|--------|
| `<enter>` | On a job: show its steps (it opens at the first failed step) |
| `o` | Open the job in the browser (on a step, at that step) |
| `r` | On a job: re-run it and the jobs that depend on it |
| `e` | Open the job's log in your editor (on a step, at that step) |
| `y` | Copy the job's URL, ID or log, or the step's log |
| `<esc>` | Go back up a level |

It uses the same token as the `gh` CLI. If you aren't logged in with `gh`, lazygit offers a browser login (the same device flow `gh auth login --web` uses) or lets you paste a personal access token. The token is saved to `gh`'s `hosts.yml`, so `gh` can use it too.

Full reference: [docs-master/GitHub_Actions.md](docs-master/GitHub_Actions.md)

## Browse all files at a commit

You can browse the **whole repository** as it was at any point in history, not just the files that a commit changed.

![Browse all files at a commit](demo/fork/gifs/browse_files.gif)

Press `E` on a commit, branch, remote branch, tag, stash entry, or reflog entry. Press `<esc>` to go back.

| Key | Action |
|-----|--------|
| `<enter>` | On a directory: expand or collapse it. On a file: focus its content so you can scroll it |
| `/` | Filter the tree by path |
| `` ` `` | Switch between tree and flat layout |
| `-` / `=` | Collapse / expand all directories |
| `c` | **Restore to worktree**: put the selected files or directories back to how they were at this commit. The result is left unstaged |
| `S` | **Save**: write a copy of the file (or of every file in a directory) as it was at this commit, either to a path you choose or over the original |
| `y` | Copy the file name, relative or absolute path, file content at this commit, or blob hash |
| `<ctrl+s>` | Filter the commit log to the selected file's history |
| `o` / `e` | Open / edit the file as it is in your working tree now |

Full reference: [docs-master/Browsing_All_Files_At_Commit.md](docs-master/Browsing_All_Files_At_Commit.md)

## Commit overview mode

In repos with lots of merged branches, the ordinary commits can hide the shape of the history. Overview mode hides them and leaves only the landmarks: merge commits, and the commits that a branch, tag, or `HEAD` points to.

![Commit overview mode](demo/fork/gifs/overview_mode.gif)

| Key | Where | Action |
|-----|-------|--------|
| `~` | Commits panel | Toggle overview mode (it's also in the `<ctrl+l>` log options menu) |

The graph keeps the same shape as it has in the full log. When overview mode is on, the commits panel title says `(overview)` and the bottom line has a reset link. If you turn it off without moving the selection, you jump back to the commit you were on.

## Show the final version of changed files

A diff shows what changed. Sometimes you'd rather read the files as they ended up. This toggle swaps the main view from diffs to the full, syntax-highlighted contents of each changed file. You can keep moving through the file list to read each file in turn.

![Show the final version of changed files](demo/fork/gifs/final_version.gif)

| Key | Where | Action |
|-----|-------|--------|
| `<alt+\>` | Files panel, or a commit's files | Toggle between diffs and final file contents |

For working-tree changes, it shows the file as it is on disk. For a commit's files, it shows the file as of that commit.

By default the highlighting uses your terminal's 16 ANSI colors, so it follows your terminal theme. To use a [Chroma style](https://xyproto.github.io/splash/docs/) instead, set it in your config:

```yaml
gui:
  syntaxHighlightStyle: monokai
```

## Full-file diff toggle

Shows the entire file around the changes, then goes back to your normal context size.

![Full-file diff toggle](demo/fork/gifs/full_file_diff.gif)

| Key | Where | Action |
|-----|-------|--------|
| `<alt+}>` or `<alt+]>` | Anywhere a diff is shown | Toggle full-file context. Press it again to restore the previous context size |

This is the same as pressing `}` until the whole file shows, but it remembers your previous setting.

## Searchable keybindings menu

Press `?`, then type. The keybindings menu filters as you type, so you don't need to press `/` first.

![Searchable keybindings menu](demo/fork/gifs/keybindings_search.gif)

| Key | Action |
|-----|--------|
| `?` | Open the keybindings menu |
| *type* | Filter the list |
| `<backspace>` | Delete the last character of the filter |
| `<esc>` | Clear the filter. Press it again to close the menu |

## Resizable side panels

You can change the width of the side panels from the keyboard in both normal and half screen mode. The width is saved for each repository.

![Resizable side panels](demo/fork/gifs/side_panel_width.gif)

| Key | Action |
|-----|--------|
| `<alt+shift+right>` | Widen the side panels |
| `<alt+shift+left>` | Narrow the side panels |
| `<alt+shift+=>` | Reset the width to the configured default |

The defaults come from `gui.sidePanelWidth` (normal mode) and `gui.halfScreenModeSidePanelWidth` (half screen mode, default `0.5`). Full screen mode has no side/main split, so these keys do nothing there.

## Installation

Build it from source. You'll need [Go](https://go.dev/doc/install).

```sh
git clone https://github.com/damiensawyer/lazygit.git
cd lazygit
go install
```

This installs the binary as `lazygit` in `$(go env GOPATH)/bin`, so it replaces any upstream lazygit installed with `go install`. Releases and package-manager builds (Homebrew, pacman, etc.) are upstream's and don't include these features.

### Re-recording the GIFs

The demos above are recorded with [vhs](https://github.com/charmbracelet/vhs) from the demo integration tests in `pkg/integration/tests/demo`. To regenerate them after changing a feature, run:

```sh
demo/fork/record.sh               # all of them
demo/fork/record.sh overview_mode # or just some
```
