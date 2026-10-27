package launcher

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/theme"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

const (
	headerView = "header"
	searchView = "search"
	listView   = "list"
	detailView = "detail"
	footerView = "footer"
	menuView   = "menu"
	promptView = "prompt"
)

// The narrowest terminal that we show the details of the selected repository
// next to the list in.
const minWidthForDetails = 100

var (
	matchStyle     = style.FgYellow.SetBold()
	dimStyle       = style.New().SetDim()
	yourForkStyle  = style.FgCyan
	clonedStyle    = style.FgGreen
	errorStyle     = style.FgRed
	yourForkBadge  = "⑂"
	clonedBadge    = "✓"
	starsIndicator = "★"
)

func (self *launcher) setKeybindings() {
	type binding struct {
		view    string
		key     gocui.Key
		handler func() error
	}
	bindings := []binding{
		{"", gocui.NewKeyStrMod("c", gocui.ModCtrl), self.quit},

		{searchView, gocui.NewKeyName(gocui.KeyTab), self.focusList},
		{searchView, gocui.NewKeyName(gocui.KeyBacktab), self.focusList},
		{searchView, gocui.NewKeyName(gocui.KeyEsc), self.clearQueryOrQuit},

		{listView, gocui.NewKeyRune('k'), func() error { return self.moveSelection(-1) }},
		{listView, gocui.NewKeyRune('j'), func() error { return self.moveSelection(1) }},
		{listView, gocui.NewKeyName(gocui.KeyHome), func() error { return self.moveSelection(-len(self.model.entries[self.model.tab])) }},
		{listView, gocui.NewKeyName(gocui.KeyEnd), func() error { return self.moveSelection(len(self.model.entries[self.model.tab])) }},
		{listView, gocui.NewKeyRune('['), func() error { return self.selectTab(self.model.tab - 1) }},
		{listView, gocui.NewKeyRune(']'), func() error { return self.selectTab(self.model.tab + 1) }},
		{listView, gocui.NewKeyRune('o'), self.openSelectedInBrowser},
		{listView, gocui.NewKeyRune('q'), self.quit},
		{listView, gocui.NewKeyName(gocui.KeyTab), self.focusSearch},
		{listView, gocui.NewKeyName(gocui.KeyBacktab), self.focusSearch},
		{listView, gocui.NewKeyName(gocui.KeyEsc), self.focusSearch},
		{listView, gocui.NewKeyRune('/'), self.focusSearch},
	}
	// The list can be navigated while typing in the search field too.
	for _, view := range []string{searchView, listView} {
		bindings = append(bindings,
			binding{view, gocui.NewKeyName(gocui.KeyArrowUp), func() error { return self.moveSelection(-1) }},
			binding{view, gocui.NewKeyName(gocui.KeyArrowDown), func() error { return self.moveSelection(1) }},
			binding{view, gocui.NewKeyName(gocui.KeyPgup), func() error { return self.moveSelection(-self.listPageSize()) }},
			binding{view, gocui.NewKeyName(gocui.KeyPgdn), func() error { return self.moveSelection(self.listPageSize()) }},
			binding{view, gocui.NewKeyName(gocui.KeyEnter), self.chooseEntry},
			binding{view, gocui.NewKeyStrMod("n", gocui.ModCtrl), func() error {
				self.openPrompt(popupInitialBranch, "")
				return nil
			}},
			binding{view, gocui.NewKeyStrMod("o", gocui.ModCtrl), self.openSelectedInBrowser},
		)
	}
	bindings = append(bindings, []binding{
		{menuView, gocui.NewKeyName(gocui.KeyArrowUp), func() error { return self.moveActionSelection(-1) }},
		{menuView, gocui.NewKeyRune('k'), func() error { return self.moveActionSelection(-1) }},
		{menuView, gocui.NewKeyName(gocui.KeyArrowDown), func() error { return self.moveActionSelection(1) }},
		{menuView, gocui.NewKeyRune('j'), func() error { return self.moveActionSelection(1) }},
		{menuView, gocui.NewKeyName(gocui.KeyEnter), self.chooseAction},
		{menuView, gocui.NewKeyName(gocui.KeyEsc), self.closePopup},

		{promptView, gocui.NewKeyName(gocui.KeyEnter), func() error {
			v, err := self.g.View(promptView)
			if err != nil {
				return err
			}
			return self.confirmPrompt(v.TextArea.GetContent())
		}},
		{promptView, gocui.NewKeyName(gocui.KeyEsc), self.closePopup},
	}...)
	for _, b := range bindings {
		self.g.SetKeybinding(b.view, b.key, func(*gocui.Gui, *gocui.View) error { return b.handler() })
	}
}

