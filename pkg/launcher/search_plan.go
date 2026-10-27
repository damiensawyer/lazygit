package launcher

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
)

// GitHub's repository search matches whole words of repository names (and
// descriptions), so a query of abbreviations like "jes laz" finds nothing
// useful, while "laz" alone happens to find lazygit. We therefore send GitHub
// only the most distinctive word of what the user typed, and narrow its
// results down locally by fuzzy-matching all of the words against their
// names.

// The fewest characters a word must have for us to search GitHub for it.
const minSearchWordLength = 2

type searchPlan struct {
	// What to send to the search API; empty for no search. Identical plans
	// share their results.
	serverQuery string
	// The words to fuzzy-match against the repositories that come back.
	localWords []string
	// Set if the query names a single repository, "owner/name" or its URL, to
	// look up directly; the search API doesn't find a repository by that.
	directOwner string
	directName  string
}

// planSearch works out how to search GitHub for a query. isKnownOwner reports
// whether a word is the login of a user or organization we've seen, which we
// then search for as such rather than as a word of the repository name: "user:
// jesseduffield laz" finds jesseduffield's lazygit, where "jesseduffield laz"
// doesn't.
func planSearch(host string, query string, isKnownOwner func(string) bool) searchPlan {
	plan := searchPlan{}
	if owner, name, ok := github_commands.ParseRepositoryName(host, query); ok {
		plan.directOwner, plan.directName = owner, name
		// Search too, since a partially typed name ("jesseduffield/laz")
		// parses as one but won't be found.
		query = owner + " " + name
	}

	qualifiers := []string{}
	keywords := []string{}
	for _, word := range strings.Fields(query) {
		switch {
		case strings.Contains(word, ":"):
			qualifiers = append(qualifiers, word)
		case isKnownOwner(word):
			qualifiers = append(qualifiers, "user:"+word)
			// It's matched against "owner/name" locally like the other words,
			// which keeps a repository it's only a part of the name of out.
			plan.localWords = append(plan.localWords, word)
		default:
			keywords = append(keywords, word)
			plan.localWords = append(plan.localWords, word)
		}
	}

	terms := qualifiers
	if longest := longestWord(keywords); len(longest) >= minSearchWordLength {
		terms = append(terms, longest)
	}
	plan.serverQuery = strings.Join(terms, " ")
	return plan
}

// longestWord returns the longest of words, or of those tied for it, the last:
// people tend to type an owner before a repository's name, and the name is the
// more distinctive of the two.
func longestWord(words []string) string {
	longest := ""
	for _, word := range words {
		if len(word) >= len(longest) {
			longest = word
		}
	}
	return longest
}
