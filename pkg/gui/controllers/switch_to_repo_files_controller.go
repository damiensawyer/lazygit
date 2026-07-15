package controllers

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// This controller lets you open a panel showing all files in the repository
// as they existed at the selected ref. It's attached to all the ref-bearing
// list panels (commits, sub-commits, reflog, branches, remote branches, tags,
// stash).

var _ types.IController = &SwitchToRepoFilesController{}

type CanSwitchToRepoFiles interface {
	types.IListContext
	GetSelectedRef() models.Ref
}

type SwitchToRepoFilesController struct {
	baseController
	c       *ControllerCommon
	context CanSwitchToRepoFiles
}

func NewSwitchToRepoFilesController(
	c *ControllerCommon,
	context CanSwitchToRepoFiles,
) *SwitchToRepoFilesController {
	return &SwitchToRepoFilesController{
		baseController: baseController{},
		c:              c,
		context:        context,
	}
}

func (self *SwitchToRepoFilesController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:              opts.GetKeys(opts.Config.Universal.BrowseAllFilesAtCommit),
			Handler:           self.enter,
			GetDisabledReason: self.canEnter,
			Description:       self.c.Tr.BrowseAllFilesAtCommit,
			Tooltip:           self.c.Tr.BrowseAllFilesAtCommitTooltip,
		},
	}

	return bindings
}

func (self *SwitchToRepoFilesController) Context() types.Context {
	return self.context
}

func (self *SwitchToRepoFilesController) enter() error {
	ref := self.context.GetSelectedRef()
	repoFilesContext := self.c.Contexts().RepoFiles

	repoFilesContext.ClearFilter()
	repoFilesContext.ReInit(ref)
	repoFilesContext.SetParentContext(self.context)
	repoFilesContext.SetWindowName(self.context.GetWindowName())
	repoFilesContext.GetView().TitlePrefix = self.context.GetView().TitlePrefix

	return self.c.WithWaitingStatus(self.c.Tr.LoadingFilesAtRefStatus, func(gocui.Task) error {
		commitHash, err := self.c.Git().Commit.ResolveRefToCommitHash(ref.RefName())
		if err != nil {
			return err
		}

		// a commit's tree is immutable, so if we already have this commit's
		// tree loaded we can reuse it (along with its collapse and selection
		// state)
		if commitHash == repoFilesContext.GetLoadedCommitHash() {
			self.c.OnUIThread(func() error {
				self.c.PostRefreshUpdate(repoFilesContext)
				self.c.Context().Push(repoFilesContext, types.OnFocusOpts{})
				return nil
			})
			return nil
		}

		files, err := self.c.Git().Loaders.TreeFileLoader.GetTreeFiles(commitHash)
		if err != nil {
			return err
		}

		self.c.OnUIThread(func() error {
			repoFilesContext.SetFiles(files, commitHash)
			repoFilesContext.CollapseAllToTopLevel()
			repoFilesContext.SetSelection(0)
			self.c.PostRefreshUpdate(repoFilesContext)
			self.c.Context().Push(repoFilesContext, types.OnFocusOpts{})
			return nil
		})
		return nil
	})
}

func (self *SwitchToRepoFilesController) canEnter() *types.DisabledReason {
	ref := self.context.GetSelectedRef()
	if ref == nil {
		return &types.DisabledReason{Text: self.c.Tr.NoItemSelected}
	}
	if ref.RefName() == "" {
		return &types.DisabledReason{Text: self.c.Tr.SelectedItemDoesNotHaveFiles}
	}

	return nil
}
