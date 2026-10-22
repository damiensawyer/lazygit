package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeybindingPlatform(t *testing.T) {
	scenarios := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "Not set falls back to the host OS",
			envValue: "",
			expected: runtime.GOOS,
		},
		{
			name:     "darwin is honored",
			envValue: "darwin",
			expected: "darwin",
		},
		{
			name:     "linux is honored",
			envValue: "linux",
			expected: "linux",
		},
		{
			name:     "windows is honored",
			envValue: "windows",
			expected: "windows",
		},
		{
			name:     "An unrecognized value falls back to the host OS",
			envValue: "mac",
			expected: runtime.GOOS,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			t.Setenv("LAZYGIT_KEYBINDING_PLATFORM", s.envValue)
			assert.Equal(t, s.expected, KeybindingPlatform())
		})
	}
}

func TestMigrationOfRenamedKeys(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name:              "Empty String",
			input:             "",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "No rename needed",
			input: `foo:
  bar: 5
`,
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "Rename one",
			input: `gui:
  skipUnstageLineWarning: true
`,
			expected: `gui:
  skipDiscardChangeWarning: true
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Renamed 'gui.skipUnstageLineWarning' to 'skipDiscardChangeWarning'"},
		},
		{
			name: "Rename several",
			input: "gui:\n" +
				"  windowSize: half\n" +
				"  skipUnstageLineWarning: true\n" +
				"keybinding:\n" +
				"  universal:\n" +
				"    executeCustomCommand: a\n" +
				"    cyclePagers: b\n" +
				"    cyclePagersReverse: c\n",
			expected: "gui:\n" +
				"  screenMode: half\n" +
				"  skipDiscardChangeWarning: true\n" +
				"keybinding:\n" +
				"  universal:\n" +
				"    executeShellCommand: a\n" +
				"    cycleDiffRenderers: b\n" +
				"    cycleDiffRenderersReverse: c\n",
			expectedDidChange: true,
			expectedChanges: []string{
				"Renamed 'gui.skipUnstageLineWarning' to 'skipDiscardChangeWarning'",
				"Renamed 'keybinding.universal.executeCustomCommand' to 'executeShellCommand'",
				"Renamed 'keybinding.universal.cyclePagers' to 'cycleDiffRenderers'",
				"Renamed 'keybinding.universal.cyclePagersReverse' to 'cycleDiffRenderersReverse'",
				"Renamed 'gui.windowSize' to 'screenMode'",
			},
		},
		{
			name: "Rename staging view options",
			input: `gui:
  wrapLinesInStagingView: false
  useHunkModeInStagingView: true
`,
			expected: `gui:
  wrapLinesInDiffView: false
  useHunkModeInDiffView: true
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Renamed 'gui.wrapLinesInStagingView' to 'wrapLinesInDiffView'",
				"Renamed 'gui.useHunkModeInStagingView' to 'useHunkModeInDiffView'",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestMigrationOfMovedKeys(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name:              "Empty String",
			input:             "",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "No move needed",
			input: `foo:
  bar: 5
`,
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "Move worktree keybinding into the universal section",
			input: `keybinding:
  universal:
    quit: q
  worktrees:
    viewWorktreeOptions: w
`,
			expected: `keybinding:
  universal:
    quit: q
    newWorktree: w
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Moved 'keybinding.worktrees.viewWorktreeOptions' to 'keybinding.universal.newWorktree'"},
		},
		{
			name: "Create the universal section if it doesn't exist",
			input: `keybinding:
  worktrees:
    viewWorktreeOptions: w
`,
			expected: `keybinding:
  universal:
    newWorktree: w
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Moved 'keybinding.worktrees.viewWorktreeOptions' to 'keybinding.universal.newWorktree'"},
		},
		{
			name: "Move author and branch colors into the theme",
			input: `gui:
  authorColors:
    John Smith: red
  theme:
    activeBorderColor:
      - green
  branchColorPatterns:
    ^docs/: blue
`,
			expected: `gui:
  theme:
    activeBorderColor:
      - green
    authorColors:
      John Smith: red
    branchColorPatterns:
      ^docs/: blue
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Moved 'gui.authorColors' to 'gui.theme.authorColors'",
				"Moved 'gui.branchColorPatterns' to 'gui.theme.branchColorPatterns'",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestMigrateNullKeybindingsToDisabled(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name:              "Empty String",
			input:             "",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "No change needed",
			input: `keybinding:
  universal:
    quit: q
`,
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "Change one",
			input: `keybinding:
  universal:
    quit: null
`,
			expected: `keybinding:
  universal:
    quit: <disabled>
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Changed 'null' to '<disabled>' for keybinding 'keybinding.universal.quit'"},
		},
		{
			name: "Change several",
			input: `keybinding:
  universal:
    quit: null
    return: <esc>
    new: null
`,
			expected: `keybinding:
  universal:
    quit: <disabled>
    return: <esc>
    new: <disabled>
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Changed 'null' to '<disabled>' for keybinding 'keybinding.universal.quit'",
				"Changed 'null' to '<disabled>' for keybinding 'keybinding.universal.new'",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestCommitPrefixMigrations(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name:              "Empty String",
			input:             "",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "Single CommitPrefix Rename",
			input: `git:
  commitPrefix:
     pattern: "^\\w+-\\w+.*"
     replace: '[JIRA $0] '
`,
			expected: `git:
  commitPrefix:
    - pattern: "^\\w+-\\w+.*"
      replace: '[JIRA $0] '
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Changed 'git.commitPrefix' to an array of strings"},
		},
		{
			name: "Complicated CommitPrefixes Rename",
			input: `git:
  commitPrefixes:
    foo:
      pattern: "^\\w+-\\w+.*"
      replace: '[OTHER $0] '
    CrazyName!@#$^*&)_-)[[}{f{[]:
      pattern: "^foo.bar*"
      replace: '[FUN $0] '
`,
			expected: `git:
  commitPrefixes:
    foo:
      - pattern: "^\\w+-\\w+.*"
        replace: '[OTHER $0] '
    CrazyName!@#$^*&)_-)[[}{f{[]:
      - pattern: "^foo.bar*"
        replace: '[FUN $0] '
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Changed 'git.commitPrefixes' elements to arrays of strings"},
		},
		{
			name:              "Incomplete Configuration",
			input:             "git:",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "No changes made when already migrated",
			input: `
git:
   commitPrefix:
    - pattern: "Hello World"
      replace: "Goodbye"
   commitPrefixes:
    foo:
      - pattern: "^\\w+-\\w+.*"
        replace: '[JIRA $0] '`,
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestCustomCommandsOutputMigration(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name:              "Empty String",
			input:             "",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "Convert subprocess to output=terminal",
			input: `customCommands:
  - command: echo 'hello'
    subprocess: true
  `,
			expected: `customCommands:
  - command: echo 'hello'
    output: terminal
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Changed 'subprocess: true' to 'output: terminal' in custom command"},
		},
		{
			name: "Convert stream to output=log",
			input: `customCommands:
  - command: echo 'hello'
    stream: true
  `,
			expected: `customCommands:
  - command: echo 'hello'
    output: log
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Changed 'stream: true' to 'output: log' in custom command"},
		},
		{
			name: "Convert showOutput to output=popup",
			input: `customCommands:
  - command: echo 'hello'
    showOutput: true
  `,
			expected: `customCommands:
  - command: echo 'hello'
    output: popup
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Changed 'showOutput: true' to 'output: popup' in custom command"},
		},
		{
			name: "Subprocess wins over the other two",
			input: `customCommands:
  - command: echo 'hello'
    subprocess: true
    stream: true
    showOutput: true
  `,
			expected: `customCommands:
  - command: echo 'hello'
    output: terminal
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Changed 'subprocess: true' to 'output: terminal' in custom command",
				"Deleted redundant 'stream: true' property in custom command",
				"Deleted redundant 'showOutput: true' property in custom command",
			},
		},
		{
			name: "Stream wins over showOutput",
			input: `customCommands:
  - command: echo 'hello'
    stream: true
    showOutput: true
  `,
			expected: `customCommands:
  - command: echo 'hello'
    output: log
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Changed 'stream: true' to 'output: log' in custom command",
				"Deleted redundant 'showOutput: true' property in custom command",
			},
		},
		{
			name: "Explicitly setting to false doesn't create an output=none key",
			input: `customCommands:
  - command: echo 'hello'
    subprocess: false
    stream: false
    showOutput: false
  `,
			expected: `customCommands:
  - command: echo 'hello'
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Deleted redundant 'subprocess: false' in custom command",
				"Deleted redundant 'stream: false' property in custom command",
				"Deleted redundant 'showOutput: false' property in custom command",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestAllBranchesLogCmdMigrations(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name:              "Incomplete Configuration Passes uneventfully",
			input:             "git:",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "Single Cmd with no Cmds",
			input: `git:
  allBranchesLogCmd: git log --graph --oneline
`,
			expected: `git:
  allBranchesLogCmds:
    - git log --graph --oneline
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Created git.allBranchesLogCmds array containing value of git.allBranchesLogCmd",
				"Removed obsolete git.allBranchesLogCmd",
			},
		},
		{
			name: "Cmd with one existing Cmds",
			input: `git:
  allBranchesLogCmd: git log --graph --oneline
  allBranchesLogCmds:
    - git log --graph --oneline --pretty
`,
			expected: `git:
  allBranchesLogCmds:
    - git log --graph --oneline
    - git log --graph --oneline --pretty
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Prepended git.allBranchesLogCmd value to git.allBranchesLogCmds array",
				"Removed obsolete git.allBranchesLogCmd",
			},
		},
		{
			name: "Only Cmds set have no changes",
			input: `git:
  allBranchesLogCmds:
    - git log
`,
			expected:        "",
			expectedChanges: []string{},
		},
		{
			name: "Removes Empty Cmd When at end of yaml",
			input: `git:
  allBranchesLogCmds:
    - git log --graph --oneline
  allBranchesLogCmd:
`,
			expected: `git:
  allBranchesLogCmds:
    - git log --graph --oneline
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Removed obsolete git.allBranchesLogCmd"},
		},
		{
			name: "Migrates when sequence defined inline",
			input: `git:
  allBranchesLogCmds: [foo, bar]
  allBranchesLogCmd: baz
`,
			expected: `git:
  allBranchesLogCmds: [baz, foo, bar]
`,
			expectedDidChange: true,
			expectedChanges: []string{
				"Prepended git.allBranchesLogCmd value to git.allBranchesLogCmds array",
				"Removed obsolete git.allBranchesLogCmd",
			},
		},
		{
			name: "Removes Empty Cmd With Keys Afterwards",
			input: `git:
  allBranchesLogCmds:
    - git log --graph --oneline
  allBranchesLogCmd:
  foo: bar
`,
			expected: `git:
  allBranchesLogCmds:
    - git log --graph --oneline
  foo: bar
`,
			expectedDidChange: true,
			expectedChanges:   []string{"Removed obsolete git.allBranchesLogCmd"},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestPagerMigration(t *testing.T) {
	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		// Migrate 'paging' to 'pagers' array
		{
			name:              "Incomplete Configuration Passes uneventfully",
			input:             "git:",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "No paging section",
			input: "git:\n" +
				"  autoFetch: true\n",
			expected: "git:\n" +
				"  autoFetch: true\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "paging is not an object",
			input: "git:\n" +
				"  paging: 5\n",
			expected: "git:\n" +
				"  paging: 5\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "pagers is not an array",
			input: "git:\n" +
				"  pagers: 5\n",
			expected: "git:\n" +
				"  pagers: 5\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "paging and pagers coexist",
			input: "git:\n" +
				"  paging:\n" +
				"    pager: delta --dark --paging=never\n" +
				"  pagers:\n" +
				"    - pager: diff-so-fancy\n",
			expected: "git:\n" +
				"  paging:\n" +
				"    pager: delta --dark --paging=never\n" +
				"  diffRenderers:\n" +
				"    - command: diff-so-fancy\n",
			expectedDidChange: true,
			expectedChanges: []string{
				"Renamed git.pagers to git.diffRenderers",
				"Renamed 'pager' to 'command' in git pager",
			},
		},
		{
			name: "paging and diffRenderers coexist",
			input: "git:\n" +
				"  paging:\n" +
				"    pager: delta --dark --paging=never\n" +
				"  diffRenderers:\n" +
				"    - command: diff-so-fancy\n",
			expected: "git:\n" +
				"  paging:\n" +
				"    pager: delta --dark --paging=never\n" +
				"  diffRenderers:\n" +
				"    - command: diff-so-fancy\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "pagers and diffRenderers coexist",
			input: "git:\n" +
				"  pagers:\n" +
				"    - pager: delta --dark --paging=never\n" +
				"  diffRenderers:\n" +
				"    - command: diff-so-fancy\n",
			expected: "git:\n" +
				"  pagers:\n" +
				"    - pager: delta --dark --paging=never\n" +
				"  diffRenderers:\n" +
				"    - command: diff-so-fancy\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "paging is moved to diffRenderers array preserving fields and order",
			input: "git:\n" +
				"  paging:\n" +
				"    name: delta\n" +
				"    colorArg: never\n" +
				"    pager: delta --dark --paging=never\n" +
				"  autoFetch: true\n",
			expected: "git:\n" +
				"  diffRenderers:\n" +
				"    - name: delta\n" +
				"      colorArg: never\n" +
				"      command: delta --dark --paging=never\n" +
				"  autoFetch: true\n",
			expectedDidChange: true,
			expectedChanges: []string{
				"Moved git.paging object to git.pagers array",
				"Renamed git.pagers to git.diffRenderers",
				"Renamed 'pager' to 'command' in git pager",
			},
		},
		{
			name: "paging is moved to diffRenderers array even if empty",
			input: "git:\n" +
				"  paging: {}\n",
			expected: "git:\n" +
				"  diffRenderers:\n" +
				"    - type: rawGit\n",
			expectedDidChange: true,
			expectedChanges: []string{
				"Moved git.paging object to git.pagers array",
				"Renamed git.pagers to git.diffRenderers",
				"Changed git pager without a command to 'type: rawGit'",
			},
		},

		// Migrate 'pagers' array to 'diffRenderers' array
		{
			name: "empty pagers array is renamed",
			input: "git:\n" +
				"  pagers: []\n",
			expected: "git:\n" +
				"  diffRenderers: []\n",
			expectedDidChange: true,
			expectedChanges:   []string{"Renamed git.pagers to git.diffRenderers"},
		},
		{
			name: "pagers array entries are adapted",
			input: "git:\n" +
				"  pagers:\n" +
				"    - name: delta\n" +
				"      colorArg: never\n" +
				"      pager: delta --dark --paging=never\n" +
				"    - name: difft\n" +
				"      colorArg: never\n" +
				"      externalDiffCommand: difft --color=always\n" +
				"    - name: git-config\n" +
				"      colorArg: never\n" +
				"      useExternalDiffGitConfig: TRUE\n" +
				"    - name: git\n" +
				"      colorArg: never\n" +
				"  autoFetch: true\n",
			expected: "git:\n" +
				"  diffRenderers:\n" +
				"    - name: delta\n" +
				"      colorArg: never\n" +
				"      command: delta --dark --paging=never\n" +
				"    - name: difft\n" +
				"      colorArg: never\n" +
				"      command: difft --color=always\n" +
				"      type: extDiff\n" +
				"    - name: git-config\n" +
				"      colorArg: never\n" +
				"      type: extDiff\n" +
				"    - name: git\n" +
				"      colorArg: never\n" +
				"      type: rawGit\n" +
				"  autoFetch: true\n",
			expectedDidChange: true,
			expectedChanges: []string{
				"Renamed git.pagers to git.diffRenderers",
				"Renamed 'pager' to 'command' in git pager",
				"Changed 'externalDiffCommand' to 'command' with 'type: extDiff' in git pager",
				"Changed 'useExternalDiffGitConfig: true' to 'type: extDiff' in git pager",
				"Changed git pager without a command to 'type: rawGit'",
			},
		},
		{
			name: "zero-valued mechanism fields do not take precedence and are removed",
			input: "git:\n" +
				"  pagers:\n" +
				"    - pager: delta --dark --paging=never\n" +
				"      externalDiffCommand: null\n" +
				"      useExternalDiffGitConfig: false\n" +
				"    - pager: \"\"\n" +
				"      externalDiffCommand: difft --color=always\n" +
				"      useExternalDiffGitConfig: false\n" +
				"    - pager: null\n" +
				"      externalDiffCommand: \"\"\n" +
				"      useExternalDiffGitConfig: YES\n" +
				"    - pager: \"\"\n" +
				"      externalDiffCommand:\n" +
				"      useExternalDiffGitConfig: false\n" +
				"    - useExternalDiffGitConfig: null\n",
			expected: "git:\n" +
				"  diffRenderers:\n" +
				"    - command: delta --dark --paging=never\n" +
				"    - command: difft --color=always\n" +
				"      type: extDiff\n" +
				"    - type: extDiff\n" +
				"    - type: rawGit\n" +
				"    - type: rawGit\n",
			expectedDidChange: true,
			expectedChanges: []string{
				"Renamed git.pagers to git.diffRenderers",
				"Renamed 'pager' to 'command' in git pager",
				"Removed empty 'externalDiffCommand' from git pager",
				"Removed 'useExternalDiffGitConfig: false' from git pager",
				"Changed 'externalDiffCommand' to 'command' with 'type: extDiff' in git pager",
				"Removed empty 'pager' from git pager",
				"Changed 'useExternalDiffGitConfig: true' to 'type: extDiff' in git pager",
				"Changed git pager without a command to 'type: rawGit'",
				"Removed empty 'useExternalDiffGitConfig' from git pager",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}

func TestSaveRepoGuiValue(t *testing.T) {
	tests := []struct {
		name     string
		existing string // empty means the file doesn't exist
		key      string
		value    float64
		expected string
	}{
		{
			name:     "creates the file when it doesn't exist",
			existing: "",
			key:      "halfScreenModeSidePanelWidth",
			value:    0.45,
			expected: "gui:\n  halfScreenModeSidePanelWidth: 0.45\n",
		},
		{
			name:     "creates the gui section when it doesn't exist",
			existing: "git:\n  paging:\n    useConfig: true\n",
			key:      "halfScreenModeSidePanelWidth",
			value:    0.45,
			expected: "git:\n  paging:\n    useConfig: true\ngui:\n  halfScreenModeSidePanelWidth: 0.45\n",
		},
		{
			name:     "replaces the value when the key already exists",
			existing: "gui:\n  halfScreenModeSidePanelWidth: 0.5\n",
			key:      "halfScreenModeSidePanelWidth",
			value:    0.6,
			expected: "gui:\n  halfScreenModeSidePanelWidth: 0.6\n",
		},
		{
			name:     "preserves other keys in the gui section",
			existing: "gui:\n  sidePanelWidth: 0.28\n  colorizeCommitReferences: true\n",
			key:      "halfScreenModeSidePanelWidth",
			value:    0.45,
			expected: "gui:\n  sidePanelWidth: 0.28\n  colorizeCommitReferences: true\n  halfScreenModeSidePanelWidth: 0.45\n",
		},
		{
			name:     "only replaces the given key",
			existing: "gui:\n  sidePanelWidth: 0.28\n  halfScreenModeSidePanelWidth: 0.5\n",
			key:      "sidePanelWidth",
			value:    0.33,
			expected: "gui:\n  sidePanelWidth: 0.33\n  halfScreenModeSidePanelWidth: 0.5\n",
		},
		{
			name:     "preserves comments",
			existing: "# my repo config\ngui:\n  sidePanelWidth: 0.2 # narrow\n",
			key:      "sidePanelWidth",
			value:    0.25,
			expected: "# my repo config\ngui:\n  sidePanelWidth: 0.25 # narrow\n",
		},
		{
			name:     "treats a file with only whitespace as empty",
			existing: "\n",
			key:      "sidePanelWidth",
			value:    0.4,
			expected: "gui:\n  sidePanelWidth: 0.4\n",
		},
		{
			name:     "treats a file with only a comment as empty",
			existing: "# nothing here yet\n",
			key:      "sidePanelWidth",
			value:    0.4,
			expected: "gui:\n  sidePanelWidth: 0.4\n",
		},
		{
			name:     "keeps the flow style of an existing gui section",
			existing: "gui: {}\n",
			key:      "sidePanelWidth",
			value:    0.3,
			expected: "gui: {sidePanelWidth: 0.3}\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lazygit.yml")
			if test.existing != "" {
				assert.NoError(t, os.WriteFile(path, []byte(test.existing), 0o644))
			}

			assert.NoError(t, saveRepoGuiValue(path, test.key, test.value))

			actual, err := os.ReadFile(path)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, string(actual))
		})
	}
}