func (self *launcher) moveSelection(delta int) error {
	self.model.moveSelection(delta)
	self.fetchSearchResults()
	return nil
}

func (self *launcher) selectTab(t tab) error {
	self.model.selectTab(t)
	self.fetchSearchResults()
	return nil
}

func (self *launcher) focusList() error {
	self.listFocused = true
	return nil
}

func (self *launcher) focusSearch() error {
	self.listFocused = false
	return nil
}

func (self *launcher) listPageSize() int {
	if v, err := self.g.View(listView); err == nil {
		return max(1, v.InnerHeight())
	}
	return 10
}

func (self *launcher) clearQueryOrQuit() error {
	if self.model.query == "" {
		return self.quit()
	}
	v, err := self.g.View(searchView)
	if err != nil {
		return err
	}
	v.TextArea.Clear()
	v.RenderTextArea()
	self.onQueryChanged("")
	return nil
}

func (self *launcher) onQueryChanged(query string) {
	if query == self.model.query {
		return
	}
	self.model.setQuery(query)
	self.updateSearch()
}

func (self *launcher) openSelectedInBrowser() error {
	if e := self.model.selectedEntry(); e != nil && e.repo != nil {
		return self.deps.OpenLink(e.repo.HTMLURL)
	}
	return nil
}

func (self *launcher) moveActionSelection(delta int) error {
	self.selectedAction = max(0, min(len(self.actions)-1, self.selectedAction+delta))
	return nil
}

func (self *launcher) openPrompt(kind popupKind, initialText string) {
	self.popup = kind
	self.promptError = ""
	_ = self.g.DeleteView(promptView)
	self.promptInitialText = initialText
}

func (self *launcher) closePopup() error {
	if self.popup == popupCloneDestination {
		self.popup = popupActions
	} else {
		self.popup = popupNone
	}
	return nil
}

func (self *launcher) layout(g *gocui.Gui) error {
	width, height := g.Size()

	header, err := self.setView(headerView, -1, -1, width, 1, false)
	if err != nil {
		return err
	}
	header.SetContent(self.renderHeader(width))

	search, err := self.setView(searchView, 0, 1, width-1, 3, true)
	if err != nil {
		return err
	}
	// The list's title is mostly taken up by its tabs, so say how its loading
	// is going here, where it's in view of what the user types anyway.
	search.Subtitle = self.listStatus()
	if search.Title == "" {
		search.Title = self.deps.Tr.LauncherSearchTitle
		search.Editable = true
		search.Editor = gocui.EditorFunc(func(v *gocui.View, key gocui.Key) bool {
			// Switch lists with [ and ] here too, as in the list. Neither can
			// be part of a repository's name, nor of a search, so they're
			// free to take from the search field.
			if !g.IsPasting {
				switch {
				case key.Equals(gocui.NewKeyRune('[')):
					_ = self.selectTab(self.model.tab - 1)
					return true
				case key.Equals(gocui.NewKeyRune(']')):
					_ = self.selectTab(self.model.tab + 1)
					return true
				}
			}
			matched := gocui.SimpleEditor(v, key)
			self.onQueryChanged(v.TextArea.GetContent())
			return matched
		})
	}

	listWidth := width
	if width >= minWidthForDetails {
		listWidth = width * 11 / 20
	}
	list, err := self.setView(listView, 0, 4, listWidth-1, height-2, true)
	if err != nil {
		return err
	}
	self.renderList(list)

	if listWidth < width {
		detail, err := self.setView(detailView, listWidth, 4, width-1, height-2, true)
		if err != nil {
			return err
		}
		detail.Wrap = true
		detail.SetContent(self.renderDetail())
	} else {
		_ = g.DeleteView(detailView)
	}

	footer, err := self.setView(footerView, -1, height-2, width, height, false)
	if err != nil {
		return err
	}
	footer.SetContent(self.renderFooter())

	currentView := searchView
	if self.listFocused {
		currentView = listView
	}
	switch self.popup {
	case popupActions:
		currentView = menuView
		_ = g.DeleteView(promptView)
		if err := self.layoutMenu(width, height); err != nil {
			return err
		}
	case popupCloneDestination, popupInitialBranch:
		currentView = promptView
		_ = g.DeleteView(menuView)
		if err := self.layoutPrompt(width, height); err != nil {
			return err
		}
	default:
		_ = g.DeleteView(menuView)
		_ = g.DeleteView(promptView)
	}
	if _, err := g.SetCurrentView(currentView); err != nil {
		return err
	}
	if _, err := g.SetViewOnTop(currentView); err != nil {
		return err
	}
	g.Cursor = currentView == searchView || currentView == promptView
	return nil
}

