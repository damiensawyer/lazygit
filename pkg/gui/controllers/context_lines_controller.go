package controllers

import (
	"fmt"
	"math"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

// This controller lets you change the context size for diffs. The 'context' in 'context size' refers to the conventional meaning of the word 'context' in a diff, as opposed to lazygit's own idea of a 'context'.

type ContextLinesController struct {
	baseController
	c *ControllerCommon
}

var _ types.IController = &ContextLinesController{}

func NewContextLinesController(
	c *ControllerCommon,
) *ContextLinesController {
	return &ContextLinesController{
		baseController: baseController{},
		c:              c,
	}
}

func (self *ContextLinesController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	bindings := []*types.Binding{
		{
			Keys:        opts.GetKeys(opts.Config.Universal.IncreaseContextInDiffView),
			Handler:     self.Increase,
			Description: self.c.Tr.IncreaseContextInDiffView,
			Tooltip:     self.c.Tr.IncreaseContextInDiffViewTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.DecreaseContextInDiffView),
			Handler:     self.Decrease,
			Description: self.c.Tr.DecreaseContextInDiffView,
			Tooltip:     self.c.Tr.DecreaseContextInDiffViewTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.ToggleContextInDiffView),
			Handler:     self.Toggle,
			Description: self.c.Tr.ToggleContextInDiffView,
			Tooltip:     self.c.Tr.ToggleContextInDiffViewTooltip,
		},
	}

	return bindings
}

func (self *ContextLinesController) Context() types.Context {
	return nil
}

func (self *ContextLinesController) Increase() error {
	if self.c.UserConfig().Git.DiffContextSize < math.MaxUint64 {
		self.c.UserConfig().Git.DiffContextSize++
	}
	return self.applyChange()
}

func (self *ContextLinesController) Decrease() error {
	if self.c.UserConfig().Git.DiffContextSize > 0 {
		self.c.UserConfig().Git.DiffContextSize--
	}
	return self.applyChange()
}

func (self *ContextLinesController) Toggle() error {
	currentSize := self.c.UserConfig().Git.DiffContextSize
	if currentSize == math.MaxUint64 {
		// Currently showing the full file: restore previous context size.
		saved := self.c.UserConfig().Git.DiffContextSizeSaved
		if saved == 0 {
			// No saved value (unusual), just decrease to a reasonable default.
			saved = 3
		}
		self.c.UserConfig().Git.DiffContextSize = saved
		self.c.Toast(fmt.Sprintf(self.c.Tr.RestoreDiffContextSize, saved))
	} else {
		self.c.UserConfig().Git.DiffContextSizeSaved = currentSize
		self.c.UserConfig().Git.DiffContextSize = math.MaxUint64
		self.c.Toast(self.c.Tr.FullFileDiffView)
	}
	self.rerender()
	return nil
}

func (self *ContextLinesController) applyChange() error {
	self.c.Toast(fmt.Sprintf(self.c.Tr.DiffContextSizeChanged, self.c.UserConfig().Git.DiffContextSize))
	self.rerender()
	return nil
}

func (self *ContextLinesController) rerender() {
	currentContext := self.c.Context().CurrentSide()
	// The diff is about to be rendered again with more or less context around
	// each change, which reads as the lines you were looking at moving up or down
	// the view; keep them where they are instead.
	self.c.Helpers().DiffLine.PreserveDiffPositionOnRerender(self.c.Contexts().Normal.GetView())
	self.c.Helpers().DiffLine.PreserveDiffPositionOnRerender(self.c.Contexts().NormalSecondary.GetView())
	currentContext.HandleRenderToMain()
}
