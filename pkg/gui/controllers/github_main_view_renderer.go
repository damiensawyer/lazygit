package controllers

import "github.com/jesseduffield/lazygit/pkg/gui/types"

// githubMainViewRenderer renders a GitHub Actions view's selected item to the
// main view. These views refresh themselves every few seconds, and an item
// that is rendered again keeps its scroll position, so that a refresh doesn't
// yank the log being read back to the top.
type githubMainViewRenderer struct {
	c     *ControllerCommon
	title func() string

	renderedItemId string
}

func (self *githubMainViewRenderer) render(itemId string, content string) {
	var task types.UpdateTask = types.NewRenderStringTask(content)
	if itemId != "" && itemId == self.renderedItemId {
		task = types.NewRenderStringWithoutScrollTask(content)
	}
	self.renderedItemId = itemId

	self.c.RenderToMainViews(types.RefreshMainOpts{
		Pair: self.c.MainViewPairs().Normal,
		Main: &types.ViewUpdateOpts{
			Title: self.title(),
			Task:  task,
		},
	})
}
