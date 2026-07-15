package controllers

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// This controller handles the panel that shows all files in the repository at
// a given commit.

type RepoFilesController struct {
	baseController
	*ListControllerTrait[*filetree.RepoFileNode]
	c *ControllerCommon
}

var _ types.IController = &RepoFilesController{}

func NewRepoFilesController(
	c *ControllerCommon,
) *RepoFilesController {
	return &RepoFilesController{
		baseController: baseController{},
		c:              c,
		ListControllerTrait: NewListControllerTrait(
			c,
			c.Contexts().RepoFiles,
			c.Contexts().RepoFiles.GetSelected,
			c.Contexts().RepoFiles.GetSelectedItems,
		),
	}
}

func (self *RepoFilesController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.GoInto),
			Handler:           self.withItem(self.enter),
			GetDisabledReason: self.require(self.singleItemSelected()),
			Description:       self.c.Tr.EnterCommitFile,
			Tooltip:           self.c.Tr.EnterRepoFileTooltip,
		},
		{
			Keys:              opts.GetKeys(opts.Config.RepoFiles.Restore),
			Handler:           self.withItems(self.restore),
			GetDisabledReason: self.require(self.itemsSelected()),
			Description:       self.c.Tr.RestoreToWorktree,
			Tooltip:           self.c.Tr.RestoreToWorktreeTooltip,
			DisplayOnScreen:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.RepoFiles.SaveFile),
			Handler:           self.withItem(self.openSaveMenu),
			GetDisabledReason: self.require(self.singleItemSelected(self.canSave)),
			Description:       self.c.Tr.SaveFilesFromCommit,
			Tooltip:           self.c.Tr.SaveFilesFromCommitTooltip,
			OpensMenu:         true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Files.ToggleTreeView),
			Handler:     self.toggleTreeView,
			Description: self.c.Tr.ToggleTreeView,
			Tooltip:     self.c.Tr.ToggleTreeViewTooltip,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Files.CollapseAll),
			Handler:           self.collapseAll,
			Description:       self.c.Tr.CollapseAll,
			Tooltip:           self.c.Tr.CollapseAllTooltip,
			GetDisabledReason: self.require(self.isInTreeMode),
		},
		{
			Keys:              opts.GetKeys(opts.Config.Files.ExpandAll),
			Handler:           self.expandAll,
			Description:       self.c.Tr.ExpandAll,
			Tooltip:           self.c.Tr.ExpandAllTooltip,
			GetDisabledReason: self.require(self.isInTreeMode),
		},
	}

	return bindings
}

func (self *RepoFilesController) context() *context.RepoFilesContext {
	return self.c.Contexts().RepoFiles
}

func (self *RepoFilesController) GetOnClick() func(opts gocui.ViewMouseBindingOpts) error {
	return func(opts gocui.ViewMouseBindingOpts) error {
		clickedIdx := self.context().GetSelectedLineIdx()
		node := self.context().RepoFileTreeViewModel.Get(clickedIdx)
		if node == nil || node.File != nil {
			return nil
		}

		// The arrow is at column visualDepth*2 (after indentation of 2 spaces per level).
		// Only treat clicks on the arrow and the trailing space as arrow clicks.
		visualDepth := self.context().RepoFileTreeViewModel.GetVisualDepth(clickedIdx)
		arrowStartCol := visualDepth * 2
		arrowEndCol := arrowStartCol + 1
		if opts.X < arrowStartCol || opts.X > arrowEndCol {
			return nil
		}

		self.context().RepoFileTreeViewModel.ToggleCollapsed(node.GetInternalPath())
		self.c.PostRefreshUpdate(self.context())

		return nil
	}
}

func (self *RepoFilesController) enter(node *filetree.RepoFileNode) error {
	if node.File == nil {
		return self.handleToggleDirCollapsed(node)
	}

	mainViewContext := self.c.Contexts().Normal
	mainViewContext.ClearSearchString()
	self.c.Context().Push(mainViewContext, types.OnFocusOpts{})
	return nil
}

