package controllers

import (
	"errors"
	"fmt"
	"math"

	"github.com/jesseduffield/lazygit/pkg/gui/context"
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
	if err := self.checkCanChangeContext(); err != nil {
		return err
	}

	if self.c.UserConfig().Git.DiffContextSize < math.MaxUint64 {
		self.c.UserConfig().Git.DiffContextSize++
	}
	return self.applyChange()
}

func (self *ContextLinesController) Decrease() error {
	if err := self.checkCanChangeContext(); err != nil {
		return err
	}

	if self.c.UserConfig().Git.DiffContextSize > 0 {
		self.c.UserConfig().Git.DiffContextSize--
	}
	return self.applyChange()
}

func (self *ContextLinesController) Toggle() error {
	if err := self.checkCanChangeContext(); err != nil {
		return err
	}

	currentSize := self.c.UserConfig().Git.DiffContextSize
	if currentSize == math.MaxUint64 {
		// Currently showing the full file: restore previous context size.
		saved := self.c.UserConfig().Git.DiffContextSizeSaved
		if saved > 0 {
			self.c.UserConfig().Git.DiffContextSize = saved
			self.c.Toast(fmt.Sprintf(self.c.Tr.DiffContextSizeChanged, saved))
		} else {
			// No saved value (unusual), just decrease to a reasonable default.
			self.c.UserConfig().Git.DiffContextSize = 3
			self.c.Toast(fmt.Sprintf(self.c.Tr.DiffContextSizeChanged, 3))
		}
		currentContext := self.c.Context().CurrentSide()
		switch currentContext.GetKey() {
		case context.PATCH_BUILDING_MAIN_CONTEXT_KEY:
			self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.PATCH_BUILDING}})
		case context.STAGING_MAIN_CONTEXT_KEY, context.STAGING_SECONDARY_CONTEXT_KEY:
			self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.STAGING}})
		default:
			currentContext.HandleRenderToMain()
		}
		return nil
	}

	// Currently in normal mode: save current size and show full file.
	self.c.UserConfig().Git.DiffContextSizeSaved = currentSize
	self.c.UserConfig().Git.DiffContextSize = math.MaxUint64
	self.c.Toast(self.c.Tr.FullFileDiffView)
	currentContext := self.c.Context().CurrentSide()
	switch currentContext.GetKey() {
	case context.PATCH_BUILDING_MAIN_CONTEXT_KEY:
		self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.PATCH_BUILDING}})
	case context.STAGING_MAIN_CONTEXT_KEY, context.STAGING_SECONDARY_CONTEXT_KEY:
		self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.STAGING}})
	default:
		currentContext.HandleRenderToMain()
	}
	return nil
}

func (self *ContextLinesController) applyChange() error {
	self.c.Toast(fmt.Sprintf(self.c.Tr.DiffContextSizeChanged, self.c.UserConfig().Git.DiffContextSize))

	currentContext := self.c.Context().CurrentSide()
	switch currentContext.GetKey() {
	// we make an exception for our staging and patch building contexts because they actually need to refresh their state afterwards.
	case context.PATCH_BUILDING_MAIN_CONTEXT_KEY:
		self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.PATCH_BUILDING}})
	case context.STAGING_MAIN_CONTEXT_KEY, context.STAGING_SECONDARY_CONTEXT_KEY:
		self.c.Refresh(types.RefreshOptions{Scope: []types.RefreshableView{types.STAGING}})
	default:
		currentContext.HandleRenderToMain()
	}
	return nil
}

func (self *ContextLinesController) checkCanChangeContext() error {
	if self.c.Git().Patch.PatchBuilder.Active() {
		return errors.New(self.c.Tr.CantChangeContextSizeError)
	}

	return nil
}