func TestSaveRepoGuiValueErrors(t *testing.T) {
	tests := []struct {
		name          string
		existing      string
		expectedError string
	}{
		{
			name:          "top-level element is a sequence",
			existing:      "- a\n",
			expectedError: "top-level element of the config file is not a mapping",
		},
		{
			name:          "gui section is a scalar",
			existing:      "gui: 3\n",
			expectedError: "the gui section of the config file is not a mapping",
		},
		{
			name:          "file is not valid yaml",
			existing:      "gui: [\n",
			expectedError: "did not find expected node content",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "lazygit.yml")
			assert.NoError(t, os.WriteFile(path, []byte(test.existing), 0o644))

			err := saveRepoGuiValue(path, "sidePanelWidth", 0.4)
			assert.ErrorContains(t, err, test.expectedError)

			// The file is left untouched
			actual, readErr := os.ReadFile(path)
			assert.NoError(t, readErr)
			assert.Equal(t, test.existing, string(actual))
		})
	}

	t.Run("path can't be read", func(t *testing.T) {
		assert.Error(t, saveRepoGuiValue(t.TempDir(), "sidePanelWidth", 0.4))
	})
}

func TestSaveSidePanelWidthsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lazygit.yml")
	appConfig := &AppConfig{}

	assert.NoError(t, appConfig.SaveSidePanelWidth(path, 0.2833))
	assert.NoError(t, appConfig.SaveHalfScreenModeSidePanelWidth(path, 0.45))

	actual, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, "gui:\n  sidePanelWidth: 0.2833\n  halfScreenModeSidePanelWidth: 0.45\n", string(actual))

	userConfig, err := loadUserConfigWithDefaults(
		[]*ConfigFile{{Path: path, Policy: ConfigFilePolicySkipIfMissing}}, true)
	assert.NoError(t, err)
	assert.Equal(t, 0.2833, userConfig.Gui.SidePanelWidth)
	assert.Equal(t, 0.45, userConfig.Gui.HalfScreenModeSidePanelWidth)
}

