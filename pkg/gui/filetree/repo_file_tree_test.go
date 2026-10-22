package filetree

import (
	"fmt"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/i18n"
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

func newTestRepoFileTreeWithConfig(paths []string, showTree bool, configure func(*config.UserConfig)) *RepoFileTree {
	files := lo.Map(paths, func(path string, _ int) *models.TreeFile {
		return &models.TreeFile{Path: path, Mode: 0o100644}
	})
	userConfig := config.GetDefaultConfig()
	configure(userConfig)
	return NewRepoFileTree(
		func() []*models.TreeFile { return files },
		common.NewDummyCommonWithUserConfigAndAppState(userConfig, nil),
		showTree,
	)
}

func newTestRepoFileTreeViewModel(paths []string) *RepoFileTreeViewModel {
	files := lo.Map(paths, func(path string, _ int) *models.TreeFile {
		return &models.TreeFile{Path: path, Mode: 0o100644}
	})
	viewModel := NewRepoFileTreeViewModel(
		func() []*models.TreeFile { return files },
		common.NewDummyCommon(),
		true,
	)
	viewModel.SetTree()
	return viewModel
}

func TestRepoFileTreeExpandToPath(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir1/file1",
		"dir1/sub/file2",
		"dir2/file3",
	})
	tree.CollapseAllToTopLevel()

	tree.ExpandToPath("./dir1/sub/file2")
	assert.Equal(t, []string{
		".",
		"./dir1",
		"./dir1/file1",
		"./dir1/sub",
		"./dir1/sub/file2",
		"./dir2",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeGetFile(t *testing.T) {
	tree := newTestRepoFileTree([]string{"dir/file1", "file2"})

	file := tree.GetFile("dir/file1")
	assert.NotNil(t, file)
	assert.Equal(t, "dir/file1", file.Path)

	assert.Nil(t, tree.GetFile("dir"))
	assert.Nil(t, tree.GetFile("missing"))
	assert.Len(t, tree.GetAllFiles(), 2)
}

func TestRepoFileTreeOutOfRangeAccess(t *testing.T) {
	tree := newTestRepoFileTree([]string{"dir/file1", "file2"})

	assert.Nil(t, tree.Get(-1))
	assert.Nil(t, tree.Get(tree.Len()))
	assert.Equal(t, -1, tree.GetVisualDepth(-1))
	assert.Equal(t, -1, tree.GetVisualDepth(tree.Len()))
	assert.Nil(t, tree.GetItem(0))
}

func TestRepoFileTreeEmpty(t *testing.T) {
	tree := newTestRepoFileTree([]string{})

	assert.Equal(t, 0, tree.Len())
	assert.Empty(t, tree.GetAllItems())
	assert.Empty(t, tree.GetFlattenedRange(0, 10))

	// none of these should blow up on an empty tree
	tree.CollapseAllToTopLevel()
	tree.CollapseAll()
	tree.ExpandAll()
	tree.ToggleShowTree()
	assert.Equal(t, 0, tree.Len())
}

func TestRepoFileTreeWithoutRootItem(t *testing.T) {
	tree := newTestRepoFileTreeWithConfig([]string{
		"dir1/file1",
		"dir1/sub/file2",
		"file3",
	}, true, func(userConfig *config.UserConfig) {
		userConfig.Gui.ShowRootItemInFileTree = false
	})

	assert.Equal(t, []string{
		"dir1",
		"dir1/file1",
		"dir1/sub",
		"dir1/sub/file2",
		"file3",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	tree.CollapseAllToTopLevel()
	assert.Equal(t, []string{
		"dir1",
		"file3",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)
	assert.Equal(t, 0, tree.GetVisualDepth(0))
}

func TestRepoFileTreeFlatMode(t *testing.T) {
	tree := newTestRepoFileTreeWithConfig([]string{
		"b/file2",
		"a/sub/file1",
		"file3",
	}, false, func(*config.UserConfig) {})

	assert.False(t, tree.InTreeMode())
	assert.Equal(t, []string{
		"./a/sub/file1",
		"./b/file2",
		"./file3",
	}, visiblePaths(tree))
	for i := range tree.Len() {
		assert.Equal(t, 0, tree.GetVisualDepth(i))
		assert.True(t, tree.Get(i).IsFile())
	}
	assertFlattenedCacheConsistent(t, tree)

	// collapsing is meaningless without directories
	tree.CollapseAllToTopLevel()
	assert.Equal(t, 3, tree.Len())
}

func TestRepoFileTreeSortOrder(t *testing.T) {
	paths := []string{"b.txt", "a/file", "C/file", "c.txt"}

	scenarios := []struct {
		sortOrder     string
		caseSensitive bool
		expected      []string
	}{
		{
			sortOrder: "mixed",
			expected:  []string{".", "./a", "./a/file", "./b.txt", "./C", "./C/file", "./c.txt"},
		},
		{
			sortOrder: "foldersFirst",
			expected:  []string{".", "./a", "./a/file", "./C", "./C/file", "./b.txt", "./c.txt"},
		},
		{
			sortOrder: "filesFirst",
			expected:  []string{".", "./b.txt", "./c.txt", "./a", "./a/file", "./C", "./C/file"},
		},
		{
			sortOrder:     "mixed",
			caseSensitive: true,
			expected:      []string{".", "./C", "./C/file", "./a", "./a/file", "./b.txt", "./c.txt"},
		},
	}

	for _, s := range scenarios {
		t.Run(fmt.Sprintf("%s caseSensitive=%v", s.sortOrder, s.caseSensitive), func(t *testing.T) {
			tree := newTestRepoFileTreeWithConfig(paths, true, func(userConfig *config.UserConfig) {
				userConfig.Gui.FileTreeSortOrder = s.sortOrder
				userConfig.Gui.FileTreeSortCaseSensitive = s.caseSensitive
			})
			assert.Equal(t, s.expected, visiblePaths(tree))
		})
	}
}

func TestRepoFileTreeUnicodeAndSpaces(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir with spaces/file name.txt",
		"日本語/ファイル.txt",
		"dïr/ünïcode.txt",
	})

	index, found := tree.GetIndexForPath("./dir with spaces/file name.txt")
	assert.True(t, found)
	node := tree.Get(index)
	assert.Equal(t, "dir with spaces/file name.txt", node.GetPath())
	assert.Equal(t, "file name.txt", node.Name())

	index, found = tree.GetIndexForPath("./日本語/ファイル.txt")
	assert.True(t, found)
	assert.Equal(t, "ファイル.txt", tree.Get(index).Name())

	_, found = tree.GetIndexForPath("./dïr/ünïcode.txt")
	assert.True(t, found)
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeFuzzyTextFilter(t *testing.T) {
	tree := newTestRepoFileTree([]string{
		"dir1/apple",
		"dir1/banana",
		"dir2/cherry",
	})

	tree.SetTextFilter("d1apl", true)
	assert.Equal(t, "d1apl", tree.GetTextFilter())
	// the root item and the only remaining directory are compressed into one
	assert.Equal(t, []string{
		"./dir1",
		"./dir1/apple",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)

	// the same filter without fuzzy matching matches nothing
	tree.SetTextFilter("d1apl", false)
	assert.Equal(t, 0, tree.Len())

	// the full file set is unaffected by filtering
	assert.Len(t, tree.GetAllFiles(), 3)
}

func TestRepoFileTreeSetTreePicksUpNewFiles(t *testing.T) {
	files := []*models.TreeFile{
		{Path: "dir/file1", Mode: 0o100644},
		{Path: "other/file2", Mode: 0o100644},
	}
	tree := NewRepoFileTree(
		func() []*models.TreeFile { return files },
		common.NewDummyCommon(),
		true,
	)
	tree.ToggleCollapsed("./dir")

	files = []*models.TreeFile{
		{Path: "dir/file1", Mode: 0o100644},
		{Path: "dir/file3", Mode: 0o100644},
		{Path: "new/file4", Mode: 0o100644},
	}
	tree.SetTree()

	// the collapsed state is kept across the rebuild
	assert.Equal(t, []string{
		".",
		"./dir",
		"./new",
		"./new/file4",
	}, visiblePaths(tree))
	assertFlattenedCacheConsistent(t, tree)
}

func TestRepoFileTreeSpecialEntriesAreLeaves(t *testing.T) {
	files := []*models.TreeFile{
		{Path: "link", Mode: 0o120000},
		{Path: "sub/module", Mode: 0o160000},
	}
	tree := NewRepoFileTree(
		func() []*models.TreeFile { return files },
		common.NewDummyCommon(),
		true,
	)

	index, found := tree.GetIndexForPath("./link")
	assert.True(t, found)
	assert.True(t, tree.Get(index).IsFile())
	assert.True(t, tree.Get(index).File.IsSymlink())

	index, found = tree.GetIndexForPath("./sub/module")
	assert.True(t, found)
	assert.True(t, tree.Get(index).IsFile())
	assert.True(t, tree.Get(index).File.IsSubmodule())
}

func TestRepoFileNodeRaw(t *testing.T) {
	assert.Nil(t, NewRepoFileNode(nil))

	var nilNode *RepoFileNode
	assert.Nil(t, nilNode.Raw())

	raw := &Node[models.TreeFile]{path: "file"}
	assert.Same(t, raw, NewRepoFileNode(raw).Raw())
}

func TestRepoFileTreeViewModelEmpty(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{})

	assert.Nil(t, viewModel.GetSelected())
	assert.Equal(t, "", viewModel.GetSelectedItemId())
	assert.Nil(t, viewModel.GetSelectedFile())
	assert.Equal(t, "", viewModel.GetSelectedPath())

	items, startIdx, endIdx := viewModel.GetSelectedItems()
	assert.Empty(t, items)
	assert.Equal(t, 0, startIdx)
	assert.Equal(t, 0, endIdx)

	// none of these should blow up without a selection
	viewModel.ToggleShowTree()
	viewModel.ToggleShowTree()
	viewModel.CollapseAll()
	viewModel.ExpandAll()
	viewModel.ClearFilter()
}

func TestRepoFileTreeViewModelSelectedItems(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{
		"dir1/file1",
		"dir1/file2",
		"file3",
	})

	viewModel.SelectPath("dir1/file1", true)
	assert.Equal(t, "dir1/file1", viewModel.GetSelectedItemId())
	assert.Equal(t, "dir1/file1", viewModel.GetSelectedFile().Path)

	viewModel.SetSelectedLineIdx(1)
	assert.Nil(t, viewModel.GetSelectedFile())
	assert.Equal(t, "dir1", viewModel.GetSelectedPath())

	viewModel.ToggleStickyRange()
	viewModel.SetSelectedLineIdx(3)

	ids, startIdx, endIdx := viewModel.GetSelectedItemIds()
	assert.Equal(t, []string{"dir1", "dir1/file1", "dir1/file2"}, ids)
	assert.Equal(t, 1, startIdx)
	assert.Equal(t, 3, endIdx)

	items, _, _ := viewModel.GetSelectedItems()
	assert.Len(t, items, 3)
	assert.Equal(t, "dir1/file2", items[2].GetPath())
}

func TestRepoFileTreeViewModelCollapseAllSelectsTopLevelAncestor(t *testing.T) {
	scenarios := []struct {
		showRootItem     bool
		expectedSelected string
	}{
		{showRootItem: true, expectedSelected: "."},
		{showRootItem: false, expectedSelected: "dir1"},
	}

	for _, s := range scenarios {
		t.Run(fmt.Sprintf("showRootItem=%v", s.showRootItem), func(t *testing.T) {
			files := []*models.TreeFile{
				{Path: "dir1/sub/file1", Mode: 0o100644},
				{Path: "dir1/other", Mode: 0o100644},
				{Path: "file2", Mode: 0o100644},
			}
			userConfig := config.GetDefaultConfig()
			userConfig.Gui.ShowRootItemInFileTree = s.showRootItem
			viewModel := NewRepoFileTreeViewModel(
				func() []*models.TreeFile { return files },
				common.NewDummyCommonWithUserConfigAndAppState(userConfig, nil),
				true,
			)
			viewModel.SetTree()
			viewModel.SelectPath("dir1/sub/file1", s.showRootItem)
			assert.Equal(t, "dir1/sub/file1", viewModel.GetSelectedPath())

			viewModel.CollapseAll()
			assert.Equal(t, s.expectedSelected, viewModel.GetSelected().GetInternalPath())

			viewModel.ExpandAll()
			assert.Equal(t, s.expectedSelected, viewModel.GetSelected().GetInternalPath())
		})
	}
}

func TestRepoFileTreeViewModelExpandAllKeepsSelection(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{
		"dir1/file1",
		"dir2/file2",
		"file3",
	})
	viewModel.CollapseAllToTopLevel()
	viewModel.SelectPath("file3", true)

	viewModel.ExpandAll()
	assert.Equal(t, "file3", viewModel.GetSelectedPath())
	assert.Equal(t, 5, viewModel.GetSelectedLineIdx())
}

func TestRepoFileTreeViewModelToggleShowTreeWithDirectorySelected(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{
		"dir1/b",
		"dir1/a",
		"file3",
	})
	index, found := viewModel.GetIndexForPath("./dir1")
	assert.True(t, found)
	viewModel.SetSelection(index)

	// a directory doesn't exist in flat mode, so its first file is selected
	viewModel.ToggleShowTree()
	assert.False(t, viewModel.InTreeMode())
	assert.Equal(t, "dir1/a", viewModel.GetSelectedPath())

	// going back to tree mode expands the parents of the selected file
	viewModel.ToggleShowTree()
	assert.True(t, viewModel.InTreeMode())
	assert.Equal(t, "dir1/a", viewModel.GetSelectedPath())
}

func TestRepoFileTreeViewModelToggleShowTreeExpandsToSelection(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{
		"dir1/sub/file1",
		"dir2/file2",
	})
	viewModel.CollapseAllToTopLevel()
	viewModel.ToggleShowTree()
	viewModel.SelectPath("dir1/sub/file1", true)

	viewModel.ToggleShowTree()
	assert.Equal(t, "dir1/sub/file1", viewModel.GetSelectedPath())
	assert.False(t, viewModel.IsCollapsed("./dir1"))
	assert.False(t, viewModel.IsCollapsed("./dir1/sub"))
	assert.True(t, viewModel.IsCollapsed("./dir2"))
}