// setView creates or repositions a view.
func (self *launcher) setView(name string, x0, y0, x1, y1 int, frame bool) (*gocui.View, error) {
	v, err := self.g.SetView(name, x0, y0, x1, y1, 0)
	if err != nil && !errors.Is(err, gocui.ErrUnknownView) {
		return nil, err
	}
	if errors.Is(err, gocui.ErrUnknownView) {
		v.Frame = frame
	}
	// The theme can change while we're running, when the terminal's
	// background does.
	v.SelFgColor = theme.ActiveBorderColor
	v.SelBgColor = theme.GocuiSelectedLineBgColor
	v.InactiveViewSelBgColor = theme.GocuiInactiveViewSelectedLineBgColor
	return v, nil
}

func (self *launcher) renderHeader(width int) string {
	tr := self.deps.Tr
	m := self.model
	left := utils.ResolvePlaceholderString(tr.LauncherNotARepository, map[string]string{"dir": m.displayPath(m.cwd)}) +
		" " + utils.ResolvePlaceholderString(tr.LauncherHint, map[string]string{"key": "ctrl+n"})
	right := ""
	switch {
	case m.login != "":
		right = utils.ResolvePlaceholderString(tr.LauncherLoggedInAs, map[string]string{"user": m.login, "host": m.host})
	case m.tokenResolved && m.token == "":
		right = utils.ResolvePlaceholderString(tr.LauncherNotLoggedInShort, map[string]string{"host": m.host})
	}
	left = utils.TruncateWithEllipsis(left, max(0, width-utils.StringWidth(right)-3))
	padding := max(1, width-utils.StringWidth(left)-utils.StringWidth(right)-1)
	return " " + left + strings.Repeat(" ", padding-1) + dimStyle.Sprint(right)
}

func (self *launcher) renderFooter() string {
	tr := self.deps.Tr
	keys := [][2]string{
		{"enter", tr.LauncherKeyChoose},
		{"tab", tr.LauncherKeyGoToList},
		{"[/]", tr.LauncherKeySwitchList},
		{"ctrl+n", tr.LauncherKeyNewRepo},
		{"ctrl+o", tr.LauncherKeyOpenInBrowser},
		{"esc", tr.LauncherKeyQuit},
	}
	if self.listFocused {
		keys = [][2]string{
			{"enter", tr.LauncherKeyChoose},
			{"tab", tr.LauncherKeyGoToSearch},
			{"[/]", tr.LauncherKeySwitchList},
			{"ctrl+n", tr.LauncherKeyNewRepo},
			{"o", tr.LauncherKeyOpenInBrowser},
			{"q", tr.LauncherKeyQuit},
		}
	}
	if self.popup != popupNone {
		keys = [][2]string{{"enter", tr.LauncherKeyConfirm}, {"esc", tr.LauncherKeyCancel}}
	}
	return " " + strings.Join(lo.Map(keys, func(key [2]string, _ int) string {
		return style.FgBlue.Sprint(key[0]) + ": " + key[1]
	}), "  ")
}

