package filetree

import "github.com/jesseduffield/lazygit/pkg/commands/models"

// RepoFileNode wraps a node and provides some repo-file-specific methods for it.
type RepoFileNode struct {
	*Node[models.TreeFile]
}

func NewRepoFileNode(node *Node[models.TreeFile]) *RepoFileNode {
	if node == nil {
		return nil
	}

	return &RepoFileNode{Node: node}
}

// returns the underlying node, without any repo-file-specific methods attached
func (self *RepoFileNode) Raw() *Node[models.TreeFile] {
	if self == nil {
		return nil
	}

	return self.Node
}