func TestSavedSidePanelWidthOverridesGlobalConfig(t *testing.T) {
	dir := t.TempDir()
	globalPath := filepath.Join(dir, "config.yml")
	repoPath := filepath.Join(dir, "lazygit.yml")
	assert.NoError(t, os.WriteFile(globalPath,
		[]byte("gui:\n  sidePanelWidth: 0.4\n  halfScreenModeSidePanelWidth: 0.6\n"), 0o644))

	assert.NoError(t, (&AppConfig{}).SaveSidePanelWidth(repoPath, 0.25))

	userConfig, err := loadUserConfigWithDefaults([]*ConfigFile{
		{Path: globalPath, Policy: ConfigFilePolicyErrorIfMissing},
		{Path: repoPath, Policy: ConfigFilePolicySkipIfMissing},
	}, true)
	assert.NoError(t, err)
	assert.Equal(t, 0.25, userConfig.Gui.SidePanelWidth)
	// Not saved per repo, so the global value still applies
	assert.Equal(t, 0.6, userConfig.Gui.HalfScreenModeSidePanelWidth)
}

func TestBranchColorsMigration(t *testing.T) {
	moved := "Moved 'gui.branchColorPatterns' to 'gui.theme.branchColorPatterns'"

	scenarios := []struct {
		name              string
		input             string
		expected          string
		expectedDidChange bool
		expectedChanges   []string
	}{
		{
			name: "No branchColors",
			input: "gui:\n" +
				"  theme:\n" +
				"    branchColorPatterns:\n" +
				"      '^docs/': blue\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "branchColors is not an object",
			input: "gui:\n" +
				"  branchColors: 5\n",
			expectedDidChange: false,
			expectedChanges:   []string{},
		},
		{
			name: "branchColors is converted to patterns",
			input: "gui:\n" +
				"  scrollHeight: 2\n" +
				"  branchColors:\n" +
				"    feature: green\n" +
				"    v1.x: '#ff0000'\n" +
				"    123: red\n" +
				"  mouseEvents: false\n",
			expected: "gui:\n" +
				"  scrollHeight: 2\n" +
				"  mouseEvents: false\n" +
				"  theme:\n" +
				"    branchColorPatterns:\n" +
				"      ^feature(/|$): green\n" +
				"      ^v1\\.x(/|$): '#ff0000'\n" +
				"      ^123(/|$): red\n",
			expectedDidChange: true,
			expectedChanges:   []string{"Converted 'gui.branchColors' to 'gui.branchColorPatterns'", moved},
		},
		{
			name: "branchColors is removed if branchColorPatterns is set",
			input: "gui:\n" +
				"  branchColors:\n" +
				"    feature: green\n" +
				"  branchColorPatterns:\n" +
				"    '^docs/': blue\n",
			expected: "gui:\n" +
				"  theme:\n" +
				"    branchColorPatterns:\n" +
				"      '^docs/': blue\n",
			expectedDidChange: true,
			expectedChanges:   []string{"Removed 'gui.branchColors'; it had no effect because 'gui.branchColorPatterns' is set", moved},
		},
		{
			name: "branchColors replaces an empty branchColorPatterns",
			input: "gui:\n" +
				"  branchColorPatterns: {}\n" +
				"  branchColors:\n" +
				"    feature: green\n",
			expected: "gui:\n" +
				"  theme:\n" +
				"    branchColorPatterns:\n" +
				"      ^feature(/|$): green\n",
			expectedDidChange: true,
			expectedChanges:   []string{"Converted 'gui.branchColors' to 'gui.branchColorPatterns'", moved},
		},
		{
			name: "branchColors replaces a null branchColorPatterns",
			input: "gui:\n" +
				"  branchColorPatterns:\n" +
				"  branchColors:\n" +
				"    feature: green\n",
			expected: "gui:\n" +
				"  theme:\n" +
				"    branchColorPatterns:\n" +
				"      ^feature(/|$): green\n",
			expectedDidChange: true,
			expectedChanges:   []string{"Converted 'gui.branchColors' to 'gui.branchColorPatterns'", moved},
		},
		{
			name: "branchColors is kept if branchColorPatterns is not an object",
			input: "gui:\n" +
				"  branchColorPatterns: 5\n" +
				"  branchColors:\n" +
				"    feature: green\n",
			expected: "gui:\n" +
				"  branchColors:\n" +
				"    feature: green\n" +
				"  theme:\n" +
				"    branchColorPatterns: 5\n",
			expectedDidChange: true,
			expectedChanges:   []string{moved},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			changes := NewChangesSet()
			actual, didChange, err := computeMigratedConfig("path doesn't matter", []byte(s.input), changes)
			assert.NoError(t, err)
			assert.Equal(t, s.expectedDidChange, didChange)
			if didChange {
				assert.Equal(t, s.expected, string(actual))
			}
			assert.Equal(t, s.expectedChanges, changes.ToSliceFromOldest())
		})
	}
}
