package controllers

import (
	"strconv"

	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

type GlobalController struct {
	baseController
	c *ControllerCommon
}

func NewGlobalController(
	c *ControllerCommon,
) *GlobalController {
	return &GlobalController{
		baseController: baseController{},
		c:              c,
	}
}

func (self *GlobalController) GetKeybindings(opts types.KeybindingsOpts) []*types.Binding {
	return []*types.Binding{
		{
			Keys:        opts.GetKeys(opts.Config.Universal.ExecuteShellCommand),
			Handler:     self.shellCommand,
			Description: self.c.Tr.ExecuteShellCommand,
			Tooltip:     self.c.Tr.ExecuteShellCommandTooltip,
			OpensMenu:   true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.CreatePatchOptionsMenu),
			Handler:     self.createCustomPatchOptionsMenu,
			Description: self.c.Tr.ViewPatchOptions,
			OpensMenu:   true,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.CreateRebaseOptionsMenu),
			Handler:           opts.Guards.NoPopupPanel(self.c.Helpers().MergeAndRebase.CreateRebaseOptionsMenu),
			Description:       self.c.Tr.ViewMergeRebaseOptions,
			Tooltip:           self.c.Tr.ViewMergeRebaseOptionsTooltip,
			OpensMenu:         true,
			GetDisabledReason: self.canShowRebaseOptions,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Refresh),
			Handler:     opts.Guards.NoPopupPanel(self.refresh),
			Description: self.c.Tr.Refresh,
			Tooltip:     self.c.Tr.RefreshTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.NextScreenMode),
			Handler:     opts.Guards.NoPopupPanel(self.nextScreenMode),
			Description: self.c.Tr.NextScreenMode,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.PrevScreenMode),
			Handler:     opts.Guards.NoPopupPanel(self.prevScreenMode),
			Description: self.c.Tr.PrevScreenMode,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.DecreaseSidePanelWidth),
			Handler:     opts.Guards.NoPopupPanel(self.decreaseSidePanelWidth),
			Description: self.c.Tr.DecreaseSidePanelWidth,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.IncreaseSidePanelWidth),
			Handler:     opts.Guards.NoPopupPanel(self.increaseSidePanelWidth),
			Description: self.c.Tr.IncreaseSidePanelWidth,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.ResetSidePanelWidth),
			Handler:     opts.Guards.NoPopupPanel(self.resetSidePanelWidth),
			Description: self.c.Tr.ResetSidePanelWidth,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.CycleDiffRenderers),
			Handler:           opts.Guards.NoPopupPanel(self.cycleDiffRenderers),
			GetDisabledReason: self.canCycleDiffRenderers,
			Description:       self.c.Tr.CycleDiffRenderers,
			Tooltip:           self.c.Tr.CycleDiffRenderersTooltip,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.CycleDiffRenderersReverse),
			Handler:           opts.Guards.NoPopupPanel(self.cycleDiffRenderersBackward),
			GetDisabledReason: self.canCycleDiffRenderers,
			Description:       self.c.Tr.CycleDiffRenderersReverse,
			Tooltip:           self.c.Tr.CycleDiffRenderersReverseTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.ToggleShowFinalVersion),
			Handler:     opts.Guards.NoPopupPanel(self.toggleShowFinalVersion),
			Description: self.c.Tr.ToggleShowFinalVersion,
			Tooltip:     self.c.Tr.ToggleShowFinalVersionTooltip,
		},
		{
			Keys:              opts.GetKeys(opts.Config.Universal.Return),
			Handler:           self.escape,
			Description:       self.c.Tr.Cancel,
			DescriptionFunc:   self.escapeDescription,
			GetDisabledReason: self.escapeEnabled,
			DisplayOnScreen:   true,
		},
		{
			ViewName:          "",
			Keys:              opts.GetKeys(opts.Config.Universal.OptionMenu),
			Description:       self.c.Tr.OpenKeybindingsMenu,
			ShortDescription:  self.c.Tr.Keybindings,
			Handler:           self.createOptionsMenu,
			GetDisabledReason: self.optionsMenuDisabledReason,
			OpensMenu:         true,
			DisplayOnScreen:   true,
		},
		{
			ViewName:    "",
			Keys:        opts.GetKeys(opts.Config.Universal.FilteringMenu),
			Handler:     opts.Guards.NoPopupPanel(self.createFilteringMenu),
			Description: self.c.Tr.OpenFilteringMenu,
			Tooltip:     self.c.Tr.OpenFilteringMenuTooltip,
			OpensMenu:   true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.DiffingMenu),
			Handler:     opts.Guards.NoPopupPanel(self.createDiffingMenu),
			Description: self.c.Tr.ViewDiffingOptions,
			Tooltip:     self.c.Tr.ViewDiffingOptionsTooltip,
			OpensMenu:   true,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.Quit),
			Description: self.c.Tr.Quit,
			Handler:     self.quit,
		},
		{
			Keys:    opts.GetKeys(opts.Config.Universal.QuitWithoutChangingDirectory),
			Handler: self.quitWithoutChangingDirectory,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.SuspendApp),
			Handler:     self.c.Helpers().SuspendResume.SuspendApp,
			Description: self.c.Tr.SuspendApp,
			GetDisabledReason: func() *types.DisabledReason {
				if !self.c.Helpers().SuspendResume.CanSuspendApp() {
					return &types.DisabledReason{
						Text: self.c.Tr.CannotSuspendApp,
					}
				}
				return nil
			},
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.ToggleWhitespaceInDiffView),
			Handler:     self.toggleWhitespace,
			Description: self.c.Tr.ToggleWhitespaceInDiffView,
			Tooltip:     self.c.Tr.ToggleWhitespaceInDiffViewTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.EditConfig),
			Handler:     self.editConfig,
			Description: self.c.Tr.EditConfig,
			Tooltip:     self.c.Tr.EditFileTooltip,
		},
		{
			Keys:        opts.GetKeys(opts.Config.Universal.OpenGithubActions),
			Handler:     opts.Guards.NoPopupPanel(self.c.Helpers().GithubActions.Open),
			Description: self.c.Tr.GithubActions.OpenGithubActions,
			Tooltip:     self.c.Tr.GithubActions.OpenGithubActionsTooltip,
		},
	}
}

