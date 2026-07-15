package filetree

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

type IRepoFileTree interface {
	ITree[models.TreeFile]

	Get(index int) *RepoFileNode
	GetFile(path string) *models.TreeFile
	GetAllItems() []*RepoFileNode
	GetAllFiles() []*models.TreeFile
	GetRoot() *RepoFileNode
	SetTextFilter(filter string, useFuzzySearch bool)
	GetTextFilter() string
	CollapseAllToTopLevel()
	GetFlattenedRange(startIdx int, endIdx int) []FlattenedRepoFileNode
}

// A visible node of the repo file tree, together with the depth information
// needed to render it.
type FlattenedRepoFileNode struct {
	Node *Node[models.TreeFile]
	// depth in the uncompressed tree; used for truncating the displayed path
	TreeDepth int
	// depth on screen; used for indentation
	VisualDepth int
}

// RepoFileTree is a tree of all files in the repository at a given commit.
//
// Unlike the other trees in this package, which operate at diff scale and
// re-walk the node tree on every access, this one caches the flattened list
// of visible nodes: at full-repository scale (hundreds of thousands of files)
// per-interaction walks are too slow. Every method that changes what's
// visible must call refreshFlattened.
type RepoFileTree struct {
	getFiles       func() []*models.TreeFile
	tree           *Node[models.TreeFile]
	showTree       bool
	common         *common.Common
	collapsedPaths *CollapsedPaths
	textFilter     string
	useFuzzySearch bool

	flattened []FlattenedRepoFileNode
}

var _ IRepoFileTree = &RepoFileTree{}

func NewRepoFileTree(getFiles func() []*models.TreeFile, common *common.Common, showTree bool) *RepoFileTree {
	tree := &RepoFileTree{
		getFiles:       getFiles,
		common:         common,
		showTree:       showTree,
		collapsedPaths: NewCollapsedPaths(),
	}
	tree.SetTree()
	return tree
}

func (self *RepoFileTree) InTreeMode() bool {
	return self.showTree
}

func (self *RepoFileTree) ExpandToPath(path string) {
	self.collapsedPaths.ExpandToPath(path)
	self.refreshFlattened()
}

func (self *RepoFileTree) ToggleShowTree() {
	self.showTree = !self.showTree
	self.SetTree()
}

func (self *RepoFileTree) Get(index int) *RepoFileNode {
	if index < 0 || index >= len(self.flattened) {
		return nil
	}

	return NewRepoFileNode(self.flattened[index].Node)
}

// A linear scan: this is only called for occasional operations (re-finding
// the selection after a mutation), so scanning is cheaper overall than
// maintaining a path->index map on every rebuild of the flattened cache.
func (self *RepoFileTree) GetIndexForPath(path string) (int, bool) {
	for i, entry := range self.flattened {
		if entry.Node.GetInternalPath() == path {
			return i, true
		}
	}
	return -1, false
}

func (self *RepoFileTree) GetAllItems() []*RepoFileNode {
	return lo.Map(self.flattened, func(entry FlattenedRepoFileNode, _ int) *RepoFileNode {
		return NewRepoFileNode(entry.Node)
	})
}

func (self *RepoFileTree) Len() int {
	return len(self.flattened)
}

func (self *RepoFileTree) GetItem(index int) types.HasUrn {
	// Unimplemented because we don't need to show inline statuses in repo file views
	return nil
}

func (self *RepoFileTree) GetAllFiles() []*models.TreeFile {
	return self.getFiles()
}

func (self *RepoFileTree) getFilesForDisplay() []*models.TreeFile {
	files := self.getFiles()
	if self.textFilter != "" {
		files = filterTreeFilesByText(files, self.textFilter, self.useFuzzySearch)
	}
	return files
}

func (self *RepoFileTree) SetTree() {
	filesForDisplay := self.getFilesForDisplay()
	guiConfig := self.common.UserConfig().Gui
	showRootItem := guiConfig.ShowRootItemInFileTree
	cmp := NodeSortComparator[models.TreeFile](guiConfig.FileTreeSortOrder, guiConfig.FileTreeSortCaseSensitive)
	if self.showTree {
		self.tree = BuildTreeFromTreeFiles(filesForDisplay, showRootItem, cmp)
	} else {
		self.tree = BuildFlatTreeFromTreeFiles(filesForDisplay, showRootItem, cmp)
	}
	self.refreshFlattened()
}