func (self *RepoFilesController) restore(selectedNodes []*filetree.RepoFileNode) error {
	selectedNodes = filetree.NormaliseSelectedNodes(selectedNodes)

	paths := lo.Map(selectedNodes, func(node *filetree.RepoFileNode, _ int) string {
		return node.GetPath()
	})

	for _, path := range paths {
		if helpers.AnyTrackedFilesInPathExceptSubmodules(path, self.c.Model().Files, self.c.Model().Submodules) {
			return errors.New(self.c.Tr.CannotCheckoutWithModifiedFilesErr)
		}
	}

	hash := self.context().GetLoadedCommitHash()

	self.c.Confirm(types.ConfirmOpts{
		Title: self.c.Tr.RestoreToWorktree,
		Prompt: utils.ResolvePlaceholderString(self.c.Tr.RestoreToWorktreePrompt, map[string]string{
			"paths": fmt.Sprintf("'%s'", utils.FormatPaths(paths)),
			"ref":   utils.ShortHash(hash),
		}),
		HandleConfirm: func() error {
			self.c.LogAction(self.c.Tr.Actions.RestoreFilesToWorktree)
			if err := self.c.Git().WorkingTree.RestoreToWorktree(hash, paths); err != nil {
				return err
			}

			self.c.Refresh(types.RefreshOptions{Mode: types.ASYNC})
			return nil
		},
	})

	return nil
}

func (self *RepoFilesController) canSave(node *filetree.RepoFileNode) *types.DisabledReason {
	if node.File != nil && node.File.IsSubmodule() {
		return &types.DisabledReason{Text: self.c.Tr.ErrCannotSaveSubmodule}
	}

	return nil
}

func (self *RepoFilesController) openSaveMenu(node *filetree.RepoFileNode) error {
	saveFileAsItem := &types.MenuItem{
		Label: self.c.Tr.SaveFileAs,
		OnPress: func() error {
			self.c.Prompt(types.PromptOpts{
				Title:          self.c.Tr.SaveFileAsPromptTitle,
				InitialContent: node.GetPath(),
				HandleConfirm: func(path string) error {
					return self.saveNode(node, path)
				},
			})
			return nil
		},
		Keys: menuKey('s'),
	}
	saveToOriginalLocationItem := &types.MenuItem{
		Label: self.c.Tr.SaveToOriginalLocation,
		OnPress: func() error {
			return self.saveNode(node, node.GetPath())
		},
		Keys: menuKey('o'),
	}

	return self.c.Menu(types.CreateMenuOptions{
		Title: self.c.Tr.SaveFilesFromCommit,
		Items: []*types.MenuItem{
			saveFileAsItem,
			saveToOriginalLocationItem,
		},
	})
}

// Saves the node's file to the given path, or, for a directory node, every
// file below it into the given directory (submodule entries are skipped).
// Asks for confirmation if the target path already exists.
func (self *RepoFilesController) saveNode(node *filetree.RepoFileNode, targetPath string) error {
	type fileToSave struct {
		srcPath string
		dstPath string
	}

	var filesToSave []fileToSave
	if node.File != nil {
		filesToSave = []fileToSave{{srcPath: node.GetPath(), dstPath: targetPath}}
	} else {
		for _, leaf := range node.GetLeaves() {
			if leaf.File.IsSubmodule() {
				continue
			}
			filesToSave = append(filesToSave, fileToSave{
				srcPath: leaf.File.Path,
				dstPath: targetPath + strings.TrimPrefix(leaf.File.Path, node.GetPath()),
			})
		}
	}

	hash := self.context().GetLoadedCommitHash()

	targetExists := false
	if _, err := os.Stat(targetPath); err == nil {
		targetExists = true
	}

	return self.c.ConfirmIf(targetExists, types.ConfirmOpts{
		Title: self.c.Tr.SaveFilesFromCommit,
		Prompt: utils.ResolvePlaceholderString(self.c.Tr.SaveFileOverwritePrompt, map[string]string{
			"path": targetPath,
		}),
		HandleConfirm: func() error {
			self.c.LogAction(self.c.Tr.Actions.SaveFilesFromCommit)
			for _, file := range filesToSave {
				content, err := self.c.Git().Commit.GetFilteredFileContent(hash, file.srcPath)
				if err != nil {
					return err
				}
				if err := self.c.OS().CreateFileWithContent(file.dstPath, content); err != nil {
					return err
				}
			}

			self.c.Toast(utils.ResolvePlaceholderString(self.c.Tr.FileSavedToast, map[string]string{
				"path": targetPath,
			}))
			self.c.Refresh(types.RefreshOptions{Mode: types.ASYNC, Scope: []types.RefreshableView{types.FILES}})
			return nil
		},
	})
}