func (self *launcher) tabTitles() []string {
	tr := self.deps.Tr
	m := self.model
	titles := []string{tr.LauncherYoursTab, tr.LauncherStarredTab, tr.LauncherGithubTab, tr.LauncherRecentTab}
	for t := range tabCount {
		if t == tabGithub && m.currentSearch() == nil && m.currentLookup() == nil {
			continue
		}
		titles[t] += fmt.Sprintf(" (%d)", len(m.entries[t]))
	}
	return titles
}

// listStatus returns what to say about the list of the current tab while it
// loads, or what's wrong with it.
func (self *launcher) listStatus() string {
	tr := self.deps.Tr
	m := self.model
	switch m.tab {
	case tabYours:
		if m.yoursState.loading && m.tokenResolved {
			return tr.LauncherLoading
		}
	case tabStarred:
		if m.starredState.loading && m.tokenResolved {
			return tr.LauncherLoading
		}
	case tabGithub:
		if wait := time.Until(m.rateLimitedUntil); wait > 0 && m.plan.serverQuery != "" {
			return utils.ResolvePlaceholderString(tr.LauncherRateLimited,
				map[string]string{"time": m.rateLimitedUntil.Format("15:04:05")})
		}
		results := m.currentSearch()
		if (results != nil && results.loading) || (m.currentLookup() != nil && m.currentLookup().loading) ||
			(m.plan.serverQuery != "" && results == nil) {
			return tr.LauncherSearching
		}
		if results != nil && results.incomplete {
			return tr.LauncherPartialResults
		}
	case tabRecent, tabCount:
	}
	if !m.tokenResolved && m.tab != tabRecent {
		return tr.LauncherLoading
	}
	return ""
}

// emptyListMessage returns what to show instead of the list of the current
// tab when it has nothing in it.
func (self *launcher) emptyListMessage() string {
	tr := self.deps.Tr
	m := self.model
	notLoggedIn := utils.ResolvePlaceholderString(tr.LauncherNotLoggedIn, map[string]string{"host": m.host})
	loadFailed := func(err error) string {
		return errorStyle.Sprint(utils.ResolvePlaceholderString(tr.LauncherLoadFailed, map[string]string{"error": err.Error()}))
	}
	listMessage := func(state loadState) string {
		switch {
		case m.tokenResolved && m.token == "":
			return notLoggedIn
		case state.err != nil:
			return loadFailed(state.err)
		case state.loading || !m.tokenResolved:
			return ""
		default:
			return tr.LauncherNoMatches
		}
	}

	switch m.tab {
	case tabYours:
		return listMessage(m.yoursState)
	case tabStarred:
		return listMessage(m.starredState)
	case tabGithub:
		results := m.currentSearch()
		switch {
		case m.plan.serverQuery == "" && m.plan.directOwner == "" && m.query == "":
			return tr.LauncherTypeToSearch
		case m.plan.serverQuery == "" && m.plan.directOwner == "":
			return tr.LauncherKeepTyping
		case results != nil && results.err != nil:
			return loadFailed(results.err)
		case self.listStatus() != "":
			return ""
		default:
			return tr.LauncherNoMatches
		}
	case tabRecent, tabCount:
	}
	if len(m.recent) == 0 {
		return tr.LauncherNoRecentRepos
	}
	return tr.LauncherNoMatches
}

