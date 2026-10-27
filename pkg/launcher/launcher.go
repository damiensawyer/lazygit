// Package launcher is the screen lazygit shows when it's started outside of a
// git repository: a picker for cloning one of the user's GitHub repositories,
// or any repository found by searching GitHub, for opening a recent repo, or
// for creating a new one in the current directory.
//
// It runs before, and separately from, the main gui, which needs a repo to
// show; it returns what the user chose for the app to carry out.
package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/i18n"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

type ChoiceKind int

const (
	ChoiceQuit ChoiceKind = iota
	// Create a repo in the current directory.
	ChoiceInit
	// Open an existing repo.
	ChoiceOpen
	// Clone a repository, forking it first if asked to.
	ChoiceClone
)

// Choice is what the user chose to do in the picker.
type Choice struct {
	Kind ChoiceKind
	// For ChoiceInit: the name of the initial branch, or empty for git's
	// default.
	InitialBranch string
	// For ChoiceOpen: the repo to open. For ChoiceClone: the directory to
	// clone into.
	Path string
	// For ChoiceClone: what to clone, or fork and clone.
	Repo *models.GithubRepository
	Fork bool

	// For talking to GitHub when carrying out a ChoiceClone.
	client *github_commands.Client
}

type Deps struct {
	Tr         *i18n.TranslationSet
	UserConfig *config.UserConfig
	Log        *logrus.Entry
	// The GitHub host to find repositories on, e.g. "github.com".
	Host        string
	RecentRepos []string
	OpenLink    func(url string) error
	// Finds the token to authenticate with; may block for a while, e.g. on
	// the system keyring.
	TokenForHost func(host string) string
}

// CanRun reports whether there's a terminal to show the picker in.
func CanRun() bool {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}
	tty, err := os.Open("/dev/tty")
	if err != nil {
		// Windows has no /dev/tty; the console is all there is.
		return os.PathSeparator == '\\'
	}
	tty.Close()
	return true
}

// Run shows the picker until the user has made a choice.
func Run(deps Deps) (Choice, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Choice{}, err
	}
	homeDir, _ := os.UserHomeDir()

	g, err := gocui.NewGui(gocui.NewGuiOpts{OutputMode: gocui.OutputTrue})
	if err != nil {
		return Choice{}, err
	}
	g.Highlight = true
	applyTheme(g, deps.UserConfig)
	g.SetColorSchemeChangeHandler(func(gocui.DetectedColorScheme) error {
		applyTheme(g, deps.UserConfig)
		return nil
	})

	self := newLauncher(deps, g, newModel(deps.Tr, deps.Host, cwd, homeDir, deps.RecentRepos))
	defer self.stop()

	g.SetManagerFunc(self.layout)
	self.setKeybindings()
	self.startLoading()

	err = g.MainLoop()
	g.Close()
	if err != nil && !errors.Is(err, gocui.ErrQuit) {
		return Choice{}, err
	}
	return self.choice, nil
}

// applyTheme sets the colors from the theme in the user config, the same way
// the gui does.
func applyTheme(g *gocui.Gui, userConfig *config.UserConfig) {
	theme.UpdateTheme(theme.ConfigForTerminal(&userConfig.Gui, g.DetectedColorScheme()))
	g.FgColor = theme.InactiveBorderColor
	g.SelFgColor = theme.ActiveBorderColor
	g.FrameColor = theme.InactiveBorderColor
	g.SelFrameColor = theme.ActiveBorderColor
}

type popupKind int

const (
	popupNone popupKind = iota
	popupActions
	popupCloneDestination
	popupInitialBranch
)

type launcher struct {
	deps  Deps
	g     *gocui.Gui
	model *model

	// Cancelled when the picker closes, which stops all background work.
	ctx    context.Context
	cancel context.CancelFunc

	// The search and lookup in progress, and the plan they're for, which
	// tells whether a change of query calls for another.
	searchCtx     context.Context
	cancelSearch  context.CancelFunc
	scheduledPlan string
	debounceTimer *time.Timer
	retryTimer    *time.Timer

	// Whether the list has the keyboard focus, rather than the search field.
	listFocused bool

	popup          popupKind
	actions        []action
	selectedAction int
	// The repository and action the clone destination is being asked for.
	pendingAction     *action
	promptInitialText string
	promptError       string

	choice Choice
}

const searchDebounceDelay = 300 * time.Millisecond

func newLauncher(deps Deps, g *gocui.Gui, m *model) *launcher {
	ctx, cancel := context.WithCancel(context.Background())
	self := &launcher{deps: deps, g: g, model: m, ctx: ctx, cancel: cancel}
	self.searchCtx, self.cancelSearch = context.WithCancel(ctx)
	return self
}

