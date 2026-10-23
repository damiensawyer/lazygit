package controllers

import (
	"fmt"
	"math"
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
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
	// The full-file diff already shows as much context as there is
	if self.c.UserConfig().Git.DiffContextSize == math.MaxUint64 {
		self.c.Toast(self.c.Tr.AlreadyShowingFullFileDiff)
		return nil
	}

	self.c.UserConfig().Git.DiffContextSize++
	return self.applyChange()
}

func (self *ContextLinesController) Decrease() error {
	// Less than the full file is what the full-file diff was toggled on from
	if self.c.UserConfig().Git.DiffContextSize == math.MaxUint64 {
		self.restoreSavedSize()
		self.rerender()
		return nil
	}

	if self.c.UserConfig().Git.DiffContextSize > 0 {
		self.c.UserConfig().Git.DiffContextSize--
	}
	return self.applyChange()
}

func (self *ContextLinesController) Toggle() error {
	currentSize := self.c.UserConfig().Git.DiffContextSize
	if currentSize == math.MaxUint64 {
		self.restoreSavedSize()
	} else {
		self.c.UserConfig().Git.DiffContextSizeSaved = &currentSize
		self.c.UserConfig().Git.DiffContextSize = math.MaxUint64
		self.c.Toast(self.c.Tr.FullFileDiffView)
	}
	self.rerender()
	return nil
}

// restoreSavedSize leaves the full-file diff for the context size that was in
// use before it.
func (self *ContextLinesController) restoreSavedSize() {
	// With no saved value (the config started out showing the full file),
	// fall back to a reasonable default.
	saved := uint64(3)
	if self.c.UserConfig().Git.DiffContextSizeSaved != nil {
		saved = *self.c.UserConfig().Git.DiffContextSizeSaved
	}
	self.c.UserConfig().Git.DiffContextSize = saved
	self.c.Toast(utils.ResolvePlaceholderString(
		self.c.Tr.RestoreDiffContextSize,
		map[string]string{"size": strconv.FormatUint(saved, 10)},
	))
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
