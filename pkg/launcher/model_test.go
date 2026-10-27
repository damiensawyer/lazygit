package launcher

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func repo(fullName string) *models.GithubRepository {
	owner, name, _ := cutSlash(fullName)
	return &models.GithubRepository{
		FullName: fullName,
		Owner:    owner,
		Name:     name,
		HTMLURL:  "https://github.com/" + fullName,
	}
}

func cutSlash(s string) (string, string, bool) {
	for i := range s {
		if s[i] == '/' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func forkOf(fork string, parent string) *models.GithubRepository {
	r := repo(fork)
	r.IsFork = true
	r.Parent = repo(parent)
	return r
}

func newTestModel() *model {
	m := newModel(i18n.EnglishTranslationSet(), "github.com", "/home/me/scratch", "/home/me",
		[]string{"/home/me/scratch", "/home/me/code/lazygit", "/home/me/code/notes"})
	m.setToken("token")
	return m
}

func entryNames(entries []entry) []string {
	return lo.Map(entries, func(e entry, _ int) string { return e.text })
}

func actionLabels(actions []action) []string {
	return lo.Map(actions, func(a action, _ int) string { return a.label })
}

func TestModelFiltersYoursByScore(t *testing.T) {
	m := newTestModel()
	m.addYours("me", []*models.GithubRepository{repo("me/agree-sight"), repo("me/git"), repo("me/notes")})

	assert.Equal(t, []string{"me/agree-sight", "me/git", "me/notes"}, entryNames(m.entries[tabYours]))

	m.setQuery("git")
	assert.Equal(t, []string{"me/git", "me/agree-sight"}, entryNames(m.entries[tabYours]))
}

func TestModelExcludesCurrentDirectoryFromRecent(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, []string{"~/code/lazygit", "~/code/notes"}, entryNames(m.entries[tabRecent]))

	m.setRemotesOfRecent(map[string][]string{"/home/me/code/lazygit": {"me/lazygit", "jesseduffield/lazygit"}})
	assert.Equal(t, []string{"~/code/lazygit  me/lazygit  jesseduffield/lazygit", "~/code/notes"}, entryNames(m.entries[tabRecent]))

	m.setQuery("jes laz")
	assert.Equal(t, []string{"~/code/lazygit  me/lazygit  jesseduffield/lazygit"}, entryNames(m.entries[tabRecent]))
}

func TestModelSwitchesToGithubTabWhenNotLoggedIn(t *testing.T) {
	m := newModel(i18n.EnglishTranslationSet(), "github.com", "/scratch", "", nil)
	assert.Equal(t, tabYours, m.tab)
	m.setToken("")
	assert.Equal(t, tabGithub, m.tab)
	assert.False(t, m.yoursState.loading)

	// Unless the user has picked a tab before we found out.
	m = newModel(i18n.EnglishTranslationSet(), "github.com", "/scratch", "", nil)
	m.selectTab(tabRecent)
	m.setToken("")
	assert.Equal(t, tabRecent, m.tab)
}

func TestModelSearchesForKnownOwnersAsSuch(t *testing.T) {
	m := newTestModel()
	m.setQuery("jesseduffield laz")
	assert.Equal(t, "jesseduffield", m.plan.serverQuery)

	m.addStarred([]*models.GithubRepository{repo("jesseduffield/lazygit")})
	assert.Equal(t, "user:jesseduffield laz", m.plan.serverQuery)
}

func TestModelGithubEntries(t *testing.T) {
	results := &searchResults{
		repos: []*models.GithubRepository{repo("jesseduffield/lazydocker"), repo("jesseduffield/lazygit"), repo("other/thing")},
		pages: 1,
	}

	m := newTestModel()
	m.setQuery("jes laz")
	m.searches[m.plan.serverQuery] = results
	m.refresh()
	// The search results that match, in GitHub's order.
	assert.Equal(t, []string{"jesseduffield/lazydocker", "jesseduffield/lazygit"}, entryNames(m.entries[tabGithub]))

	m.setQuery("jesseduffield/lazygit")
	m.searches[m.plan.serverQuery] = results
	m.lookups["jesseduffield/lazygit"] = &lookup{repo: repo("jesseduffield/lazygit")}
	m.refresh()
	// The repository named comes first, and only once.
	assert.Equal(t, []string{"jesseduffield/lazygit"}, entryNames(m.entries[tabGithub]))

	m.setQuery("jesseduffield/laz")
	m.searches[m.plan.serverQuery] = results
	m.lookups["jesseduffield/laz"] = &lookup{}
	m.refresh()
	assert.Equal(t, []string{"jesseduffield/lazydocker", "jesseduffield/lazygit"}, entryNames(m.entries[tabGithub]))
}

func TestModelWantsMoreSearchResults(t *testing.T) {
	m := newTestModel()
	m.setQuery("laz")
	repos := lo.Times(50, func(i int) *models.GithubRepository {
		return repo("o/laz" + string(rune('a'+i%26)) + string(rune('a'+i/26)))
	})
	m.searches["laz"] = &searchResults{repos: repos, pages: 1, totalCount: 5000}
	m.refresh()

	assert.False(t, m.wantsMoreSearchResults(), "not looking at them")
	m.selectTab(tabGithub)
	assert.False(t, m.wantsMoreSearchResults(), "plenty left to look through")
	m.moveSelection(40)
	assert.True(t, m.wantsMoreSearchResults(), "near the end")

	m.setQuery("laz o/lazaa")
	m.searches["laz"].repos = repos
	m.refresh()
	assert.True(t, m.wantsMoreSearchResults(), "few matches")
}

func TestSearchResultsHasMorePages(t *testing.T) {
	assert.True(t, (&searchResults{pages: 1, totalCount: 5000}).hasMorePages())
	assert.False(t, (&searchResults{pages: 1, totalCount: 80}).hasMorePages())
	assert.False(t, (&searchResults{pages: 3, totalCount: 5000}).hasMorePages())
}

func TestModelActions(t *testing.T) {
	lazygit := repo("jesseduffield/lazygit")
	horcrux := repo("jesseduffield/horcrux")

	m := newTestModel()
	m.addYours("me", []*models.GithubRepository{forkOf("me/lazygit", "jesseduffield/lazygit"), repo("me/notes")})
	m.setRemotesOfRecent(map[string][]string{
		"/home/me/code/lazygit": {"me/lazygit", "jesseduffield/lazygit"},
		"/home/me/code/notes":   {"me/notes"},
	})

	assert.Equal(t, []string{
		"Open ~/code/lazygit",
		"Clone your fork me/lazygit",
		"Clone jesseduffield/lazygit",
		"Open in browser",
	}, actionLabels(m.actions(lazygit)))

	assert.Equal(t, []string{
		"Clone jesseduffield/horcrux",
		"Fork to me and clone",
		"Open in browser",
	}, actionLabels(m.actions(horcrux)))

	// No forking your own repository.
	assert.Equal(t, []string{
		"Open ~/code/notes",
		"Clone me/notes",
		"Open in browser",
	}, actionLabels(m.actions(m.yours[1])))

	assert.Equal(t, []string{
		"Open ~/code/lazygit",
		"Clone me/lazygit (upstream: jesseduffield/lazygit)",
		"Open in browser",
	}, actionLabels(m.actions(m.yours[0])))

	// Nor forking at all when not logged in.
	m.setToken("")
	assert.Equal(t, []string{"Clone jesseduffield/horcrux", "Open in browser"}, actionLabels(m.actions(horcrux)))
}

func TestModelPrefersOwnForkToOrganizations(t *testing.T) {
	m := newTestModel()
	m.addYours("me", []*models.GithubRepository{forkOf("acme/lazygit", "jesseduffield/lazygit")})
	assert.Equal(t, "acme/lazygit", m.forkOf(repo("jesseduffield/lazygit")).FullName)

	m.addYours("me", []*models.GithubRepository{forkOf("me/lazygit", "jesseduffield/lazygit")})
	assert.Equal(t, "me/lazygit", m.forkOf(repo("jesseduffield/lazygit")).FullName)
}

func TestDefaultCloneDestination(t *testing.T) {
	lazygit := repo("jesseduffield/lazygit")
	assert.Equal(t, "lazygit", defaultCloneDestination("/home/me/scratch", true, lazygit))
	assert.Equal(t, "lazygit", defaultCloneDestination("/home/me/lazygit", false, lazygit))
	assert.Equal(t, ".", defaultCloneDestination("/home/me/LazyGit", true, lazygit))
}

func TestDisplayPath(t *testing.T) {
	m := newTestModel()
	assert.Equal(t, "~", m.displayPath("/home/me"))
	assert.Equal(t, "~/code", m.displayPath("/home/me/code"))
	assert.Equal(t, "/home/meander", m.displayPath("/home/meander"))
	assert.Equal(t, "/tmp", m.displayPath("/tmp"))
}