func (self *GlobalController) Context() types.Context {
	return nil
}

func (self *GlobalController) shellCommand() error {
	return (&ShellCommandAction{c: self.c}).Call()
}

func (self *GlobalController) createCustomPatchOptionsMenu() error {
	return (&CustomPatchOptionsMenuAction{c: self.c}).Call()
}

func (self *GlobalController) refresh() error {
	self.c.Refresh(types.RefreshOptions{})
	return nil
}

func (self *GlobalController) nextScreenMode() error {
	return (&ScreenModeActions{c: self.c}).Next()
}

func (self *GlobalController) prevScreenMode() error {
	return (&ScreenModeActions{c: self.c}).Prev()
}

const (
	minSidePanelWidth = 0.1
	maxSidePanelWidth = 0.9
)

func (self *GlobalController) decreaseSidePanelWidth() error {
	return self.adjustSidePanelWidth(-0.05)
}

func (self *GlobalController) increaseSidePanelWidth() error {
	return self.adjustSidePanelWidth(0.05)
}

func (self *GlobalController) adjustSidePanelWidth(delta float64) error {
	setting, ok := self.sidePanelWidthSetting()
	if !ok {
		return nil
	}

	width := setting.current + delta
	if width < minSidePanelWidth {
		width = minSidePanelWidth
	}
	if width > maxSidePanelWidth {
		width = maxSidePanelWidth
	}

	setting.apply(width)
	self.c.Refresh(types.RefreshOptions{})
	self.c.Toast(utils.ResolvePlaceholderString(
		self.c.Tr.SidePanelWidthChanged,
		map[string]string{"width": strconv.FormatFloat(width, 'f', 2, 64)},
	))
	return nil
}

func (self *GlobalController) resetSidePanelWidth() error {
	setting, ok := self.sidePanelWidthSetting()
	if !ok {
		return nil
	}

	setting.apply(setting.resetTo)
	self.c.Refresh(types.RefreshOptions{})
	self.c.Toast(self.c.Tr.ResetSidePanelWidth)
	return nil
}

// sidePanelWidthSetting describes the adjustable side panel width for the
// current screen mode: its current value, the value to restore on reset, and
// how to persist a new value. There is no setting in full screen mode, where
// the focused panel takes the entire screen.
type sidePanelWidthSetting struct {
	current float64
	resetTo float64
	apply   func(float64)
}

func (self *GlobalController) sidePanelWidthSetting() (sidePanelWidthSetting, bool) {
	repoState := self.c.State().GetRepoState()
	switch repoState.GetScreenMode() {
	case types.SCREEN_NORMAL:
		return sidePanelWidthSetting{
			current: repoState.GetSidePanelWidth(),
			resetTo: self.c.UserConfig().Gui.SidePanelWidth,
			apply: func(width float64) {
				repoState.SetSidePanelWidth(width)
				self.c.SaveSidePanelWidth()
			},
		}, true
	case types.SCREEN_HALF:
		return sidePanelWidthSetting{
			current: repoState.GetHalfScreenModeSidePanelWidth(),
			resetTo: 0.5,
			apply: func(width float64) {
				repoState.SetHalfScreenModeSidePanelWidth(width)
				self.c.SaveHalfScreenModeSidePanelWidth()
			},
		}, true
	default:
		return sidePanelWidthSetting{}, false
	}
}

