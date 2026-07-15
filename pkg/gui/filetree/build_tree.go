package filetree

import (
	"sort"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
)

func buildTree[T any](
	files []*T,
	getPath func(*T) string,
	showRootItem bool,
	cmp func(a, b *Node[T]) int,
) *Node[T] {
	root := &Node[T]{}

	childrenMapsByNode := make(map[*Node[T]]map[string]*Node[T])

	var curr *Node[T]
	for _, file := range files {
		splitPath := SplitFileTreePath(getPath(file), showRootItem)
		curr = root
	outer:
		for i := range splitPath {
			var setFile *T
			isFile := i == len(splitPath)-1
			if isFile {
				setFile = file
			}

			path := join(splitPath[:i+1])

			var currNodeChildrenMap map[string]*Node[T]
			var isCurrNodeMapped bool

			if currNodeChildrenMap, isCurrNodeMapped = childrenMapsByNode[curr]; !isCurrNodeMapped {
				currNodeChildrenMap = make(map[string]*Node[T])
				childrenMapsByNode[curr] = currNodeChildrenMap
			}

			child, doesCurrNodeHaveChildAlready := currNodeChildrenMap[path]
			if doesCurrNodeHaveChildAlready {
				curr = child
				continue outer
			}

			if i == 0 && len(files) == 1 && len(splitPath) == 2 {
				// skip the root item when there's only one file at top level; we don't need it in that case
				continue outer
			}

			newChild := &Node[T]{
				path: path,
				File: setFile,
			}
			curr.Children = append(curr.Children, newChild)

			currNodeChildrenMap[path] = newChild

			curr = newChild
		}
	}

	root.Sort(cmp)
	root.Compress()

	return root
}

func BuildTreeFromFiles(
	files []*models.File,
	showRootItem bool,
	cmp func(a, b *Node[models.File]) int,
) *Node[models.File] {
	return buildTree(files, (*models.File).GetPath, showRootItem, cmp)
}

func BuildFlatTreeFromCommitFiles(
	files []*models.CommitFile,
	showRootItem bool,
	cmp func(a, b *Node[models.CommitFile]) int,
) *Node[models.CommitFile] {
	rootAux := BuildTreeFromCommitFiles(files, showRootItem, cmp)
	sortedFiles := rootAux.GetLeaves()

	return &Node[models.CommitFile]{Children: sortedFiles}
}

func BuildTreeFromCommitFiles(
	files []*models.CommitFile,
	showRootItem bool,
	cmp func(a, b *Node[models.CommitFile]) int,
) *Node[models.CommitFile] {
	return buildTree(files, (*models.CommitFile).GetPath, showRootItem, cmp)
}

func BuildFlatTreeFromFiles(
	files []*models.File,
	showRootItem bool,
	cmp func(a, b *Node[models.File]) int,
) *Node[models.File] {
	rootAux := BuildTreeFromFiles(files, showRootItem, cmp)
	sortedFiles := rootAux.GetLeaves()

	// from top down we have merge conflict files, then tracked file, then untracked
	// files. This is the one way in which sorting differs between flat mode and
	// tree mode
	sort.SliceStable(sortedFiles, func(i, j int) bool {
		iFile := sortedFiles[i].File
		jFile := sortedFiles[j].File

		// never going to happen but just to be safe
		if iFile == nil || jFile == nil {
			return false
		}

		if iFile.HasMergeConflicts && !jFile.HasMergeConflicts {
			return true
		}

		if jFile.HasMergeConflicts && !iFile.HasMergeConflicts {
			return false
		}

		if iFile.Tracked && !jFile.Tracked {
			return true
		}

		if jFile.Tracked && !iFile.Tracked {
			return false
		}

		return false
	})

	return &Node[models.File]{Children: sortedFiles}
}

func split(str string) []string {
	return strings.Split(str, "/")
}

func join(strs []string) string {
	return strings.Join(strs, "/")
}

func SplitFileTreePath(path string, showRootItem bool) []string {
	return split(InternalTreePathForFilePath(path, showRootItem))
}

func InternalTreePathForFilePath(path string, showRootItem bool) string {
	if showRootItem {
		return "./" + path
	}

	return path
}
