package launcher

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

type tab int

const (
	tabYours tab = iota
	tabStarred
	tabGithub
	tabRecent
	tabCount
)

const (
	// How many of their own repositories, and of their organizations', and
	// how many starred ones, we list for a user at most.
	maxListedRepositories = 1000
	searchPageSize        = 100
	// So at most 300 results per search; a search that needs more than that
	// to find something needs more words instead.
	maxSearchPages = 3
)

// loadState is how far loading one of the lists has got.
type loadState struct {
	loading bool
	err     error
}

type searchResults struct {
	repos      []*models.GithubRepository
	totalCount int
	incomplete bool
	pages      int
	loading    bool
	err        error
}

func (self *searchResults) hasMorePages() bool {
	// The search API serves only the first thousand matches.
	return self.pages < maxSearchPages && self.pages*searchPageSize < min(self.totalCount, 1000)
}

// lookup is looking up a repository the user named by "owner/name" or URL.
type lookup struct {
	loading bool
	// nil if there's no such repository, or the lookup failed.
	repo *models.GithubRepository
}

// entry is a row of one of the lists.
type entry struct {
	// nil for a recent repo.
	repo *models.GithubRepository
	// The directory of a recent repo.
	path string
	// The text the query was matched against, and the byte offsets in it of
	// the characters it matched.
	text      string
	positions []int
}

type actionKind int

const (
	actionOpen actionKind = iota
	actionClone
	actionForkAndClone
	actionOpenInBrowser
)

// action is something to do with the repository the user chose.
type action struct {
	kind  actionKind
	label string
	// What to clone or fork, or open in the browser.
	repo *models.GithubRepository
	// The directory to open.
	path string
}

// model is the state of the picker. Everything about it happens on the UI
// thread; the background work that fills it in hands its results there.
type model struct {
	tr      *i18n.TranslationSet
	host    string
	cwd     string
	homeDir string

	// Whether we've found out yet if the user is logged in, which may take a
	// while if the system keyring is slow to answer.
	tokenResolved bool
	token         string
	login         string

	query string
	plan  searchPlan
	tab   tab
	// Whether the user has picked a tab themselves, after which we leave the
	// choice to them.
	tabChosen bool
	selection [tabCount]int

	yours        []*models.GithubRepository
	yoursState   loadState
	starred      []*models.GithubRepository
	starredState loadState
	recent       []string
	// The GitHub repositories that the remotes of each recent repo point to.
	remotesOfRecent map[string][]string
	// The reverse: the recent repos that have a remote pointing to a GitHub
	// repository, by its lowercased full name.
	clonesOf map[string][]string
	// The user's forks, by the lowercased full name of what they're forks of.
	forksOf map[string]*models.GithubRepository
	// Lowercased logins of the users and organizations we've come across.
	knownOwners map[string]bool

	searches map[string]*searchResults
	lookups  map[string]*lookup
	// When the search API's rate limit, which we've hit, resets.
	rateLimitedUntil time.Time

	entries [tabCount][]entry
}

func newModel(tr *i18n.TranslationSet, host string, cwd string, homeDir string, recent []string) *model {
	self := &model{
		tr:              tr,
		host:            host,
		cwd:             cwd,
		homeDir:         homeDir,
		recent:          lo.Filter(recent, func(path string, _ int) bool { return path != cwd }),
		remotesOfRecent: map[string][]string{},
		clonesOf:        map[string][]string{},
		forksOf:         map[string]*models.GithubRepository{},
		knownOwners:     map[string]bool{},
		searches:        map[string]*searchResults{},
		lookups:         map[string]*lookup{},
		yoursState:      loadState{loading: true},
		starredState:    loadState{loading: true},
	}
	self.refresh()
	return self
}

func (self *model) setToken(token string) {
	self.tokenResolved = true
	self.token = token
	if token == "" {
		self.yoursState.loading = false
		self.starredState.loading = false
		if !self.tabChosen {
			self.tab = tabGithub
		}
	}
	self.refresh()
}

func (self *model) setQuery(query string) {
	self.query = query
	self.selection = [tabCount]int{}
	self.refresh()
}

func (self *model) addYours(login string, repos []*models.GithubRepository) {
	self.login = login
	self.yours = appendNew(self.yours, repos)
	self.refresh()
}

