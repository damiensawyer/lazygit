package helpers

import (
	"path/filepath"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type CommitFilesHelper struct {
	c *HelperCommon
}

func NewCommitFilesHelper(c *HelperCommon) *CommitFilesHelper {
	return &CommitFilesHelper{c: c}
}

type ViewCommitFilesOpts struct {
	Ref       models.Ref
	RefRange  *types.RefRange
	CanRebase bool
	// the panel the files are shown in place of, and that escape goes back to
	Context types.Context
}

// ViewCommitFiles shows the files that a ref changes, or a range of refs, in the
// commit files panel.
func (self *CommitFilesHelper) ViewCommitFiles(opts ViewCommitFilesOpts) error {
	commitFilesContext := self.c.Contexts().CommitFiles

	commitFilesContext.ClearFilter()
	commitFilesContext.ReInit(opts.Ref, opts.RefRange)
	commitFilesContext.SetSelection(0)
	commitFilesContext.SetCanRebase(opts.CanRebase)
	commitFilesContext.SetParentContext(opts.Context)
	commitFilesContext.SetWindowName(opts.Context.GetWindowName())
	commitFilesContext.GetView().TitlePrefix = opts.Context.GetView().TitlePrefix

	self.c.Refresh(types.RefreshOptions{
		Scope: []types.RefreshableView{types.COMMIT_FILES},
		Then: func() error {
			if filterPath := self.c.Modes().Filtering.GetPath(); filterPath != "" {
				path, err := filepath.Rel(self.c.Git().RepoPaths.RepoPath(), filterPath)
				if err != nil {
					path = filterPath
				}
				commitFilesContext.CommitFileTreeViewModel.SelectPath(
					filepath.ToSlash(path), self.c.UserConfig().Gui.ShowRootItemInFileTree)
			}
			self.c.Context().Push(commitFilesContext, types.OnFocusOpts{})
			return nil
		},
	})
	return nil
}
