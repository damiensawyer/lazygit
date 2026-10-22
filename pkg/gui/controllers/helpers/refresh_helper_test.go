package helpers

import (
	"path/filepath"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/context/traits"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stefanhaller/git-todo-parser/todo"
	"github.com/stretchr/testify/assert"
)

func TestCaptureLocalCommitSelectionRange(t *testing.T) {
	testCases := []struct {
		name          string
		commits       []*models.Commit
		selectedIdx   int
		rangeStartIdx int
		expected      *localCommitSelectionRange
	}{
		{
			name:          "captures selected commit and range start",
			commits:       makeCommits("a", "b"),
			selectedIdx:   1,
			rangeStartIdx: 0,
			expected: &localCommitSelectionRange{
				selectedHash:   "b",
				rangeStartHash: "a",
				mode:           traits.RangeSelectModeSticky,
			},
		},
		{
			name:          "ignores invalid range start index",
			commits:       makeCommits("a"),
			selectedIdx:   0,
			rangeStartIdx: 1,
			expected:      nil,
		},
		{
			name:          "ignores empty selected hash",
			commits:       append(makeCommits("a"), makeTodoCommit(todo.UpdateRef)),
			selectedIdx:   1,
			rangeStartIdx: 0,
			expected:      nil,
		},
		{
			name:          "ignores empty range start hash",
			commits:       append(makeCommits("a"), makeTodoCommit(todo.Exec)),
			selectedIdx:   0,
			rangeStartIdx: 1,
			expected:      nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			selectionRange := captureLocalCommitSelectionRange(
				testCase.commits,
				testCase.selectedIdx,
				testCase.rangeStartIdx,
				traits.RangeSelectModeSticky,
			)

			assert.Equal(t, testCase.expected, selectionRange)
		})
	}
}

func TestFindLocalCommitSelectionRange(t *testing.T) {
	type expectation struct {
		selectedIdx   int
		rangeStartIdx int
		found         bool
	}

	selectionRange := localCommitSelectionRange{
		selectedHash:   "b",
		rangeStartHash: "c",
		mode:           traits.RangeSelectModeSticky,
	}

	testCases := []struct {
		name     string
		commits  []*models.Commit
		expected expectation
	}{
		{
			name:    "finds selection after commits are inserted above it",
			commits: makeCommits("new", "a", "b", "c"),
			expected: expectation{
				selectedIdx:   2,
				rangeStartIdx: 3,
				found:         true,
			},
		},
		{
			name:    "finds selection that did not move",
			commits: makeCommits("a", "b", "c"),
			expected: expectation{
				selectedIdx:   1,
				rangeStartIdx: 2,
				found:         true,
			},
		},
		{
			name:     "reports not found when a hash is missing",
			commits:  makeCommits("a", "b"),
			expected: expectation{},
		},
		{
			name: "skips todo entries with the same hash as a selected commit",
			commits: []*models.Commit{
				makeTodoCommitWithHash("b", todo.Revert),
				makeCommits("a")[0],
				makeCommits("b")[0],
				makeCommits("c")[0],
			},
			expected: expectation{
				selectedIdx:   2,
				rangeStartIdx: 3,
				found:         true,
			},
		},
		{
			name: "falls back to a todo entry when the selected commit became one",
			commits: []*models.Commit{
				makeTodoCommitWithHash("b", todo.Pick),
				makeCommits("c")[0],
			},
			expected: expectation{
				selectedIdx:   0,
				rangeStartIdx: 1,
				found:         true,
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			selectedIdx, rangeStartIdx, found := findLocalCommitSelectionRange(testCase.commits, &selectionRange)
			actual := expectation{
				selectedIdx:   selectedIdx,
				rangeStartIdx: rangeStartIdx,
				found:         found,
			}

			assert.Equal(t, testCase.expected, actual)
		})
	}
}

func TestFilterCommitsForOverview(t *testing.T) {
	hashPool := &utils.StringPool{}

	commit := func(hash string, extraInfo string, parents ...string) *models.Commit {
		return models.NewCommit(hashPool, models.NewCommitOpts{
			Hash:      hash,
			ExtraInfo: extraInfo,
			Parents:   parents,
		})
	}

	commits := []*models.Commit{
		commit("master3", "(HEAD -> master)", "master2"),
		commit("master2", "", "M"),
		commit("M", "", "master1", "feature2"),
		commit("feature2", "(feature)", "feature1"),
		commit("feature1", "", "base"),
		commit("master1", "", "base"),
		commit("base", "(tag: v1)"),
	}
	todoCommit := models.NewCommit(hashPool, models.NewCommitOpts{Hash: "todo", Action: todo.Pick})
	commits = append([]*models.Commit{todoCommit}, commits...)

	kept, indices := filterCommitsForOverview(commits)

	// merge commits, ref tips, and rebase todo entries are kept; everything
	// else is hidden
	assert.Equal(t, []string{"todo", "master3", "M", "feature2", "base"},
		lo.Map(kept, func(commit *models.Commit, _ int) string { return commit.Hash() }))

	// each kept commit's index points back at it in the full slice
	assert.Equal(t, []int{0, 1, 3, 4, 7}, indices)
	for i, modelIdx := range indices {
		assert.Same(t, commits[modelIdx], kept[i])
	}
}