func (self *model) addStarred(repos []*models.GithubRepository) {
	self.starred = appendNew(self.starred, repos)
	self.refresh()
}

func (self *model) setRemotesOfRecent(remotesOfRecent map[string][]string) {
	self.remotesOfRecent = remotesOfRecent
	self.clonesOf = map[string][]string{}
	// In the order of the recent repos, so the most recently used comes first.
	for _, path := range self.recent {
		for _, fullName := range remotesOfRecent[path] {
			key := strings.ToLower(fullName)
			self.clonesOf[key] = append(self.clonesOf[key], path)
		}
	}
	self.refresh()
}

// appendNew appends the repositories of repos that aren't in list yet.
func appendNew(list []*models.GithubRepository, repos []*models.GithubRepository) []*models.GithubRepository {
	seen := lo.SliceToMap(list, func(repo *models.GithubRepository) (string, bool) { return repo.FullName, true })
	for _, repo := range repos {
		if !seen[repo.FullName] {
			seen[repo.FullName] = true
			list = append(list, repo)
		}
	}
	return list
}

func (self *model) selectTab(t tab) {
	self.tab = (t + tabCount) % tabCount
	self.tabChosen = true
}

func (self *model) moveSelection(delta int) {
	entries := self.entries[self.tab]
	if len(entries) == 0 {
		return
	}
	self.selection[self.tab] = max(0, min(len(entries)-1, self.selection[self.tab]+delta))
}

func (self *model) selectedEntry() *entry {
	entries := self.entries[self.tab]
	if len(entries) == 0 {
		return nil
	}
	return &entries[self.selection[self.tab]]
}

// refresh recomputes everything derived from the loaded data and the query.
func (self *model) refresh() {
	self.forksOf = map[string]*models.GithubRepository{}
	self.knownOwners = map[string]bool{}
	if self.login != "" {
		self.knownOwners[strings.ToLower(self.login)] = true
	}
	for _, repo := range self.yours {
		self.knownOwners[strings.ToLower(repo.Owner)] = true
		if repo.IsFork && repo.Parent != nil {
			key := strings.ToLower(repo.Parent.FullName)
			// Prefer the user's own fork to one of their organization's.
			if existing := self.forksOf[key]; existing == nil || existing.Owner != self.login {
				self.forksOf[key] = repo
			}
		}
	}
	for _, repo := range self.starred {
		self.knownOwners[strings.ToLower(repo.Owner)] = true
	}

	self.plan = planSearch(self.host, self.query, func(word string) bool {
		return self.knownOwners[strings.ToLower(word)]
	})

	self.entries[tabYours] = self.repoEntries(self.yours, true)
	self.entries[tabStarred] = self.repoEntries(self.starred, true)
	self.entries[tabGithub] = self.githubEntries()
	self.entries[tabRecent] = self.recentEntries()

	for t := range tabCount {
		self.selection[t] = max(0, min(len(self.entries[t])-1, self.selection[t]))
	}
}

func (self *model) repoEntries(repos []*models.GithubRepository, sortByScore bool) []entry {
	names := lo.Map(repos, func(repo *models.GithubRepository, _ int) string { return repo.FullName })
	return lo.Map(fuzzyFilter(self.plan.localWords, names, sortByScore), func(m match, _ int) entry {
		return entry{repo: repos[m.index], text: names[m.index], positions: m.positions}
	})
}

func (self *model) githubEntries() []entry {
	entries := []entry{}
	var direct *models.GithubRepository
	if lookup := self.currentLookup(); lookup != nil && lookup.repo != nil {
		direct = lookup.repo
		entries = append(entries, entry{repo: direct, text: direct.FullName})
	}
	if results := self.currentSearch(); results != nil {
		// GitHub's order is the best measure of relevance we have, so keep it.
		for _, e := range self.repoEntries(results.repos, false) {
			if direct == nil || e.repo.FullName != direct.FullName {
				entries = append(entries, e)
			}
		}
	}
	return entries
}