func (self *GlobalController) cycleDiffRenderers() error {
	self.c.State().GetDiffRendererConfigManager().CycleDiffRenderers()
	self.onDiffRenderersChanged()
	return nil
}

func (self *GlobalController) cycleDiffRenderersBackward() error {
	self.c.State().GetDiffRendererConfigManager().CycleDiffRenderersBackward()
	self.onDiffRenderersChanged()
	return nil
}

// rerenderMainView re-renders the main view from the current side panel, so
// that a change to how the main view is rendered takes effect immediately. It
// does nothing unless the main view belongs to the current side panel, since
// otherwise there is nothing on screen to bring up to date.
func (self *GlobalController) rerenderMainView() {
	currentSide := self.c.Context().CurrentSide()
	currentKey := self.c.Context().Current().GetKey()
	if currentSide.GetKey() == currentKey ||
		currentKey == context.NORMAL_MAIN_CONTEXT_KEY ||
		currentKey == context.NORMAL_SECONDARY_CONTEXT_KEY {
		currentSide.HandleRenderToMain()
	}
}

// onDiffRenderersChanged re-renders the main view so the newly selected diff
// renderer takes effect, and shows a toast naming it.
func (self *GlobalController) onDiffRenderersChanged() {
	self.rerenderMainView()

	diffRendererConfigManager := self.c.State().GetDiffRendererConfigManager()
	current, total := diffRendererConfigManager.CurrentDiffRendererIndex()
	name := diffRendererConfigManager.CurrentDiffRendererName(self.c.Tr)
	self.c.Toast(utils.ResolvePlaceholderString(self.c.Tr.SelectedDiffRenderers, map[string]string{
		"name":    name,
		"current": strconv.Itoa(current + 1),
		"total":   strconv.Itoa(total),
	}))
}

func (self *GlobalController) toggleShowFinalVersion() error {
	showFinalVersion := !self.c.State().GetShowFinalVersion()
	self.c.State().SetShowFinalVersion(showFinalVersion)

	self.rerenderMainView()

	if showFinalVersion {
		self.c.Toast(self.c.Tr.ShowingFinalVersion)
	} else {
		self.c.Toast(self.c.Tr.ShowingDiffs)
	}

	return nil
}

func (self *GlobalController) canCycleDiffRenderers() *types.DisabledReason {
	_, total := self.c.State().GetDiffRendererConfigManager().CurrentDiffRendererIndex()
	if total <= 1 {
		return &types.DisabledReason{
			Text: self.c.Tr.CycleDiffRenderersDisabledReason,
		}
	}
	return nil
}

func (self *GlobalController) createOptionsMenu() error {
	return (&OptionsMenuAction{c: self.c}).Call()
}

func (self *GlobalController) optionsMenuDisabledReason() *types.DisabledReason {
	ctx := self.c.Context().Current()
	// Don't show options menu while displaying popup.
	if ctx.GetKind() == types.PERSISTENT_POPUP || ctx.GetKind() == types.TEMPORARY_POPUP {
		// The empty error text is intentional. We don't want to show an error
		// toast for this, but only hide it from the options map.
		return &types.DisabledReason{Text: ""}
	}
	return nil
}

func (self *GlobalController) createFilteringMenu() error {
	return (&FilteringMenuAction{c: self.c}).Call()
}

func (self *GlobalController) createDiffingMenu() error {
	return (&DiffingMenuAction{c: self.c}).Call()
}

func (self *GlobalController) quit() error {
	return (&QuitActions{c: self.c}).Quit()
}

func (self *GlobalController) quitWithoutChangingDirectory() error {
	return (&QuitActions{c: self.c}).QuitWithoutChangingDirectory()
}

func (self *GlobalController) escape() error {
	return (&QuitActions{c: self.c}).Escape()
}

func (self *GlobalController) escapeDescription() string {
	return (&QuitActions{c: self.c}).EscapeDescription()
}

func (self *GlobalController) escapeEnabled() *types.DisabledReason {
	if (&QuitActions{c: self.c}).EscapeEnabled() {
		return nil
	}

	// The empty error text is intentional. We don't want to show an error
	// toast for this, but only hide it from the options map.
	return &types.DisabledReason{Text: ""}
}

func (self *GlobalController) toggleWhitespace() error {
	return (&ToggleWhitespaceAction{c: self.c}).Call()
}

func (self *GlobalController) editConfig() error {
	return (&EditConfigAction{c: self.c}).Call()
}

func (self *GlobalController) canShowRebaseOptions() *types.DisabledReason {
	if self.c.Model().WorkingTreeStateAtLastCommitRefresh.None() {
		return &types.DisabledReason{
			Text: self.c.Tr.NotMergingOrRebasing,
		}
	}
	return nil
}
