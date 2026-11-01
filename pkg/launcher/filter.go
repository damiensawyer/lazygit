package launcher

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

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

// fuzzyFilter returns the targets that every one of words matches, ignoring
// case: "jes laz" matches "jesseduffield/lazygit". With sortByScore, the best
// matches come first; otherwise, and among equally good ones, they stay in the
// order of targets. With no words, every target matches.
//
// A word matches within a single segment of the target — one of the words the
// target is made of, like the "samtests" of "damiensawyer/samtests". Matching
// spread over several segments only connects stray letters, so it counts only
// when no target matches the word within one segment; that keeps finds like
// "odam" in "damiensawyer/NodaMoney" possible.
func fuzzyFilter(words []string, targets []string, sortByScore bool) []match {
	matches := lo.Times(len(targets), func(i int) *match { return &match{index: i} })
	for _, word := range words {
		matchesOfWord := matchWord(word, targets)
		for i, m := range matches {
			if m == nil {
				continue
			}
			if matchesOfWord[i] == nil {
				matches[i] = nil
				continue
			}
			m.score += matchesOfWord[i].score
			m.positions = append(m.positions, matchesOfWord[i].positions...)
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

// matchWord matches word against every target, giving the best match of each
// target, or nil where it doesn't match.
func matchWord(word string, targets []string) []*match {
	texts := []string{}
	owners := []int{}
	offsets := []int{}
	for i, target := range targets {
		for _, segment := range segmentsOf(target) {
			texts = append(texts, segment.text)
			owners = append(owners, i)
			offsets = append(offsets, segment.offset)
		}
	}

	best := make([]*match, len(targets))
	withinSegment := false
	for _, m := range fuzzy.FindNoSort(word, texts) {
		withinSegment = true
		target := owners[m.Index]
		matchOfSegment := &match{
			index: target,
			score: m.Score,
			positions: lo.Map(m.MatchedIndexes, func(p, _ int) int {
				return p + offsets[m.Index]
			}),
		}
		// On equal scores the earlier segment wins.
		if best[target] == nil || matchOfSegment.score > best[target].score {
			best[target] = matchOfSegment
		}
	}
	if withinSegment {
		return best
	}

	matches := make([]*match, len(targets))
	for _, m := range fuzzy.FindNoSort(word, targets) {
		matches[m.Index] = &match{index: m.Index, score: m.Score, positions: m.MatchedIndexes}
	}
	return matches
}

// segment is a run of a target between separators, or a camelCase run, that a
// query word may be matched within.
type segment struct {
	text string
	// The offset of the segment's text within its target.
	offset int
}

// The characters a target breaks its segments at, the same ones fuzzy.Find's
// scoring gives a bonus for following.
const segmentSeparators = "/-_ .\\"

func isSegmentSeparator(r rune) bool {
	return strings.ContainsRune(segmentSeparators, r)
}

func segmentsOf(target string) []segment {
	segments := []segment{}
	start := 0
	var last rune
	for i, r := range target {
		if i > start && (isSegmentSeparator(r) || (unicode.IsLower(last) && unicode.IsUpper(r))) {
			segments = append(segments, segment{text: target[start:i], offset: start})
		}
		if isSegmentSeparator(r) {
			start = i + utf8.RuneLen(r)
		}
		last = r
	}
	if start < len(target) {
		segments = append(segments, segment{text: target[start:], offset: start})
	}
	return segments
}