func (self *model) recentEntries() []entry {
	texts := lo.Map(self.recent, func(path string, _ int) string {
		return strings.Join(append([]string{self.displayPath(path)}, self.remotesOfRecent[path]...), "  ")
	})
	return lo.Map(fuzzyFilter(self.plan.localWords, texts, self.query != ""), func(m match, _ int) entry {
		return entry{path: self.recent[m.index], text: texts[m.index], positions: m.positions}
	})
}

func (self *model) currentSearch() *searchResults {
	if self.plan.serverQuery == "" {
		return nil
	}
	return self.searches[self.plan.serverQuery]
}

func (self *model) lookupKey() string {
	if self.plan.directOwner == "" {
		return ""
	}
	return strings.ToLower(self.plan.directOwner + "/" + self.plan.directName)
}

func (self *model) currentLookup() *lookup {
	return self.lookups[self.lookupKey()]
}

// wantsMoreSearchResults reports whether to fetch another page of the current
// search: when the user is looking at its results and has run out of matches
// to look through, or is about to.
func (self *model) wantsMoreSearchResults() bool {
	if self.tab != tabGithub {
		return false
	}
	count := len(self.entries[tabGithub])
	return count < 30 || self.selection[tabGithub] >= count-10
}

func (self *model) isOwnRepo(repo *models.GithubRepository) bool {
	return self.login != "" && strings.EqualFold(repo.Owner, self.login)
}

// forkOf returns the user's fork of repo, or nil if they haven't got one, or
// we don't know of it (yet).
func (self *model) forkOf(repo *models.GithubRepository) *models.GithubRepository {
	return self.forksOf[strings.ToLower(repo.FullName)]
}

// clonesOfRepo returns the recent repos that have a remote pointing to repo.
func (self *model) clonesOfRepo(repo *models.GithubRepository) []string {
	return self.clonesOf[strings.ToLower(repo.FullName)]
}

// actions returns what the user can do with repo, the likeliest first.
func (self *model) actions(repo *models.GithubRepository) []action {
	actions := []action{}
	fork := self.forkOf(repo)

	opened := map[string]bool{}
	addOpens := func(paths []string) {
		for _, path := range paths {
			if !opened[path] {
				opened[path] = true
				actions = append(actions, action{
					kind:  actionOpen,
					label: utils.ResolvePlaceholderString(self.tr.LauncherOpenClone, map[string]string{"path": self.displayPath(path)}),
					path:  path,
				})
			}
		}
	}
	if fork != nil {
		addOpens(self.clonesOfRepo(fork))
	}
	addOpens(self.clonesOfRepo(repo))

	if fork != nil {
		actions = append(actions, action{
			kind:  actionClone,
			label: utils.ResolvePlaceholderString(self.tr.LauncherCloneYourFork, map[string]string{"fork": fork.FullName}),
			repo:  fork,
		})
	}
	cloneLabel := utils.ResolvePlaceholderString(self.tr.LauncherClone, map[string]string{"repo": repo.FullName})
	if repo.Parent != nil {
		cloneLabel = utils.ResolvePlaceholderString(self.tr.LauncherCloneFork, map[string]string{
			"repo":   repo.FullName,
			"parent": repo.Parent.FullName,
		})
	}
	actions = append(actions, action{kind: actionClone, label: cloneLabel, repo: repo})
	if fork == nil && self.token != "" && self.login != "" && !self.isOwnRepo(repo) {
		actions = append(actions, action{
			kind:  actionForkAndClone,
			label: utils.ResolvePlaceholderString(self.tr.LauncherForkAndClone, map[string]string{"user": self.login}),
			repo:  repo,
		})
	}
	actions = append(actions, action{kind: actionOpenInBrowser, label: self.tr.LauncherOpenInBrowser, repo: repo})
	return actions
}

// defaultCloneDestination is where to suggest cloning repo into: a directory
// named after it, or the current one if that is an empty directory named
// after it, which the user presumably made for it.
func defaultCloneDestination(cwd string, cwdIsEmpty bool, repo *models.GithubRepository) string {
	if cwdIsEmpty && strings.EqualFold(filepath.Base(cwd), repo.Name) {
		return "."
	}
	return repo.Name
}

func (self *model) displayPath(path string) string {
	if self.homeDir != "" {
		if rest, found := strings.CutPrefix(path, self.homeDir); found && (rest == "" || rest[0] == filepath.Separator) {
			return "~" + rest
		}
	}
	return path
}
