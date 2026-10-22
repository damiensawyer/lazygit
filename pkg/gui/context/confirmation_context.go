package context

import (
	"github.com/jesseduffield/lazygit/pkg/gui/types"
)

type ConfirmationContext struct {
	*SimpleContext
	c *ContextCommon

	State ConfirmationContextState
}

type ConfirmationContextState struct {
	OnConfirm func() error
	OnClose   func() error
	// Text placed on the clipboard by the popup's copy-to-clipboard
	// keybinding. When empty, that keybinding copies the popup's whole text.
	CopyableText string
}

var _ types.Context = (*ConfirmationContext)(nil)

func NewConfirmationContext(
	c *ContextCommon,
) *ConfirmationContext {
	return &ConfirmationContext{
		c: c,
		SimpleContext: NewSimpleContext(NewBaseContext(NewBaseContextOpts{
			View:                  c.Views().Confirmation,
			WindowName:            "confirmation",
			Key:                   CONFIRMATION_CONTEXT_KEY,
			Kind:                  types.TEMPORARY_POPUP,
			Focusable:             true,
			HasUncontrolledBounds: true,
		})),
	}
}
