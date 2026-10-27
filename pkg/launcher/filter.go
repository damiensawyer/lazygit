package launcher

import (
	"slices"

	"github.com/sahilm/fuzzy"
	"github.com/samber/lo"
)

type match struct {
	// The index of the matching target.
	index int
	score int
	// The byte offsets in the target of the characters the words matched, to
	// highlight; sorted and without duplicates.
	positions []int
}

// fuzzyFilter returns the targets that every one of words fuzzy-matches,
// ignoring case: "jes laz" matches "jesseduffield/lazygit". With sortByScore,
// the best matches come first; otherwise, and among equally good ones, they
// stay in the order of targets. With no words, every target matches.
func fuzzyFilter(words []string, targets []string, sortByScore bool) []match {
	matches := lo.Times(len(targets), func(i int) *match { return &match{index: i} })
	for _, word := range words {
		matchesOfWord := make([]*fuzzy.Match, len(targets))
		for _, m := range fuzzy.FindNoSort(word, targets) {
			matchesOfWord[m.Index] = &m
		}
		for i, m := range matches {
			if m == nil {
				continue
			}
			if matchesOfWord[i] == nil {
				matches[i] = nil
				continue
			}
			m.score += matchesOfWord[i].Score
			m.positions = append(m.positions, matchesOfWord[i].MatchedIndexes...)
		}
	}

	result := []match{}
	for _, m := range matches {
		if m != nil {
			slices.Sort(m.positions)
			m.positions = slices.Compact(m.positions)
			result = append(result, *m)
		}
	}
	if sortByScore {
		slices.SortStableFunc(result, func(a, b match) int { return b.score - a.score })
	}
	return result
}