func (self *RepoFilesController) GetOnRenderToMain() func() {
	return func() {
		node := self.context().GetSelected()
		if node == nil {
			return
		}

		hash := self.context().GetLoadedCommitHash()
		path := node.GetPath()

		self.c.RenderToMainViews(types.RefreshMainOpts{
			Pair: self.c.MainViewPairs().Normal,
			Main: &types.ViewUpdateOpts{
				Title: path,
				Task:  self.mainViewTask(node, hash, path),
			},
		})
	}
}

func (self *RepoFilesController) mainViewTask(node *filetree.RepoFileNode, hash string, path string) types.UpdateTask {
	if node.File == nil {
		return self.dirListingTask(hash, path)
	}

	if node.File.IsSubmodule() {
		return types.NewRenderStringTask(utils.ResolvePlaceholderString(
			self.c.Tr.RepoFilesSubmodule,
			map[string]string{"hash": node.File.BlobHash},
		))
	}

	if node.File.IsSymlink() {
		// a symlink's blob content is its target path, which is tiny
		target, err := self.c.Git().Commit.ShowFileContentCmdObj(hash, path).RunWithOutput()
		if err != nil {
			return types.NewRenderStringTask(err.Error())
		}
		return types.NewRenderStringTask(utils.ResolvePlaceholderString(
			self.c.Tr.RepoFilesSymlink,
			map[string]string{"target": strings.TrimSuffix(target, "\n")},
		))
	}

	isBinary, err := self.c.Git().Commit.IsFileBinary(hash, path)
	if err != nil {
		return types.NewRenderStringTask(err.Error())
	}
	if isBinary {
		sizeStr := ""
		if size, err := self.c.Git().Commit.GetBlobSize(hash, path); err == nil {
			sizeStr = utils.FormatBytes(size)
		}
		return types.NewRenderStringTask(utils.ResolvePlaceholderString(
			self.c.Tr.RepoFilesBinaryFile,
			map[string]string{"size": sizeStr},
		))
	}

	cmdObj := self.c.Git().Commit.ShowFileContentCmdObj(hash, path)
	return types.NewRunPtyTask(cmdObj.GetCmd())
}

// shows the immediate entries of the directory (one level deep), like a file
// explorer would; showing the full contents of every file below the directory
// would be unbounded
func (self *RepoFilesController) dirListingTask(hash string, dir string) types.UpdateTask {
	entries, err := self.c.Git().Loaders.TreeFileLoader.GetTreeDirEntries(hash, dir)
	if err != nil {
		return types.NewRenderStringTask(err.Error())
	}

	lines := lo.Map(entries, func(entry *models.TreeFile, _ int) string {
		name := filepath.Base(entry.Path)
		if entry.IsDir() {
			return name + "/"
		}
		return name
	})

	return types.NewRenderStringTask(strings.Join(lines, "\n"))
}

func (self *RepoFilesController) handleToggleDirCollapsed(node *filetree.RepoFileNode) error {
	self.context().RepoFileTreeViewModel.ToggleCollapsed(node.GetInternalPath())

	self.c.PostRefreshUpdate(self.context())

	return nil
}

func (self *RepoFilesController) toggleTreeView() error {
	self.context().RepoFileTreeViewModel.ToggleShowTree()

	self.c.PostRefreshUpdate(self.context())
	return nil
}

func (self *RepoFilesController) collapseAll() error {
	self.context().RepoFileTreeViewModel.CollapseAll()

	self.c.PostRefreshUpdate(self.context())

	return nil
}

func (self *RepoFilesController) expandAll() error {
	self.context().RepoFileTreeViewModel.ExpandAll()

	self.c.PostRefreshUpdate(self.context())

	return nil
}

func (self *RepoFilesController) isInTreeMode() *types.DisabledReason {
	if !self.context().RepoFileTreeViewModel.InTreeMode() {
		return &types.DisabledReason{Text: self.c.Tr.DisabledInFlatView}
	}

	return nil
}
