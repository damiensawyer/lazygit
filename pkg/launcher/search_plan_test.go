package launcher

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestPlanSearch(t *testing.T) {
	knownOwners := []string{"jesseduffield", "me"}
	isKnownOwner := func(word string) bool { return lo.Contains(knownOwners, word) }

	scenarios := []struct {
		query    string
		expected searchPlan
	}{
		{
			query:    "",
			expected: searchPlan{},
		},
		{
			query:    "  ",
			expected: searchPlan{},
		},
		{
			query:    "l",
			expected: searchPlan{localWords: []string{"l"}},
		},
		{
			query:    "laz",
			expected: searchPlan{serverQuery: "laz", localWords: []string{"laz"}},
		},
		{
			query:    "jes laz",
			expected: searchPlan{serverQuery: "laz", localWords: []string{"jes", "laz"}},
		},
		{
			query:    "jes lazyg",
			expected: searchPlan{serverQuery: "lazyg", localWords: []string{"jes", "lazyg"}},
		},
		{
			query:    "jesseduffield laz",
			expected: searchPlan{serverQuery: "user:jesseduffield laz", localWords: []string{"jesseduffield", "laz"}},
		},
		{
			query:    "jesseduffield",
			expected: searchPlan{serverQuery: "user:jesseduffield", localWords: []string{"jesseduffield"}},
		},
		{
			query:    "react language:typescript",
			expected: searchPlan{serverQuery: "language:typescript react", localWords: []string{"react"}},
		},
		{
			query: "jesseduffield/lazygit",
			expected: searchPlan{
				serverQuery: "user:jesseduffield lazygit",
				localWords:  []string{"jesseduffield", "lazygit"},
				directOwner: "jesseduffield",
				directName:  "lazygit",
			},
		},
		{
			query: "https://github.com/facebook/react/issues",
			expected: searchPlan{
				serverQuery: "facebook",
				localWords:  []string{"facebook", "react"},
				directOwner: "facebook",
				directName:  "react",
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.query, func(t *testing.T) {
			assert.Equal(t, s.expected, planSearch("github.com", s.query, isKnownOwner))
		})
	}
}
