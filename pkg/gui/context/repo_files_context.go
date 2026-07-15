package context

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gui/filetree"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation"
	"github.com/jesseduffield/lazygit/pkg/gui/presentation/icons"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/samber/lo"
)

// RepoFilesContext is the panel that shows all files in the repository as
// they existed at a given commit (as opposed to CommitFilesContext, which
// shows the files *changed* by a commit).
type RepoFilesContext struct {
	*filetree.RepoFileTreeViewModel
	*ListContextTrait
	*DynamicTitleBuilder

	// the tree files of the currently loaded commit, and the hash of that
	// commit. A commit's tree is immutable, so re-entering the panel for the
	// same commit doesn't need to reload anything.
	files            []*models.TreeFile
	loadedCommitHash string
}

var (
	_ types.IListContext       = (*RepoFilesContext)(nil)
	_ types.IFilterableContext = (*RepoFilesContext)(nil)
)

func NewRepoFilesContext(c *ContextCommon) *RepoFilesContext {
	ctx := &RepoFilesContext{}

	viewModel := filetree.NewRepoFileTreeViewModel(
		func() []*models.TreeFile { return ctx.files },
		c.Common,
		c.UserConfig().Gui.ShowFileTree,
	)

	getDisplayStrings := func(startIdx int, endIdx int) [][]string {
		showFileIcons := icons.IsIconEnabled() && c.UserConfig().Gui.ShowFileIcons
		lines := presentation.RenderRepoFileTree(viewModel, showFileIcons, &c.UserConfig().Gui.CustomIcons, startIdx, endIdx)
		return lo.Map(lines, func(line string, _ int) []string {
			return []string{line}
		})
	}

	ctx.RepoFileTreeViewModel = viewModel
	ctx.DynamicTitleBuilder = NewDynamicTitleBuilder(c.Tr.RepoFilesDynamicTitle)
	ctx.ListContextTrait = &ListContextTrait{
		Context: NewSimpleContext(
			NewBaseContext(NewBaseContextOpts{
				View:       c.Views().RepoFiles,
				WindowName: "commits",
				Key:        REPO_FILES_CONTEXT_KEY,
				Kind:       types.SIDE_CONTEXT,
				Focusable:  true,
				Transient:  true,
			}),
		),
		ListRenderer: ListRenderer{
			list:              viewModel,
			getDisplayStrings: getDisplayStrings,
		},
		renderOnlyVisibleLines: true,
		c:                      c,
	}

	return ctx
}

func (self *RepoFilesContext) ReInit(ref models.Ref) {
	self.SetRef(ref)
	self.SetTitleRef(ref.Description())
	self.GetView().Title = self.Title()
}

func (self *RepoFilesContext) GetLoadedCommitHash() string {
	return self.loadedCommitHash
}

// Replaces the loaded tree with the files of the given commit. Must be called
// on the UI thread since it rebuilds the view model's tree.
func (self *RepoFilesContext) SetFiles(files []*models.TreeFile, commitHash string) {
	self.files = files
	self.loadedCommitHash = commitHash
	self.SetTree()
}