func (self *RepoFileTree) SetTextFilter(filter string, useFuzzySearch bool) {
	self.textFilter = filter
	self.useFuzzySearch = useFuzzySearch
	self.SetTree()
}

func (self *RepoFileTree) GetTextFilter() string {
	return self.textFilter
}

func (self *RepoFileTree) IsCollapsed(path string) bool {
	return self.collapsedPaths.IsCollapsed(path)
}

func (self *RepoFileTree) ToggleCollapsed(path string) {
	self.collapsedPaths.ToggleCollapsed(path)
	self.refreshFlattened()
}

func (self *RepoFileTree) CollapseAll() {
	collapseAllAux(self.tree, self.collapsedPaths, false)
	self.refreshFlattened()
}

func (self *RepoFileTree) ExpandAll() {
	self.collapsedPaths.ExpandAll()
	self.refreshFlattened()
}

// Collapses every directory except the root item, so that only the top level
// of the repository is visible, like in a file explorer that has just been
// opened.
func (self *RepoFileTree) CollapseAllToTopLevel() {
	for _, child := range self.tree.Children {
		if child.GetInternalPath() == "." {
			// this is the root item; keep it expanded so that the top-level
			// entries below it show
			for _, grandchild := range child.Children {
				collapseAllAux(grandchild, self.collapsedPaths, true)
			}
		} else {
			collapseAllAux(child, self.collapsedPaths, true)
		}
	}
	self.refreshFlattened()
}

// Collapses the given node (if includeNode is set) and every directory below
// it, whether currently visible or not.
func collapseAllAux[T any](node *Node[T], collapsedPaths *CollapsedPaths, includeNode bool) {
	if node.IsFile() {
		return
	}
	if includeNode {
		collapsedPaths.Collapse(node.GetInternalPath())
	}
	for _, child := range node.Children {
		collapseAllAux(child, collapsedPaths, true)
	}
}

func (self *RepoFileTree) GetRoot() *RepoFileNode {
	return NewRepoFileNode(self.tree)
}

func (self *RepoFileTree) CollapsedPaths() *CollapsedPaths {
	return self.collapsedPaths
}

func (self *RepoFileTree) GetFile(path string) *models.TreeFile {
	for _, file := range self.getFiles() {
		if file.Path == path {
			return file
		}
	}

	return nil
}

func (self *RepoFileTree) GetVisualDepth(index int) int {
	if index < 0 || index >= len(self.flattened) {
		return -1
	}

	return self.flattened[index].VisualDepth
}

func (self *RepoFileTree) GetFlattenedRange(startIdx int, endIdx int) []FlattenedRepoFileNode {
	startIdx = max(startIdx, 0)
	endIdx = min(endIdx, len(self.flattened))
	if startIdx >= endIdx {
		return nil
	}

	return self.flattened[startIdx:endIdx]
}

func (self *RepoFileTree) refreshFlattened() {
	self.flattened = self.flattened[:0]
	self.flattenAux(self.tree, -1, -1)
}

// The depth accounting mirrors renderAux in the presentation package: the
// root is invisible (treeDepth -1), and a compressed node (e.g. "a/b/")
// advances the tree depth by more than one while advancing the visual depth
// by exactly one.
func (self *RepoFileTree) flattenAux(node *Node[models.TreeFile], treeDepth int, visualDepth int) {
	if node == nil {
		return
	}

	isRoot := treeDepth == -1
	if !isRoot {
		self.flattened = append(self.flattened, FlattenedRepoFileNode{
			Node:        node,
			TreeDepth:   treeDepth,
			VisualDepth: visualDepth,
		})
	}

	if node.IsFile() {
		return
	}

	if !isRoot && self.collapsedPaths.IsCollapsed(node.GetInternalPath()) {
		return
	}

	for _, child := range node.Children {
		self.flattenAux(child, treeDepth+1+node.CompressionLevel, visualDepth+1)
	}
}
