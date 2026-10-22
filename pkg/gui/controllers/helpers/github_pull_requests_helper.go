package helpers

import (
	goContext "context"
	"os"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/context"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

const (
	// How often the pull requests are reloaded while they are shown: more
	// often while some of them have checks running, to show those finish.
	pullRequestsActiveRefreshInterval = 20 * time.Second
	pullRequestsIdleRefreshInterval   = time.Minute
)

// PullRequestsLoadState says what the pull requests panel has to show for, in
// place of pull requests, while it has none.
type PullRequestsLoadState int

const (
	PullRequestsNotLoaded PullRequestsLoadState = iota
	PullRequestsLoading
	PullRequestsLoaded
	// we couldn't load them without asking the user to log in, or to choose
	// which remote is the GitHub repository
	PullRequestsNeedLogin
	PullRequestsLoadFailed
)

// GithubPullRequestDetails is what we load about a pull request in addition
// to what the list says about it, for showing it in the main view.
type GithubPullRequestDetails struct {
	// the pull request as loaded along with its details, which may be newer
	// than the one in the list
	PullRequest *models.GithubPullRequest
	Details     *models.GithubPullRequestDetails
	Err         error

	// what the list said about the pull request when the details were
	// loaded; checks finishing changes its checks state but not when it was
	// last updated
	listUpdatedAt   time.Time
	listChecksState string
}

// GithubPullRequestsHelper loads the pull requests of the current repository's
// GitHub repository and keeps them fresh while they are shown.
type GithubPullRequestsHelper struct {
	c             *HelperCommon
	windowHelper  *WindowHelper
	clientHelper  *GithubClientHelper
	commitsHelper *CommitsHelper
	filesHelper   *FilesHelper

	loadState PullRequestsLoadState
	loadErr   error

	// bumped whenever the pull requests are loaded, so that an auto-refresh
	// scheduled before then knows it has been superseded
	refreshGeneration int

	// keyed by number; only touched on the UI thread
	details        map[int]*GithubPullRequestDetails
	detailsLoading map[int]bool
}

func NewGithubPullRequestsHelper(
	c *HelperCommon,
	windowHelper *WindowHelper,
	clientHelper *GithubClientHelper,
	commitsHelper *CommitsHelper,
	filesHelper *FilesHelper,
) *GithubPullRequestsHelper {
	return &GithubPullRequestsHelper{
		c:              c,
		windowHelper:   windowHelper,
		clientHelper:   clientHelper,
		commitsHelper:  commitsHelper,
		filesHelper:    filesHelper,
		details:        map[int]*GithubPullRequestDetails{},
		detailsLoading: map[int]bool{},
	}
}

func (self *GithubPullRequestsHelper) LoadState() (PullRequestsLoadState, error) {
	return self.loadState, self.loadErr
}

// OnFocus loads the pull requests the first time the panel is shown, if that
// can be done without asking the user anything.
func (self *GithubPullRequestsHelper) OnFocus() {
	if self.loadState != PullRequestsNotLoaded {
		return
	}
	if !self.clientHelper.HasGithubRemote() {
		self.loadState = PullRequestsNeedLogin
		return
	}

	self.setLoadState(PullRequestsLoading, nil)
	_ = self.clientHelper.TryClient(
		func(*github_commands.Client) error {
			self.c.OnWorkerBackground(func(gocui.Task) error {
				return self.loadPullRequests(true, func() {})
			})
			return nil
		},
		func() error {
			self.setLoadState(PullRequestsNeedLogin, nil)
			return nil
		})
}

// Refresh reloads the pull requests, logging in first if needed.
func (self *GithubPullRequestsHelper) Refresh() error {
	return self.clientHelper.WithClient(func(*github_commands.Client) error {
		if self.loadState != PullRequestsLoaded {
			self.setLoadState(PullRequestsLoading, nil)
		}
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.Loading, func(gocui.Task) error {
			return self.loadPullRequests(false, func() {})
		})
	})
}