func (self *launcher) renderList(v *gocui.View) {
	m := self.model
	v.Tabs = self.tabTitles()
	v.TabIndex = int(m.tab)

	entries := m.entries[m.tab]
	if len(entries) == 0 {
		v.Highlight = false
		v.SetContent(" " + self.emptyListMessage())
		v.SetOrigin(0, 0)
		return
	}

	v.Highlight = true
	// The arrow keys move the selection while typing in the search field
	// too, so it's highlighted as active then; only a popup takes that away.
	v.HighlightInactive = true
	if self.popup == popupNone {
		v.InactiveViewSelBgColor = theme.GocuiSelectedLineBgColor
	}
	width := v.InnerWidth()
	lines := lo.Map(entries, func(e entry, _ int) string {
		if e.repo == nil {
			return self.renderRecentRow(e)
		}
		return self.renderRepoRow(e, width)
	})
	v.SetContent(strings.Join(lines, "\n"))
	v.FocusPoint(0, m.selection[m.tab], true)
}

func (self *launcher) renderRepoRow(e entry, width int) string {
	m := self.model
	repo := e.repo
	// Most of a user's own repositories have no stars, and a column of
	// zeros would only be noise.
	stars := strings.Repeat(" ", 7)
	if repo.Stars > 0 {
		stars = fmt.Sprintf("%s %5s", starsIndicator, formatCount(repo.Stars))
	}

	name := highlight(e.text, e.positions)
	if repo.IsArchived {
		name = dimStyle.Sprint(utils.Decolorise(name))
	}

	badges := []string{}
	if m.forkOf(repo) != nil {
		badges = append(badges, yourForkStyle.Sprint(yourForkBadge))
	}
	if len(m.clonesOfRepo(repo)) > 0 {
		badges = append(badges, clonedStyle.Sprint(clonedBadge))
	}
	labels := []string{}
	if repo.IsFork {
		labels = append(labels, self.deps.Tr.LauncherForkBadge)
	}
	if repo.IsPrivate {
		labels = append(labels, self.deps.Tr.LauncherPrivate)
	}
	if repo.IsArchived {
		labels = append(labels, self.deps.Tr.LauncherArchived)
	}
	if repo.Language != "" {
		labels = append(labels, repo.Language)
	}

	row := dimStyle.Sprint(stars) + "  " + name
	if len(badges) > 0 {
		row += " " + strings.Join(badges, " ")
	}
	if len(labels) > 0 {
		row += "  " + dimStyle.Sprint(strings.Join(labels, " · "))
	}
	if repo.Description != "" {
		remaining := width - utils.StringWidth(utils.Decolorise(row)) - 4
		if remaining > 10 {
			row += "  " + dimStyle.Sprint(utils.TruncateWithEllipsis(singleLine(repo.Description), remaining))
		}
	}
	return " " + row
}

func (self *launcher) renderRecentRow(e entry) string {
	return " " + highlight(e.text, e.positions)
}

