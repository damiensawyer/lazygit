package launcher

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestFuzzyFilter(t *testing.T) {
	targets := []string{
		"lord-of-dock/lazydocker",
		"jesseduffield/lazygit",
		"JesseDuffield/LazyDocker",
		"jesseduffield/horcrux",
		"kdheepak/lazygit.nvim",
	}
	matched := func(words []string, sortByScore bool) []string {
		return lo.Map(fuzzyFilter(words, targets, sortByScore), func(m match, _ int) string { return targets[m.index] })
	}

	assert.Equal(t, targets, matched(nil, false))
	assert.Equal(t,
		[]string{"jesseduffield/lazygit", "JesseDuffield/LazyDocker"},
		matched([]string{"jes", "laz"}, false))
	assert.Equal(t,
		[]string{"jesseduffield/lazygit", "kdheepak/lazygit.nvim"},
		matched([]string{"lazyg"}, false))
	assert.Empty(t, matched([]string{"jes", "zzz"}, false))
}

func TestFuzzyFilterSortsByScore(t *testing.T) {
	targets := []string{"agree/sight", "owner/git", "other/git"}
	matched := func(sortByScore bool) []string {
		return lo.Map(fuzzyFilter([]string{"git"}, targets, sortByScore), func(m match, _ int) string { return targets[m.index] })
	}

	assert.Equal(t, targets, matched(false))
	// A contiguous match beats a scattered one; equal ones keep their order.
	assert.Equal(t, []string{"owner/git", "other/git", "agree/sight"}, matched(true))
}

func TestFuzzyFilterPositions(t *testing.T) {
	matches := fuzzyFilter([]string{"jes", "laz", "jesse"}, []string{"jesseduffield/lazygit"}, false)

	assert.Len(t, matches, 1)
	assert.Equal(t, []int{0, 1, 2, 3, 4, 14, 15, 16}, matches[0].positions)
}
