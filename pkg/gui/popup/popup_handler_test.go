package popup

import (
	"context"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/stretchr/testify/assert"
)

// newPopupHandler returns a popup handler that records the popups it is asked
// to create instead of showing them.
func newPopupHandler() (*PopupHandler, *[]types.CreatePopupPanelOpts) {
	created := []types.CreatePopupPanelOpts{}
	handler := NewPopupHandler(
		common.NewDummyCommon(),
		func(_ context.Context, opts types.CreatePopupPanelOpts) { created = append(created, opts) },
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	return handler, &created
}

func TestConfirmPassesOnCopyableText(t *testing.T) {
	scenarios := []struct {
		name         string
		copyableText string
	}{
		{name: "with copyable text", copyableText: "WDJB-MJHT"},
		{name: "without copyable text", copyableText: ""},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			handler, created := newPopupHandler()

			handler.Confirm(types.ConfirmOpts{Title: "Log in", Prompt: "Enter the code WDJB-MJHT", CopyableText: s.copyableText})

			assert.Len(t, *created, 1)
			assert.Equal(t, "Log in", (*created)[0].Title)
			assert.Equal(t, "Enter the code WDJB-MJHT", (*created)[0].Prompt)
			assert.Equal(t, s.copyableText, (*created)[0].CopyableText)
		})
	}
}

func TestConfirmIfPassesOnCopyableText(t *testing.T) {
	handler, created := newPopupHandler()

	assert.NoError(t, handler.ConfirmIf(true, types.ConfirmOpts{Prompt: "Enter the code", CopyableText: "WDJB-MJHT"}))

	assert.Len(t, *created, 1)
	assert.Equal(t, "WDJB-MJHT", (*created)[0].CopyableText)
}

func TestConfirmIfWithoutConditionConfirmsWithoutPopup(t *testing.T) {
	handler, created := newPopupHandler()

	confirmed := false
	assert.NoError(t, handler.ConfirmIf(false, types.ConfirmOpts{
		CopyableText:  "WDJB-MJHT",
		HandleConfirm: func() error { confirmed = true; return nil },
	}))

	assert.True(t, confirmed)
	assert.Empty(t, *created)
}

func TestAlertHasNoCopyableText(t *testing.T) {
	handler, created := newPopupHandler()

	handler.Alert("Title", "Message")

	assert.Len(t, *created, 1)
	assert.Empty(t, (*created)[0].CopyableText)
}