// SetListFilter changes which pull requests are shown, and reloads them.
func (self *GithubPullRequestsHelper) SetListFilter(filter github_commands.ListPullRequestsOptions) error {
	prsContext := self.c.Contexts().PullRequests
	prsContext.SetListFilter(filter)
	self.updateSubtitle()
	return self.clientHelper.WithClient(func(*github_commands.Client) error {
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.Loading, func(gocui.Task) error {
			return self.loadPullRequests(false, func() {
				prsContext.SetSelection(0)
				self.c.PostRefreshUpdate(prsContext)
			})
		})
	})
}

func (self *GithubPullRequestsHelper) setLoadState(state PullRequestsLoadState, err error) {
	self.loadState = state
	self.loadErr = err
	if self.c.Context().IsCurrent(self.c.Contexts().PullRequests) {
		self.c.Contexts().PullRequests.HandleRenderToMain()
	}
}

// loadPullRequests runs on a worker. It fetches the pull requests and, on the
// UI thread, shows them and then calls then. A background load is one the
// user didn't ask for, so it doesn't report errors in a popup.
func (self *GithubPullRequestsHelper) loadPullRequests(background bool, then func()) error {
	client := self.clientHelper.Client()
	if client == nil {
		return nil
	}
	generation := self.c.State().GetRepoGeneration()
	filter := self.c.Contexts().PullRequests.ListFilter()

	prs, err := client.ListPullRequests(goContext.Background(), filter)

	onUIThread := self.c.OnUIThread
	if background {
		onUIThread = self.c.OnUIThreadBackground
	}
	onUIThread(func() error {
		if self.c.State().GetRepoGeneration() != generation {
			return nil
		}
		if err != nil {
			if self.loadState != PullRequestsLoaded {
				self.setLoadState(PullRequestsLoadFailed, err)
			}
			self.scheduleAutoRefresh()
			if background {
				self.c.Log.Warnf("Failed to load pull requests: %v", err)
				return nil
			}
			return self.clientHelper.HandleError(err, self.Refresh)
		}
		self.setPullRequests(prs)
		self.setLoadState(PullRequestsLoaded, nil)
		then()
		self.scheduleAutoRefresh()
		return nil
	})
	return nil
}

func (self *GithubPullRequestsHelper) setPullRequests(prs []*models.GithubPullRequest) {
	prsContext := self.c.Contexts().PullRequests
	selectedId := prsContext.GetSelectedItemId()

	self.c.Model().ListedPullRequests = prs
	prsContext.ReApplyFilter(self.c.UserConfig().Gui.UseFuzzySearch())

	// A pull request that was updated moves to the top, so keep the selection
	// on the pull request rather than on the position.
	if selectedId != "" {
		if _, idx, found := lo.FindIndexOf(prsContext.GetItems(), func(pr *models.GithubPullRequest) bool {
			return pr.ID() == selectedId
		}); found {
			prsContext.SetSelection(idx)
		}
	}

	self.c.PostRefreshUpdateWithOptions(prsContext,
		types.OnFocusOpts{KeepScrollPosition: true})
}

func (self *GithubPullRequestsHelper) updateSubtitle() {
	self.c.Views().PullRequests.Subtitle = self.DescribeListFilter()
}

// DescribeListFilter says in a few words which pull requests are shown, or
// returns "" for the open ones.
func (self *GithubPullRequestsHelper) DescribeListFilter() string {
	tr := self.c.Tr.PullRequests
	filter := self.c.Contexts().PullRequests.ListFilter()

	parts := []string{}
	switch filter.State {
	case github_commands.PullRequestStateClosed:
		parts = append(parts, tr.FilterClosed)
	case github_commands.PullRequestStateMerged:
		parts = append(parts, tr.FilterMerged)
	case github_commands.PullRequestStateAll:
		parts = append(parts, tr.FilterAll)
	}
	add := func(template string, value string) {
		if value != "" {
			parts = append(parts, utils.ResolvePlaceholderString(template, map[string]string{"value": value}))
		}
	}
	add(tr.FilterDescriptionAuthor, filter.Author)
	add(tr.FilterDescriptionAssignee, filter.Assignee)
	add(tr.FilterDescriptionReviewRequested, filter.ReviewRequested)
	for _, label := range filter.Labels {
		add(tr.FilterDescriptionLabel, label)
	}
	add(tr.FilterDescriptionBase, filter.BaseBranch)
	add(tr.FilterDescriptionSearch, filter.Search)
	return strings.Join(parts, " · ")
}