func TestFilterCommitsForOverviewKeepingNothing(t *testing.T) {
	hashPool := &utils.StringPool{}
	commits := []*models.Commit{
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: "a", Parents: []string{"b"}}),
		models.NewCommit(hashPool, models.NewCommitOpts{Hash: "b"}),
	}

	for _, commits := range [][]*models.Commit{commits, {}} {
		kept, indices := filterCommitsForOverview(commits)

		// non-nil, so that the model can tell an empty overview from overview
		// mode being off
		assert.NotNil(t, kept)
		assert.Empty(t, kept)
		assert.NotNil(t, indices)
		assert.Empty(t, indices)
	}
}

func TestFindNewConflictedCommit(t *testing.T) {
	testCases := []struct {
		name            string
		previousCommits []*models.Commit
		commits         []*models.Commit
		expectedIdx     *int
	}{
		{
			name:            "finds a newly conflicted commit",
			previousCommits: makeCommits("a", "b"),
			commits: []*models.Commit{
				makeCommits("a")[0],
				makeConflictedCommit("b"),
			},
			expectedIdx: lo.ToPtr(1),
		},
		{
			name: "finds a different conflicted commit",
			previousCommits: []*models.Commit{
				makeConflictedCommit("a"),
			},
			commits: []*models.Commit{
				makeConflictedCommit("b"),
			},
			expectedIdx: lo.ToPtr(0),
		},
		{
			name: "ignores the same conflicted commit",
			previousCommits: []*models.Commit{
				makeConflictedCommit("a"),
			},
			commits: []*models.Commit{
				makeConflictedCommit("a"),
			},
			expectedIdx: nil,
		},
		{
			name:            "reports not found when there is no conflict",
			previousCommits: makeCommits("a"),
			commits:         makeCommits("a", "b"),
			expectedIdx:     nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			idx := findNewConflictedCommit(testCase.previousCommits, testCase.commits)

			assert.Equal(t, testCase.expectedIdx != nil, idx != nil)
			if idx != nil {
				assert.Equal(t, *testCase.expectedIdx, *idx)
			}
		})
	}
}

func TestMarkWorktreeFiles(t *testing.T) {
	worktreePath := filepath.Join("/", "path", "to", "repo")
	worktrees := []*models.Worktree{
		{Path: worktreePath},
		{Path: filepath.Join(worktreePath, "worktree1")},
		{Path: filepath.Join(worktreePath, "dir", "worktree2")},
		{Path: filepath.Join("/", "path", "to", "worktree3")},
	}

	t.Run("marks the files that are worktrees", func(t *testing.T) {
		files := []*models.File{
			{Path: "file"},
			{Path: "worktree1"},
			{Path: "dir/worktree2"},
			{Path: "dir"},
		}

		assert.True(t, markWorktreeFiles(files, worktrees, worktreePath))
		assert.Equal(t, []*models.File{
			{Path: "file"},
			{Path: "worktree1", IsWorktree: true},
			{Path: "dir/worktree2", IsWorktree: true},
			{Path: "dir"},
		}, files)
	})

	t.Run("reports no change when there is nothing to mark", func(t *testing.T) {
		files := []*models.File{{Path: "file"}, {Path: "dir"}}

		assert.False(t, markWorktreeFiles(files, worktrees, worktreePath))
	})

	t.Run("unmarks a file whose worktree is gone", func(t *testing.T) {
		files := []*models.File{{Path: "worktree1", IsWorktree: true}}

		assert.True(t, markWorktreeFiles(files, nil, worktreePath))
		assert.Equal(t, []*models.File{{Path: "worktree1"}}, files)
	})
}

func makeCommits(hashes ...string) []*models.Commit {
	hashPool := &utils.StringPool{}
	return lo.Map(hashes, func(hash string, _ int) *models.Commit {
		return models.NewCommit(hashPool, models.NewCommitOpts{Hash: hash})
	})
}

func makeTodoCommit(action todo.TodoCommand) *models.Commit {
	return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Action: action})
}

func makeTodoCommitWithHash(hash string, action todo.TodoCommand) *models.Commit {
	return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: hash, Action: action})
}

func makeConflictedCommit(hash string) *models.Commit {
	return models.NewCommit(&utils.StringPool{}, models.NewCommitOpts{Hash: hash, Status: models.StatusConflicted})
}