func (self *launcher) stop() {
	self.cancel()
	if self.debounceTimer != nil {
		self.debounceTimer.Stop()
	}
	if self.retryTimer != nil {
		self.retryTimer.Stop()
	}
}

func (self *launcher) client() *github_commands.Client {
	return github_commands.NewClient(github_commands.Repo{Host: self.model.host, Token: self.model.token})
}

// onUIThread runs f on the UI thread, unless the picker has closed by then.
func (self *launcher) onUIThread(f func()) {
	self.g.Update(func(*gocui.Gui) error {
		if self.ctx.Err() == nil {
			f()
		}
		return nil
	})
}

func (self *launcher) startLoading() {
	go utils.Safe(func() {
		token := self.deps.TokenForHost(self.model.host)
		self.onUIThread(func() {
			self.model.setToken(token)
			if token != "" {
				self.loadYours()
				self.loadStarred()
			}
			self.updateSearch()
		})
	})

	go utils.Safe(func() {
		remotes := findRemotesOfRepos(self.ctx, self.model.host, self.model.recent)
		self.onUIThread(func() { self.model.setRemotesOfRecent(remotes) })
	})
}

func (self *launcher) loadYours() {
	client := self.client()
	go utils.Safe(func() {
		onPage := func(login string) func([]*models.GithubRepository) {
			return func(repos []*models.GithubRepository) {
				self.onUIThread(func() {
					self.model.addYours(login, repos)
					self.updateSearch()
				})
			}
		}
		// Find out the login first, so the user's own repositories are known
		// as such the moment they arrive.
		login, err := github_commands.FetchUsername(self.ctx, self.model.host, self.model.token)
		if err == nil {
			_, err = client.ListViewerRepositories(self.ctx, github_commands.ViewerOwnedRepositories, maxListedRepositories, onPage(login))
		}
		if err == nil {
			_, err = client.ListViewerRepositories(self.ctx, github_commands.ViewerOtherRepositories, maxListedRepositories, onPage(login))
		}
		self.onUIThread(func() {
			if login != "" {
				self.model.login = login
			}
			self.model.yoursState = loadState{err: err}
			self.model.refresh()
		})
	})
}

func (self *launcher) loadStarred() {
	client := self.client()
	go utils.Safe(func() {
		err := client.ListStarredRepositories(self.ctx, maxListedRepositories, func(repos []*models.GithubRepository) {
			self.onUIThread(func() { self.model.addStarred(repos) })
		})
		self.onUIThread(func() {
			self.model.starredState = loadState{err: err}
			self.model.refresh()
		})
	})
}

// updateSearch starts searching GitHub for what the query now calls for, if
// it calls for something new: right away if the results are at hand, or
// otherwise once the user pauses typing, so as not to search for every
// keystroke and run into the search API's rate limit.
func (self *launcher) updateSearch() {
	plan := self.model.plan
	key := plan.serverQuery + "\x00" + self.model.lookupKey()
	if key == self.scheduledPlan {
		// The same search; but the user may have scrolled far enough to want
		// more of its results.
		self.fetchSearchResults()
		return
	}
	self.scheduledPlan = key

	if self.debounceTimer != nil {
		self.debounceTimer.Stop()
	}
	self.cancelSearch()
	self.searchCtx, self.cancelSearch = context.WithCancel(self.ctx)
	for _, results := range self.model.searches {
		results.loading = false
	}
	for lookupKey, lookup := range self.model.lookups {
		if lookup.loading {
			delete(self.model.lookups, lookupKey)
		}
	}
	// Give a search that failed another go.
	if results := self.model.currentSearch(); results != nil && results.err != nil {
		delete(self.model.searches, plan.serverQuery)
	}

	if self.model.currentSearch() != nil || (plan.serverQuery == "" && plan.directOwner == "") {
		self.fetchSearchResults()
		return
	}
	self.debounceTimer = time.AfterFunc(searchDebounceDelay, utils.SafeFunc(func() {
		self.onUIThread(func() {
			if key == self.scheduledPlan {
				self.fetchSearchResults()
			}
		})
	}))
}

// fetchSearchResults fetches whatever the current search still needs: its
// first page of results, or another if the user wants more, and the
// repository the query names, if any.
func (self *launcher) fetchSearchResults() {
	m := self.model
	if !m.tokenResolved {
		return
	}

	if key := m.lookupKey(); key != "" && m.lookups[key] == nil {
		m.lookups[key] = &lookup{loading: true}
		self.fetchLookup(key, m.plan.directOwner, m.plan.directName)
	}

	if m.plan.serverQuery == "" {
		return
	}
	if wait := time.Until(m.rateLimitedUntil); wait > 0 {
		self.retryAfter(wait)
		return
	}
	results := m.currentSearch()
	if results == nil {
		results = &searchResults{}
		m.searches[m.plan.serverQuery] = results
	}
	if results.loading || results.err != nil {
		return
	}
	if results.pages == 0 || (results.hasMorePages() && m.wantsMoreSearchResults()) {
		self.fetchSearchPage(m.plan.serverQuery, results)
	}
}

