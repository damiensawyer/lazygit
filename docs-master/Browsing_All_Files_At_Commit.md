# Browsing all files at a commit

Lazygit lets you explore the entire repository exactly as it existed at any
commit — not just the files the commit changed. Press `E` (configurable as
`keybinding.universal.browseAllFilesAtCommit`) on a commit, branch, tag, stash
entry, reflog entry, or remote branch to open the panel.

The panel opens with just the top level of the repository expanded, like a
file explorer. Selecting a file shows its content (at that commit) in the main
view; selecting a directory shows a listing of its entries. Binary files show
a size placeholder, symlinks show their target, and submodules show the commit
they were pinned to. Press `esc` to return to the panel you came from.

Because a commit's tree never changes, re-entering the panel for the same
commit is instant and remembers which directories you had expanded.

## Navigation

| Key | Action |
|-----|--------|
| `enter` | On a directory: toggle collapsed. On a file: focus the main view to scroll its content |
| `` ` `` | Toggle between tree and flat layout |
| `-` / `=` | Collapse / expand all directories |
| `/` | Filter the tree by path (fuzzy, like other panels) |

## Actions

| Key | Action |
|-----|--------|
| `c` | **Restore to worktree**: replace the selected files/directories in your working tree with their state at this commit, using `git restore --source`. The restored versions are left *unstaged*, as if you had edited the files yourself, so you can review them in the files panel before staging. Works on directories and multi-selections |
| `S` | **Save**: write a copy of the selected file (or every file under the selected directory) as it existed at this commit, either to a path you enter or over the original location. Smudge/eol filters are applied as they would be on checkout |
| `y` | Copy menu: file name, relative/absolute path, file content at this commit, blob hash |
| `<ctrl+s>` | Filtering menu: with a file selected, offers to filter the commit log by that file's path — i.e. the file's history |
| `o` / `e` | Open / edit the *current working tree* version of the file (disabled if it no longer exists) |
