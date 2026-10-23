package components

import (
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestKeyGlyph(t *testing.T) {
	scenarios := []struct {
		label    string
		expected string
	}{
		{"j", "j"},
		{"J", "J"},
		{"?", "?"},
		{"<space>", "space"},
		{"<enter>", "⏎"},
		{"<esc>", "⎋"},
		{"<tab>", "⇥"},
		{"<backtab>", "⇧⇥"},
		{"<backspace>", "⌫"},
		{"<up>", "↑"},
		{"<down>", "↓"},
		{"<left>", "←"},
		{"<right>", "→"},
		{"<c-a>", "⌃a"},
		{"<a-v>", "⌥v"},
		{"<a-]>", "⌥]"},
		{"<s-down>", "⇧↓"},
		{"<c-s-a>", "⌃⇧a"},
		{"<a-right>", "⌥→"},
		{"<c-space>", "⌃space"},
		{"<f5>", "f5"},
	}

	for _, s := range scenarios {
		t.Run(s.label, func(t *testing.T) {
			assert.Equal(t, s.expected, keyGlyph(s.label))
		})
	}
}

func keycastTexts(k *keycast) []string {
	return lo.Map(k.tokens, func(token keycastToken, _ int) string {
		return token.text
	})
}

func TestKeycastShowsPressedKeysInOrder(t *testing.T) {
	k := &keycast{}
	now := time.Now()
	for _, label := range []string{"<c-a>", "j", "j", "<space>", "<enter>"} {
		k.press(label, now)
	}

	assert.Equal(t, []string{"⌃a", "j", "j", "space", "⏎"}, keycastTexts(k))
	assert.Equal(t, " ⌃a   j   j   space   ⏎ ", utils.Decolorise(k.render()))
}

func TestKeycastMergesTypedCharacters(t *testing.T) {
	k := &keycast{}
	now := time.Now()
	k.press("c", now)
	for _, char := range "fix typo" {
		k.typeChar(string(char), now)
	}
	k.press("<enter>", now)
	k.typeChar("a", now)

	assert.Equal(t, []string{"c", "fix typo", "⏎", "a"}, keycastTexts(k))
}

func TestKeycastDropsOldestKeysThatDontFit(t *testing.T) {
	k := &keycast{}
	now := time.Now()
	for range 20 {
		k.press("j", now)
	}
	k.press("<enter>", now)

	texts := keycastTexts(k)
	assert.Less(t, len(texts), 21)
	assert.Equal(t, "⏎", texts[len(texts)-1])
	assert.LessOrEqual(t, utils.StringWidth(utils.Decolorise(k.render())), keycastMaxWidth)

	// one more keycap would no longer fit
	assert.Greater(t, utils.StringWidth(utils.Decolorise(k.render()))+len(" j  "), keycastMaxWidth)
}

func TestKeycastShowsTheEndOfLongTypedText(t *testing.T) {
	k := &keycast{}
	now := time.Now()
	k.press("c", now)
	for _, char := range "Update tagline and install instructions" {
		k.typeChar(string(char), now)
	}
	k.press("<enter>", now)

	assert.Equal(t, " c   …nd install instructions   ⏎ ", utils.Decolorise(k.render()))
}

func TestKeycastStartsAfreshAfterAPause(t *testing.T) {
	k := &keycast{}
	now := time.Now()
	k.press("j", now)
	k.press("k", now.Add(time.Second))
	assert.Equal(t, []string{"j", "k"}, keycastTexts(k))

	k.press("<enter>", now.Add(time.Second+keycastPause+time.Millisecond))
	assert.Equal(t, []string{"⏎"}, keycastTexts(k))

	k.typeChar("a", now.Add(2*time.Second+2*keycastPause))
	assert.Equal(t, []string{"a"}, keycastTexts(k))
}