func (self *launcher) fetchLookup(key string, owner string, name string) {
	ctx := self.searchCtx
	client := self.client()
	go utils.Safe(func() {
		requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		repo, err := client.GetRepository(requestCtx, owner, name)
		self.onUIThread(func() {
			// A lookup that the query has moved on from was reset when it
			// was cancelled.
			if ctx.Err() != nil {
				return
			}
			if err != nil && !github_commands.IsNotFound(err) {
				self.deps.Log.WithError(err).Warn("looking up repository failed")
			}
			self.model.lookups[key] = &lookup{repo: repo}
			self.model.refresh()
		})
	})
}

func (self *launcher) fetchSearchPage(query string, results *searchResults) {
	results.loading = true
	page := results.pages + 1
	ctx := self.searchCtx
	client := self.client()
	go utils.Safe(func() {
		requestCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		result, err := client.SearchRepositories(requestCtx, query, page, searchPageSize)
		self.onUIThread(func() {
			// A search that the query has moved on from was reset when it
			// was cancelled.
			if ctx.Err() != nil {
				return
			}
			results.loading = false
			if err != nil {
				if reset, limited := github_commands.RateLimitReset(err); limited {
					self.model.rateLimitedUntil = reset
					self.retryAfter(time.Until(reset))
				} else {
					results.err = err
				}
				self.model.refresh()
				return
			}
			results.repos = appendNew(results.repos, result.Repositories)
			results.totalCount = result.TotalCount
			results.incomplete = result.Incomplete
			results.pages = page
			self.model.refresh()
			self.fetchSearchResults()
		})
	})
}

func (self *launcher) retryAfter(wait time.Duration) {
	if self.retryTimer != nil {
		self.retryTimer.Stop()
	}
	self.retryTimer = time.AfterFunc(wait+time.Second, utils.SafeFunc(func() {
		self.onUIThread(self.fetchSearchResults)
	}))
}

func (self *launcher) quit() error {
	self.choice = Choice{Kind: ChoiceQuit}
	return gocui.ErrQuit
}

func (self *launcher) choose(choice Choice) error {
	choice.client = self.client()
	self.choice = choice
	return gocui.ErrQuit
}

// chooseEntry acts on the selected row: opens a recent repo, or offers what
// can be done with a repository.
func (self *launcher) chooseEntry() error {
	e := self.model.selectedEntry()
	if e == nil {
		return nil
	}
	if e.repo == nil {
		return self.choose(Choice{Kind: ChoiceOpen, Path: e.path})
	}
	self.actions = self.model.actions(e.repo)
	self.selectedAction = 0
	self.popup = popupActions
	return nil
}

func (self *launcher) chooseAction() error {
	a := self.actions[self.selectedAction]
	switch a.kind {
	case actionOpen:
		return self.choose(Choice{Kind: ChoiceOpen, Path: a.path})
	case actionOpenInBrowser:
		self.popup = popupNone
		return self.deps.OpenLink(a.repo.HTMLURL)
	case actionClone, actionForkAndClone:
		self.pendingAction = &a
		self.openPrompt(popupCloneDestination,
			defaultCloneDestination(self.model.cwd, isEmptyDir(self.model.cwd), a.repo))
	}
	return nil
}

func (self *launcher) confirmPrompt(text string) error {
	switch self.popup {
	case popupCloneDestination:
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		dest := text
		if !filepath.IsAbs(dest) {
			dest = filepath.Join(self.model.cwd, dest)
		}
		if _, err := os.Stat(dest); err == nil && !isEmptyDir(dest) {
			self.promptError = utils.ResolvePlaceholderString(self.deps.Tr.LauncherDestinationNotEmpty,
				map[string]string{"path": self.model.displayPath(dest)})
			return nil
		}
		return self.choose(Choice{
			Kind: ChoiceClone,
			Path: dest,
			Repo: self.pendingAction.repo,
			Fork: self.pendingAction.kind == actionForkAndClone,
		})
	case popupInitialBranch:
		return self.choose(Choice{Kind: ChoiceInit, InitialBranch: strings.TrimSpace(text)})
	case popupNone, popupActions:
	}
	return nil
}

func isEmptyDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}
