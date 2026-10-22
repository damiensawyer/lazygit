package gui

import (
	"errors"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

const keycastViewName = "keycast"

func (gui *Gui) setKeycast(keys string) {
	gui.keycast = keys
	if view, err := gui.g.View(keycastViewName); err == nil {
		view.SetContent(keycastContent(keys))
	}
	gui.c.Render()
}

// one column of padding on either side of the keys, inside the frame
func keycastContent(keys string) string {
	return " " + keys + " "
}

// layoutKeycast shows the keys that a demo pressed most recently in a small box
// in the bottom right corner, just above the caption, so that whoever watches
// the recording can tell what made things happen. Only demos have this box;
// the view isn't even created otherwise.
func (gui *Gui) layoutKeycast(width int, height int) error {
	keysWidth := utils.StringWidth(utils.Decolorise(gui.keycast))

	x1 := width - 1
	x0 := x1 - utils.StringWidth(keycastContent("")) - keysWidth - 1
	y1 := height - 2
	y0 := y1 - 2

	view, err := gui.g.SetView(keycastViewName, x0, y0, x1, y1, 0)
	if err != nil {
		if !errors.Is(err, gocui.ErrUnknownView) {
			return err
		}
		view.Frame = true
		view.FrameRunes = []rune{'─', '│', '╭', '╮', '╰', '╯'}
		view.FrameColor = gocui.ColorCyan
		view.SetContent(keycastContent(gui.keycast))
	}

	view.Visible = keysWidth > 0

	// Popups are moved to the top when they open, but the keycast box has to
	// stay in front of them, since it shows the key that opened them.
	_, err = gui.g.SetViewOnTop(keycastViewName)
	return err
}