// IsShowing reports whether the pull requests panel is visible.
func (self *GithubPullRequestsHelper) IsShowing() bool {
	prsContext := self.c.Contexts().PullRequests
	return self.windowHelper.GetViewNameForWindow(prsContext.GetWindowName()) == prsContext.GetViewName()
}

func (self *GithubPullRequestsHelper) scheduleAutoRefresh() {
	self.refreshGeneration++
	refreshGeneration := self.refreshGeneration
	repoGeneration := self.c.State().GetRepoGeneration()

	interval := pullRequestsIdleRefreshInterval
	if lo.SomeBy(self.c.Model().ListedPullRequests, func(pr *models.GithubPullRequest) bool {
		return pr.ChecksState == "PENDING"
	}) {
		interval = pullRequestsActiveRefreshInterval
	}

	time.AfterFunc(interval, utils.SafeFunc(func() {
		self.c.OnUIThreadBackground(func() error {
			if refreshGeneration != self.refreshGeneration ||
				repoGeneration != self.c.State().GetRepoGeneration() ||
				self.clientHelper.Client() == nil {
				return nil
			}
			if !self.IsShowing() {
				// Look again later; the panel may be shown again by switching
				// back to its tab.
				self.scheduleAutoRefresh()
				return nil
			}
			self.c.OnWorkerBackground(func(gocui.Task) error {
				return self.loadPullRequests(true, func() {})
			})
			return nil
		})
	}))
}

// GetDetails returns the details of a pull request if they have been loaded,
// or nil if they haven't. Unless they are up to date, it starts loading them,
// and re-renders the main view once they have been, if the pull request is
// still selected by then. Outdated details are returned in the meantime, so
// that the main view doesn't flash a loading message on every refresh. Must
// be called on the UI thread.
func (self *GithubPullRequestsHelper) GetDetails(pr *models.GithubPullRequest) *GithubPullRequestDetails {
	details, loaded := self.details[pr.Number]
	upToDate := loaded && details.listUpdatedAt.Equal(pr.UpdatedAt) && details.listChecksState == pr.ChecksState
	if upToDate || self.detailsLoading[pr.Number] || self.clientHelper.Client() == nil {
		return details
	}
	self.loadDetailsInBackground(pr, func(*GithubPullRequestDetails) {
		prsContext := self.c.Contexts().PullRequests
		if self.c.Context().IsCurrent(prsContext) && prsContext.GetSelectedItemId() == pr.ID() {
			prsContext.HandleRenderToMain()
		}
	})
	return details
}

// WithDetails passes the up-to-date details of a pull request to f on the UI
// thread, loading them first if needed.
func (self *GithubPullRequestsHelper) WithDetails(pr *models.GithubPullRequest, f func(*GithubPullRequestDetails) error) error {
	return self.clientHelper.WithClient(func(client *github_commands.Client) error {
		if details, ok := self.details[pr.Number]; ok && details.Err == nil &&
			details.listUpdatedAt.Equal(pr.UpdatedAt) && details.listChecksState == pr.ChecksState {
			return f(details)
		}
		generation := self.c.State().GetRepoGeneration()
		return self.c.WithWaitingStatus(self.c.Tr.PullRequests.LoadingDetails, func(gocui.Task) error {
			details := loadPullRequestDetails(client, pr)
			self.c.OnUIThread(func() error {
				if self.c.State().GetRepoGeneration() != generation {
					return nil
				}
				if details.Err != nil {
					return self.clientHelper.HandleError(details.Err, self.Refresh)
				}
				self.details[pr.Number] = details
				return f(details)
			})
			return nil
		})
	})
}

// InvalidateDetails makes the details of a pull request reload the next time
// they are needed, for after changing it.
func (self *GithubPullRequestsHelper) InvalidateDetails(number int) {
	delete(self.details, number)
}

