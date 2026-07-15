package filetree

import (
	"fmt"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func newTestRepoFileTree(paths []string) *RepoFileTree {
	files := lo.Map(paths, func(path string, _ int) *models.TreeFile {
		return &models.TreeFile{Path: path, Mode: 0o100644}
	})
	return NewRepoFileTree(
		func() []*models.TreeFile { return files },
		common.NewDummyCommon(),
		true,
	)
}

func visiblePaths(tree *RepoFileTree) []string {
	return lo.Map(tree.GetAllItems(), func(node *RepoFileNode, _ int) string {
		return node.GetInternalPath()
	})
}

// The flattened cache must always agree with a fresh walk of the node tree;
// this is the invariant that all the mutating methods must preserve.
func assertFlattenedCacheConsistent(t *testing.T, tree *RepoFileTree) {
	t.Helper()

	freshWalk := tree.GetRoot().Raw().Flatten(tree.CollapsedPaths())[1:] // ignoring root
	assert.Equal(t, len(freshWalk), tree.Len())
	for i, node := range freshWalk {
		assert.Equal(t, node, tree.Get(i).Raw())

		index, found := tree.GetIndexForPath(node.GetInternalPath())
		assert.True(t, found)
		assert.Equal(t, i, index)

		assert.Equal(t, tree.GetRoot().Raw().GetVisualDepthAtIndex(i+1, tree.CollapsedPaths()), tree.GetVisualDepth(i))
	}
}

func TestRepoFileTreeFlattenedCache(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir1/file1",
		"dir1/sub/file2",
		"dir2/deep/nested/file3",
		"file4",
	})

	assert.Equal(t, []string{
		".",
		"./dir1",
		"./dir1/file1",
		"./dir1/sub",
		"./dir1/sub/file2",
		"./dir2/deep/nested",
		"./dir2/deep/nested/file3",
		"./file4",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	tree.ToggleCollapsed("./dir1")
	assert.Equal(t, []string{
		".",
		"./dir1",
		"./dir2/deep/nested",
		"./dir2/deep/nested/file3",
		"./file4",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	_, found := tree.GetIndexForPath("./dir1/file1")
	assert.False(t, found)

	tree.ToggleCollapsed("./dir1")
	assertFlattenedCacheConsistent(t, tree)

	tree.CollapseAll()
	assert.Equal(t, []string{"."}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	tree.ExpandAll()
	assert.Equal(t, 8, tree.Len())
	assertFlattenedCacheConsistent(t, tree)

	tree.ToggleShowTree()
	assert.False(t, tree.InTreeMode())
	assert.Equal(t, []string{
		"./dir1/file1",
		"./dir1/sub/file2",
		"./dir2/deep/nested/file3",
		"./file4",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	tree.ToggleShowTree()
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeCollapseAllToTopLevel(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir1/file1",
		"dir1/sub/file2",
		"dir2/file3",
		"file4",
	})

	tree.CollapseAllToTopLevel()

	assert.Equal(t, []string{
		".",
		"./dir1",
		"./dir2",
		"./file4",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	// directories below the top level were collapsed too, so expanding one
	// level shows collapsed subdirectories
	tree.ToggleCollapsed("./dir1")
	assert.Equal(t, []string{
		".",
		"./dir1",
		"./dir1/file1",
		"./dir1/sub",
		"./dir2",
		"./file4",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeTextFilter(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir1/apple",
		"dir1/banana",
		"dir2/apple",
	})

	tree.SetTextFilter("apple", false)
	assert.Equal(t, []string{
		".",
		"./dir1",
		"./dir1/apple",
		"./dir2",
		"./dir2/apple",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	tree.SetTextFilter("", false)
	assert.Equal(t, 6, tree.Len())
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeVisualDepthWithCompression(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"a/b/c/file1",
		"a/b/c/file2",
	})

	// the root item and "a/b/c" are compressed into a single node at visual
	// depth 0, and the files sit at visual depth 1
	index, found := tree.GetIndexForPath("./a/b/c")
	assert.True(t, found)
	assert.Equal(t, 0, tree.GetVisualDepth(index))

	index, found = tree.GetIndexForPath("./a/b/c/file1")
	assert.True(t, found)
	assert.Equal(t, 1, tree.GetVisualDepth(index))
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeGetFlattenedRange(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir1/file1",
		"dir1/file2",
		"file3",
	})

	all := tree.GetFlattenedRange(0, tree.Len())
	assert.Len(t, all, 5)

	sub := tree.GetFlattenedRange(1, 3)
	assert.Len(t, sub, 2)
	assert.Equal(t, "./dir1", sub[0].Node.GetInternalPath())
	assert.Equal(t, "./dir1/file1", sub[1].Node.GetInternalPath())

	assert.Empty(t, tree.GetFlattenedRange(3, 3))
	assert.Len(t, tree.GetFlattenedRange(-5, 100), 5)
}

// Exercises tree building and the flattened-cache rebuild at the scale of a
// very large repository (300k files). Cursor movement and rendering don't
// appear here because they are O(1)/O(viewport) against the cache.
func BenchmarkRepoFileTreeAtScale(b *testing.B) {
	files := make([]*models.TreeFile, 0, 300_000)
	for i := range 300_000 {
		files = append(files, &models.TreeFile{
			Path: fmt.Sprintf("dir%d/sub%d/file%d.txt", i%1000, i%37, i),
			Mode: 0o100644,
		})
	}
	tree := NewRepoFileTree(
		func() []*models.TreeFile { return files },
		common.NewDummyCommon(),
		true,
	)

	b.Run("SetTree", func(b *testing.B) {
		for b.Loop() {
			tree.SetTree()
		}
	})

	b.Run("ToggleCollapsed", func(b *testing.B) {
		for b.Loop() {
			// rebuilds the flattened cache twice
			tree.ToggleCollapsed("./dir1")
			tree.ToggleCollapsed("./dir1")
		}
	})
}

func TestRepoFileTreeViewModelSelectionFollowsMutations(t *testing.T) {
	files := []*models.TreeFile{
		{Path: "dir1/file1", Mode: 0o100644},
		{Path: "dir1/file2", Mode: 0o100644},
		{Path: "dir2/file3", Mode: 0o100644},
	}
	viewModel := NewRepoFileTreeViewModel(
		func() []*models.TreeFile { return files },
		common.NewDummyCommon(),
		true,
	)
	viewModel.SetTree()

	viewModel.SelectPath("dir2/file3", true)
	assert.Equal(t, "dir2/file3", viewModel.GetSelectedPath())

	// toggling to flat mode keeps the same file selected
	viewModel.ToggleShowTree()
	assert.Equal(t, "dir2/file3", viewModel.GetSelectedPath())

	viewModel.ToggleShowTree()
	assert.Equal(t, "dir2/file3", viewModel.GetSelectedPath())

	// a filter that excludes the selection clamps it back into range
	viewModel.SetFilter("file1", false)
	assert.NotNil(t, viewModel.GetSelected())
	assert.Less(t, viewModel.GetSelectedLineIdx(), viewModel.Len())

	viewModel.ClearFilter()
	assert.NotNil(t, viewModel.GetSelected())
}
