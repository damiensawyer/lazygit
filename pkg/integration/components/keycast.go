package components

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// keycast keeps track of the keys most recently pressed in a demo, so that the
// recording can show them as keycaps, the way keycastr and screenkey do. A
// viewer can't otherwise tell which key made something happen.
type keycast struct {
	tokens    []keycastToken
	lastInput time.Time
}

type keycastToken struct {
	text string
	// Text that was typed rather than pressed as a command. Consecutive typed
	// characters make up a single token, because "fix typo" reads better than
	// eight separate keycaps.
	typed bool
}

const (
	// The width that the keycaps may take up between them, so that the box
	// fits in the corner of the screen without covering much of what the demo
	// is showing.
	keycastMaxWidth = 40
	// Typed text longer than this shows only its end, since that's where the
	// typing is happening. This leaves room for the keys pressed after it,
	// such as the enter that confirms it.
	keycastMaxTextWidth = 24
	// After a pause this long, the next key starts afresh, so that the keys of
	// one step of a demo don't run into those of the next one.
	keycastPause = 2500 * time.Millisecond
	// Each keycap is padded with this much space on either side
	keycastPadding = " "
	keycastGap     = " "
)

// press records a key that was pressed, given by its label as the keybinding
// config spells it, e.g. "j" or "<c-a>".
func (self *keycast) press(label string, now time.Time) {
	self.startAfreshAfterPause(now)
	self.tokens = append(self.tokens, keycastToken{text: keyGlyph(label)})
	self.dropOldTokens()
}

// typeChar records a character that was typed into a text field.
func (self *keycast) typeChar(char string, now time.Time) {
	self.startAfreshAfterPause(now)
	if len(self.tokens) > 0 && self.tokens[len(self.tokens)-1].typed {
		self.tokens[len(self.tokens)-1].text += char
	} else {
		self.tokens = append(self.tokens, keycastToken{text: char, typed: true})
	}
	self.dropOldTokens()
}

func (self *keycast) startAfreshAfterPause(now time.Time) {
	if now.Sub(self.lastInput) > keycastPause {
		self.tokens = nil
	}
	self.lastInput = now
}

func (self *keycast) dropOldTokens() {
	for len(self.tokens) > 1 && self.width() > keycastMaxWidth {
		self.tokens = self.tokens[1:]
	}
}

func (self *keycast) width() int {
	return utils.StringWidth(utils.Decolorise(self.render()))
}

// render returns the keycaps, oldest first, with the most recent one
// highlighted.
func (self *keycast) render() string {
	keycaps := lo.Map(self.tokens, func(token keycastToken, i int) string {
		textStyle := style.New().SetReverse()
		if i == len(self.tokens)-1 {
			textStyle = style.FgCyan.SetReverse().SetBold()
		}
		text := truncateStart(token.text, keycastMaxTextWidth)
		return textStyle.Sprint(keycastPadding + text + keycastPadding)
	})

	return strings.Join(keycaps, keycastGap)
}

func truncateStart(text string, width int) string {
	if utils.StringWidth(text) <= width {
		return text
	}

	runes := []rune(text)
	for len(runes) > 0 && utils.StringWidth(string(runes)) > width-1 {
		runes = runes[1:]
	}
	return "…" + string(runes)
}

var keyGlyphByName = map[gocui.KeyName]string{
	gocui.KeyEnter:      "⏎",
	gocui.KeyEsc:        "⎋",
	gocui.KeyTab:        "⇥",
	gocui.KeyBacktab:    "⇧⇥",
	gocui.KeyBackspace:  "⌫",
	gocui.KeyDelete:     "⌦",
	gocui.KeyArrowUp:    "↑",
	gocui.KeyArrowDown:  "↓",
	gocui.KeyArrowLeft:  "←",
	gocui.KeyArrowRight: "→",
	gocui.KeyPgup:       "⇞",
	gocui.KeyPgdn:       "⇟",
	gocui.KeyHome:       "home",
	gocui.KeyEnd:        "end",
}

// keyGlyph returns how a keycap shows the key with the given label, e.g. "⌃a"
// for "<c-a>".
func keyGlyph(label string) string {
	key, ok := config.KeyFromLabel(label)
	if !ok {
		return label
	}

	glyph := ""
	if key.Mod()&gocui.ModCtrl != 0 {
		glyph += "⌃"
	}
	if key.Mod()&gocui.ModAlt != 0 {
		glyph += "⌥"
	}
	if key.Mod()&gocui.ModShift != 0 {
		glyph += "⇧"
	}
	if key.Mod()&gocui.ModMeta != 0 {
		glyph += "⌘"
	}

	if key.KeyName() == gocui.KeyName(tcell.KeyRune) {
		if key.Str() == " " {
			return glyph + "space"
		}
		return glyph + key.Str()
	}

	if name, ok := keyGlyphByName[key.KeyName()]; ok {
		return glyph + name
	}

	// The function keys and the like read well enough as they're labelled
	return glyph + strings.Trim(config.LabelForKey(gocui.NewKey(key.KeyName(), "", gocui.ModNone)), "<>")
}