func (self *GithubPullRequestsHelper) loadDetailsInBackground(pr *models.GithubPullRequest, then func(*GithubPullRequestDetails)) {
	self.detailsLoading[pr.Number] = true

	client := self.clientHelper.Client()
	generation := self.c.State().GetRepoGeneration()
	self.c.OnWorkerBackground(func(gocui.Task) error {
		details := loadPullRequestDetails(client, pr)
		self.c.OnUIThreadBackground(func() error {
			if self.c.State().GetRepoGeneration() != generation {
				return nil
			}
			delete(self.detailsLoading, pr.Number)
			self.details[pr.Number] = details
			then(details)
			return nil
		})
		return nil
	})
}

func loadPullRequestDetails(client *github_commands.Client, pr *models.GithubPullRequest) *GithubPullRequestDetails {
	loadedPr, details, err := client.GetPullRequest(goContext.Background(), pr.Number)
	return &GithubPullRequestDetails{
		PullRequest:     loadedPr,
		Details:         details,
		Err:             err,
		listUpdatedAt:   pr.UpdatedAt,
		listChecksState: pr.ChecksState,
	}
}

// RunAction changes a pull request on GitHub, like merging it, and then
// reloads the pull requests to show its effect.
func (self *GithubPullRequestsHelper) RunAction(
	pr *models.GithubPullRequest,
	waitingStatus string,
	doneToast string,
	action func(ctx goContext.Context, client *github_commands.Client) error,
) error {
	return self.clientHelper.WithClient(func(client *github_commands.Client) error {
		return self.c.WithWaitingStatus(waitingStatus, func(gocui.Task) error {
			if err := action(goContext.Background(), client); err != nil {
				self.c.OnUIThread(func() error { return self.clientHelper.HandleError(err, self.Refresh) })
				return nil
			}
			self.c.OnUIThread(func() error {
				if pr != nil {
					self.InvalidateDetails(pr.Number)
				}
				if doneToast != "" {
					self.c.Toast(doneToast)
				}
				return nil
			})
			return self.loadPullRequests(false, func() {})
		})
	})
}

// MainViewMessage says what the main view shows while the panel has no pull
// request to show.
func (self *GithubPullRequestsHelper) MainViewMessage() string {
	tr := self.c.Tr.PullRequests
	switch self.loadState {
	case PullRequestsNeedLogin:
		if !self.clientHelper.HasGithubRemote() {
			return tr.NoGithubRemote
		}
		template := tr.NotLoggedIn
		if self.clientHelper.BaseRemoteName() == "" {
			template = tr.ChooseRepository
		}
		return utils.ResolvePlaceholderString(template, map[string]string{
			"key": self.c.UserConfig().Keybinding.Universal.Refresh.String(),
		})
	case PullRequestsLoadFailed:
		return utils.ResolvePlaceholderString(tr.LoadError, map[string]string{"error": self.loadErr.Error()})
	case PullRequestsLoaded:
		return tr.NoPullRequests
	default:
		return tr.Loading
	}
}

type EditTextOpts struct {
	Title string
	// what the text starts out as: a first line, and the rest of it
	InitialText string
}

// EditText lets the user write a comment or the like, in the commit message
// panel or, from there, in their editor, and passes it to onConfirm.
func (self *GithubPullRequestsHelper) EditText(opts EditTextOpts, onConfirm func(text string) error) error {
	self.commitsHelper.OpenCommitMessagePanel(&OpenCommitMessagePanelOpts{
		CommitIndex:       context.NoCommitIndex,
		SummaryTitle:      opts.Title,
		DescriptionTitle:  self.c.Tr.PullRequests.TextContinued,
		InitialMessage:    opts.InitialText,
		UnwrapDescription: true,
		OnConfirm: func(summary string, description string) error {
			return onConfirm(joinFirstLineAndRest(summary, description))
		},
		OnSwitchToEditor: func(path string) error {
			if err := self.filesHelper.EditFileAtLineAndWait(path, 1); err != nil {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := strings.TrimSpace(string(content))
			if text == "" {
				return nil
			}
			return onConfirm(text)
		},
	})
	return nil
}

func joinFirstLineAndRest(firstLine string, rest string) string {
	if strings.TrimSpace(rest) == "" {
		return strings.TrimSpace(firstLine)
	}
	return strings.TrimSpace(firstLine) + "\n\n" + strings.TrimSpace(rest)
}
