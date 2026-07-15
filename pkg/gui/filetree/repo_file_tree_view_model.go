package filetree

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/context/traits"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type IRepoFileTreeViewModel interface {
	IRepoFileTree
	types.IListCursor

	GetRef() models.Ref
	SetRef(models.Ref)
}

// RepoFileTreeViewModel is the view model for the panel that shows all files
// in the repository at a given commit.
type RepoFileTreeViewModel struct {
	types.IListCursor
	IRepoFileTree

	// the ref (commit, branch, tag, stash entry) whose tree we're viewing
	ref models.Ref

	searchHistory *utils.HistoryBuffer[string]
}

var _ IRepoFileTreeViewModel = &RepoFileTreeViewModel{}

func NewRepoFileTreeViewModel(getFiles func() []*models.TreeFile, common *common.Common, showTree bool) *RepoFileTreeViewModel {
	fileTree := NewRepoFileTree(getFiles, common, showTree)
	listCursor := traits.NewListCursor(fileTree.Len)
	return &RepoFileTreeViewModel{
		IRepoFileTree: fileTree,
		IListCursor:   listCursor,
		ref:           nil,
		searchHistory: utils.NewHistoryBuffer[string](1000),
	}
}

func (self *RepoFileTreeViewModel) GetRef() models.Ref {
	return self.ref
}

func (self *RepoFileTreeViewModel) SetRef(ref models.Ref) {
	self.ref = ref
}

func (self *RepoFileTreeViewModel) GetSelected() *RepoFileNode {
	if self.Len() == 0 {
		return nil
	}

	return self.Get(self.GetSelectedLineIdx())
}

func (self *RepoFileTreeViewModel) GetSelectedItemId() string {
	item := self.GetSelected()
	if item == nil {
		return ""
	}

	return item.ID()
}

func (self *RepoFileTreeViewModel) GetSelectedItems() ([]*RepoFileNode, int, int) {
	if self.Len() == 0 {
		return nil, 0, 0
	}

	startIdx, endIdx := self.GetSelectionRange()

	nodes := []*RepoFileNode{}
	for i := startIdx; i <= endIdx; i++ {
		nodes = append(nodes, self.Get(i))
	}

	return nodes, startIdx, endIdx
}

func (self *RepoFileTreeViewModel) GetSelectedItemIds() ([]string, int, int) {
	selectedItems, startIdx, endIdx := self.GetSelectedItems()

	ids := lo.Map(selectedItems, func(item *RepoFileNode, _ int) string {
		return item.ID()
	})

	return ids, startIdx, endIdx
}

func (self *RepoFileTreeViewModel) GetSelectedFile() *models.TreeFile {
	node := self.GetSelected()
	if node == nil {
		return nil
	}

	return node.File
}

func (self *RepoFileTreeViewModel) GetSelectedPath() string {
	node := self.GetSelected()
	if node == nil {
		return ""
	}

	return node.GetPath()
}

// SetTree rebuilds the tree and clamps the selection so it stays in range
// after a shrinking rebuild (e.g. when a filter is applied).
func (self *RepoFileTreeViewModel) SetTree() {
	self.IRepoFileTree.SetTree()
	self.ClampSelection()
}

// duplicated from commit_file_tree_view_model.go. Generics will help here
func (self *RepoFileTreeViewModel) ToggleShowTree() {
	selectedNode := self.GetSelected()

	self.IRepoFileTree.ToggleShowTree()

	if selectedNode == nil {
		return
	}
	path := selectedNode.GetInternalPath()

	if self.InTreeMode() {
		self.ExpandToPath(path)
	} else if len(selectedNode.Children) > 0 {
		path = selectedNode.GetLeaves()[0].GetInternalPath()
	}

	index, found := self.GetIndexForPath(path)
	if found {
		self.SetSelection(index)
	}
}

func (self *RepoFileTreeViewModel) CollapseAll() {
	selectedNode := self.GetSelected()

	self.IRepoFileTree.CollapseAll()
	if selectedNode == nil {
		return
	}

	topLevelPath := strings.Split(selectedNode.GetInternalPath(), "/")[0]
	index, found := self.GetIndexForPath(topLevelPath)
	if found {
		self.SetSelectedLineIdx(index)
	}
}

func (self *RepoFileTreeViewModel) ExpandAll() {
	selectedNode := self.GetSelected()

	self.IRepoFileTree.ExpandAll()

	if selectedNode == nil {
		return
	}

	index, found := self.GetIndexForPath(selectedNode.GetInternalPath())
	if found {
		self.SetSelectedLineIdx(index)
	}
}

// Try to select the given path if present. If it doesn't exist, or one of the parent directories is
// collapsed, do nothing.
// Note that filepath is an actual file path, not an internal tree path as with e.g.
// ToggleCollapsed. It must be a relative path (relative to the repo root), and it must contain
// forward slashes rather than backslashes even on Windows.
func (self *RepoFileTreeViewModel) SelectPath(filepath string, showRootItem bool) {
	index, found := self.GetIndexForPath(InternalTreePathForFilePath(filepath, showRootItem))
	if found {
		self.SetSelection(index)
	}
}

// IFilterableContext methods

func (self *RepoFileTreeViewModel) SetFilter(filter string, useFuzzySearch bool) {
	self.IRepoFileTree.SetTextFilter(filter, useFuzzySearch)
	self.ClampSelection()
}

func (self *RepoFileTreeViewModel) GetFilter() string {
	return self.IRepoFileTree.GetTextFilter()
}

func (self *RepoFileTreeViewModel) ClearFilter() {
	selectedNode := self.GetSelected()
	var selectedPath string
	if selectedNode != nil {
		selectedPath = selectedNode.GetInternalPath()
	}

	self.IRepoFileTree.SetTextFilter("", false)

	if selectedPath != "" {
		self.ExpandToPath(selectedPath)
		if idx, found := self.GetIndexForPath(selectedPath); found {
			self.SetSelection(idx)
			return
		}
	}
	self.ClampSelection()
}

func (self *RepoFileTreeViewModel) ReApplyFilter(useFuzzySearch bool) {
	self.IRepoFileTree.SetTextFilter(self.IRepoFileTree.GetTextFilter(), useFuzzySearch)
	self.ClampSelection()
}

func (self *RepoFileTreeViewModel) IsFiltering() bool {
	return self.IRepoFileTree.GetTextFilter() != ""
}

// used for type switch
func (self *RepoFileTreeViewModel) IsFilterableContext() {}

func (self *RepoFileTreeViewModel) FilterPrefix(tr *i18n.TranslationSet) string {
	return tr.FilterPrefix
}

func (self *RepoFileTreeViewModel) GetSearchHistory() *utils.HistoryBuffer[string] {
	return self.searchHistory
}
