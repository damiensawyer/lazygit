package helpers

import (
	"os"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/highlight"
	"github.com/jesseduffield/lazygit/pkg/tasks"
)

// binarySniffLength is how much of a file we inspect for a NUL byte before
// calling it binary, matching the heuristic git itself uses.
const binarySniffLength = 8000

// FinalVersionHelper renders a changed file's full contents to the main view in
// place of a diff of it, so that the file list can be used to read through the
// files a change set touches rather than only what changed in them.
type FinalVersionHelper struct {
	c *HelperCommon

	// Reading and syntax-highlighting a file costs roughly a millisecond per
	// kilobyte, which is enough to make scrolling through the file list stutter
	// if done on the UI thread. The handler moves that work to a worker and
	// drops all but the newest result, so holding down a movement key doesn't
	// queue up renders of files that have already been scrolled past.
	asyncHandler *tasks.AsyncHandler
}

func NewFinalVersionHelper(c *HelperCommon) *FinalVersionHelper {
	return &FinalVersionHelper{
		c:            c,
		asyncHandler: tasks.NewAsyncHandler(c.OnWorker),
	}
}

// RenderWorktreeFile renders the contents of a file as it currently exists on
// disk.
func (self *FinalVersionHelper) RenderWorktreeFile(path string) {
	self.render(path, func() (string, error) {
		content, err := os.ReadFile(path)
		return string(content), err
	})
}

// RenderCommitFile renders the contents of a file as of the given commit.
func (self *FinalVersionHelper) RenderCommitFile(hash string, path string) {
	self.render(path, func() (string, error) {
		return self.c.Git().Commit.ShowFileContentCmdObj(hash, path).RunWithOutput()
	})
}

func (self *FinalVersionHelper) render(path string, read func() (string, error)) {
	// Read on the UI thread rather than inside the worker, because the config
	// isn't safe to read from one.
	styleName := self.c.UserConfig().Gui.SyntaxHighlightStyle

	self.asyncHandler.Do(func() func() {
		task := self.buildTask(path, styleName, read)

		return func() {
			self.c.OnUIThread(func() error {
				// The toggle can be turned off while we're highlighting, in
				// which case the diff has already been rendered and this result
				// would overwrite it.
				if !self.c.State().GetShowFinalVersion() {
					return nil
				}

				self.c.RenderToMainViews(types.RefreshMainOpts{
					Pair: self.c.MainViewPairs().Normal,
					Main: &types.ViewUpdateOpts{
						Title: self.c.Tr.FinalVersionTitle,
						Task:  task,
					},
				})
				return nil
			})
		}
	})
}

func (self *FinalVersionHelper) buildTask(path string, styleName string, read func() (string, error)) types.UpdateTask {
	content, err := read()
	if err != nil {
		// The file is in the list because the change set touched it, so the
		// only way it can't be read is that the change deleted it.
		return types.NewRenderStringTask(self.c.Tr.FinalVersionMissing)
	}

	if isBinary(content) {
		// Writing raw bytes to the view would corrupt the display.
		return types.NewRenderStringTask(self.c.Tr.FinalVersionOfBinaryFile)
	}

	return types.NewRenderStringTask(highlight.FileContent(path, content, styleName))
}

func isBinary(content string) bool {
	return strings.ContainsRune(content[:min(len(content), binarySniffLength)], 0)
}
