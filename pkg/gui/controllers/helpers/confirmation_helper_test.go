package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMenuFilterPrompt(t *testing.T) {
	longPrompt := "Filter ('@' for keybindings): "
	shortPrompt := "Filter: "

	tests := []struct {
		name         string
		candidates   []string
		contentWidth int
		expected     string
	}{
		{name: "room for four characters", candidates: []string{shortPrompt}, contentWidth: 12, expected: shortPrompt},
		{name: "room for three characters", candidates: []string{shortPrompt}, contentWidth: 11, expected: ""},
		{name: "prefers the first candidate", candidates: []string{longPrompt, shortPrompt}, contentWidth: 34, expected: longPrompt},
		{name: "falls back to the next one", candidates: []string{longPrompt, shortPrompt}, contentWidth: 33, expected: shortPrompt},
		{name: "measures display width", candidates: []string{"篩選: "}, contentWidth: 9, expected: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, menuFilterPrompt(test.candidates, test.contentWidth))
		})
	}
}

func TestMenuTitle(t *testing.T) {
	tests := []struct {
		name        string
		breadcrumbs []string
		width       int
		expected    string
	}{
		{name: "a menu that is not a submenu", breadcrumbs: []string{"Menu"}, width: 80, expected: "Menu"},
		{name: "all titles fit", breadcrumbs: []string{"Menu", "Submenu", "Subsubmenu"}, width: 27, expected: "Menu › Submenu › Subsubmenu"},
		{name: "leaves out the first title", breadcrumbs: []string{"Menu", "Submenu", "Subsubmenu"}, width: 26, expected: "… › Submenu › Subsubmenu"},
		{name: "keeps the last title even if it doesn't fit", breadcrumbs: []string{"Menu", "Submenu", "Subsubmenu"}, width: 5, expected: "… › Subsubmenu"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, menuTitle(test.breadcrumbs, test.width))
		})
	}
}