func (self *launcher) renderDetail() string {
	tr := self.deps.Tr
	m := self.model
	e := m.selectedEntry()
	if e == nil {
		return ""
	}
	lines := []string{}
	if e.repo == nil {
		lines = append(lines, style.AttrBold.Sprint(m.displayPath(e.path)))
		for _, name := range m.remotesOfRecent[e.path] {
			lines = append(lines, dimStyle.Sprint(name))
		}
		return strings.Join(lines, "\n")
	}

	repo := e.repo
	lines = append(lines, style.AttrBold.Sprint(repo.FullName))
	if repo.Description != "" {
		lines = append(lines, singleLine(repo.Description))
	}
	lines = append(lines, "")

	facts := []string{
		starsIndicator + " " + utils.ResolvePlaceholderString(tr.LauncherStars, map[string]string{"count": formatCount(repo.Stars)}),
		utils.ResolvePlaceholderString(tr.LauncherForks, map[string]string{"count": formatCount(repo.Forks)}),
	}
	if repo.Language != "" {
		facts = append(facts, repo.Language)
	}
	if repo.IsPrivate {
		facts = append(facts, tr.LauncherPrivate)
	}
	if repo.IsArchived {
		facts = append(facts, tr.LauncherArchived)
	}
	lines = append(lines, strings.Join(facts, " · "))
	if !repo.PushedAt.IsZero() {
		lines = append(lines, dimStyle.Sprint(utils.ResolvePlaceholderString(tr.LauncherPushedAgo,
			map[string]string{"ago": utils.TimeAgo(repo.PushedAt, time.Now())})))
	}
	lines = append(lines, dimStyle.Sprint(repo.HTMLURL), "")

	if repo.Parent != nil {
		lines = append(lines, utils.ResolvePlaceholderString(tr.LauncherForkOf, map[string]string{"parent": repo.Parent.FullName}))
	}
	if fork := m.forkOf(repo); fork != nil {
		lines = append(lines, yourForkStyle.Sprint(yourForkBadge+" "+
			utils.ResolvePlaceholderString(tr.LauncherYourFork, map[string]string{"fork": fork.FullName})))
	}
	for _, path := range m.clonesOfRepo(repo) {
		lines = append(lines, clonedStyle.Sprint(clonedBadge+" "+
			utils.ResolvePlaceholderString(tr.LauncherClonedAt, map[string]string{"path": m.displayPath(path)})))
	}
	return strings.Join(lines, "\n")
}

func (self *launcher) layoutMenu(width int, height int) error {
	labels := lo.Map(self.actions, func(a action, _ int) string { return a.label })
	menuWidth := min(width-4, max(40, utils.MaxFn(labels, utils.StringWidth)+4))
	menuHeight := len(labels) + 1
	x0 := (width - menuWidth) / 2
	y0 := max(0, (height-menuHeight)/2)
	v, err := self.setView(menuView, x0, y0, x0+menuWidth, y0+menuHeight, true)
	if err != nil {
		return err
	}
	if e := self.model.selectedEntry(); e != nil && e.repo != nil {
		v.Title = e.repo.FullName
	}
	v.Highlight = true
	v.SetContent(strings.Join(lo.Map(labels, func(label string, _ int) string { return " " + label }), "\n"))
	v.FocusPoint(0, self.selectedAction, true)
	return nil
}

func (self *launcher) layoutPrompt(width int, height int) error {
	promptWidth := min(width-4, 80)
	x0 := (width - promptWidth) / 2
	y0 := max(0, height/2-1)
	_, err := self.g.View(promptView)
	isNew := err != nil
	v, err := self.setView(promptView, x0, y0, x0+promptWidth, y0+2, true)
	if err != nil {
		return err
	}
	if isNew {
		v.Editable = true
		v.Editor = gocui.EditorFunc(func(v *gocui.View, key gocui.Key) bool {
			self.promptError = ""
			return gocui.SimpleEditor(v, key)
		})
		v.TextArea.TypeString(self.promptInitialText)
		v.RenderTextArea()
	}
	tr := self.deps.Tr
	if self.popup == popupInitialBranch {
		v.Title = tr.LauncherInitialBranch
	} else {
		v.Title = utils.ResolvePlaceholderString(tr.LauncherCloneInto, map[string]string{"dir": self.model.displayPath(self.model.cwd)})
	}
	v.Footer = ""
	if self.promptError != "" {
		v.Footer = errorStyle.Sprint(self.promptError)
	}
	return nil
}

// highlight renders text with the characters at the given byte offsets
// highlighted.
func highlight(text string, positions []int) string {
	if len(positions) == 0 {
		return text
	}
	var builder strings.Builder
	next := 0
	for i, r := range text {
		if next < len(positions) && positions[next] == i {
			builder.WriteString(matchStyle.Sprint(string(r)))
			next++
		} else {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// formatCount formats a count compactly: 999, 1.2k, 58k, 1.3m.
func formatCount(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprint(n)
	case n < 10_000:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1000), ".0") + "k"
	case n < 1_000_000:
		return fmt.Sprintf("%dk", n/1000)
	default:
		return strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1_000_000), ".0") + "m"
	}
}

func singleLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