func TestRepoFileTreeViewModelFilter(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{
		"dir1/apple",
		"dir1/banana",
		"dir2/cherry",
	})
	assert.False(t, viewModel.IsFiltering())

	viewModel.SelectPath("dir2/cherry", true)
	assert.Equal(t, 5, viewModel.GetSelectedLineIdx())

	viewModel.SetFilter("banana", false)
	assert.True(t, viewModel.IsFiltering())
	assert.Equal(t, "banana", viewModel.GetFilter())
	assert.Equal(t, 2, viewModel.Len())
	assert.Equal(t, 1, viewModel.GetSelectedLineIdx())
	assert.Equal(t, "dir1/banana", viewModel.GetSelectedPath())

	viewModel.ReApplyFilter(false)
	assert.Equal(t, "banana", viewModel.GetFilter())
	assert.Equal(t, 2, viewModel.Len())

	// clearing the filter keeps the selected file selected, even if it was
	// inside a collapsed directory
	viewModel.ToggleCollapsed("./dir1/banana")
	viewModel.ClearFilter()
	assert.False(t, viewModel.IsFiltering())
	assert.Equal(t, "dir1/banana", viewModel.GetSelectedPath())
	assert.Equal(t, 6, viewModel.Len())
}

func TestRepoFileTreeViewModelClearFilterExpandsCollapsedParents(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{
		"dir1/apple",
		"dir2/sub/cherry",
		"dir2/other",
	})
	viewModel.CollapseAllToTopLevel()

	// while filtering, the root item, dir2 and sub are compressed into a
	// single node, so expanding it doesn't expand dir2 itself
	viewModel.SetFilter("cherry", false)
	viewModel.ToggleCollapsed("./dir2/sub")
	viewModel.SelectPath("dir2/sub/cherry", true)
	assert.Equal(t, "dir2/sub/cherry", viewModel.GetSelectedPath())
	assert.True(t, viewModel.IsCollapsed("./dir2"))

	viewModel.ClearFilter()
	assert.Equal(t, "dir2/sub/cherry", viewModel.GetSelectedPath())
	assert.False(t, viewModel.IsCollapsed("./dir2"))
	assert.False(t, viewModel.IsCollapsed("./dir2/sub"))
	assert.True(t, viewModel.IsCollapsed("./dir1"))
}

func TestRepoFileTreeViewModelMisc(t *testing.T) {
	viewModel := newTestRepoFileTreeViewModel([]string{"file1", "file2"})

	assert.Nil(t, viewModel.GetRef())
	ref := &models.Tag{Name: "v1.0"}
	viewModel.SetRef(ref)
	assert.Equal(t, ref, viewModel.GetRef())

	assert.Equal(t, i18n.EnglishTranslationSet().FilterPrefix, viewModel.FilterPrefix(i18n.EnglishTranslationSet()))
	assert.NotNil(t, viewModel.GetSearchHistory())

	// SelectPath with a path that isn't there leaves the selection alone
	viewModel.SetSelection(1)
	viewModel.SelectPath("missing", true)
	assert.Equal(t, 1, viewModel.GetSelectedLineIdx())
}
