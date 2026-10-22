package helpers

import (
	"context"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/github_commands"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
)

// GithubLoginHelper gets the user a GitHub token, either through the OAuth
// device flow in their browser or by pasting a personal access token, and
// stores it where gh keeps its own.
type GithubLoginHelper struct {
	c *HelperCommon

	// cancels the device login that is waiting for the user to finish in the
	// browser, if any
	cancelPendingLogin context.CancelFunc
}

func NewGithubLoginHelper(c *HelperCommon) *GithubLoginHelper {
	return &GithubLoginHelper{c: c}
}

// Login asks the user how they want to log in to host, and calls onSuccess on
// the UI thread with the token once they have.
func (self *GithubLoginHelper) Login(host string, onSuccess func(token string) error) error {
	tr := self.c.Tr.GithubActions
	placeholders := map[string]string{"host": host}

	items := []*types.MenuItem{}
	if github_commands.SupportsDeviceLogin(host) {
		items = append(items, &types.MenuItem{
			Label:   tr.LoginWithBrowser,
			OnPress: func() error { return self.deviceLogin(host, onSuccess) },
		})
	}
	items = append(items, &types.MenuItem{
		Label:   tr.LoginWithToken,
		Tooltip: tr.LoginWithTokenTooltip,
		OnPress: func() error {
			self.tokenLogin(host, onSuccess)
			return nil
		},
	})

	return self.c.Menu(types.CreateMenuOptions{
		Title:  utils.ResolvePlaceholderString(tr.LoginTitle, placeholders),
		Prompt: utils.ResolvePlaceholderString(tr.LoginPrompt, placeholders),
		Items:  items,
	})
}

func (self *GithubLoginHelper) deviceLogin(host string, onSuccess func(token string) error) error {
	tr := self.c.Tr.GithubActions

	return self.c.WithWaitingStatus(tr.RequestingLoginCode, func(gocui.Task) error {
		code, err := github_commands.RequestDeviceCode(context.Background(), host)
		if err != nil {
			return self.loginFailed(err)
		}

		self.c.OnUIThread(func() error {
			self.c.Confirm(types.ConfirmOpts{
				Title: tr.DeviceLoginTitle,
				Prompt: utils.ResolvePlaceholderString(tr.DeviceLoginPrompt, map[string]string{
					"url":  code.VerificationURI,
					"code": code.UserCode,
				}),
				CopyableText: code.UserCode,
				HandleConfirm: func() error {
					if err := self.c.OS().CopyToClipboard(code.UserCode); err != nil {
						self.c.Log.Error(err)
					}
					if err := self.c.OS().OpenLink(code.VerificationURI); err != nil {
						return err
					}
					return self.awaitDeviceLogin(host, code, onSuccess)
				},
			})
			return nil
		})
		return nil
	})
}

func (self *GithubLoginHelper) awaitDeviceLogin(host string, code *github_commands.DeviceCode, onSuccess func(token string) error) error {
	if self.cancelPendingLogin != nil {
		self.cancelPendingLogin()
	}
	ctx, cancel := context.WithCancel(context.Background())
	self.cancelPendingLogin = cancel

	return self.c.WithWaitingStatus(self.c.Tr.GithubActions.WaitingForBrowserLogin, func(gocui.Task) error {
		token, err := github_commands.AwaitDeviceToken(ctx, host, code)
		if ctx.Err() != nil {
			// superseded by a newer login attempt
			return nil
		}
		if err != nil {
			return self.loginFailed(err)
		}
		return self.completeLogin(host, token, onSuccess)
	})
}

func (self *GithubLoginHelper) tokenLogin(host string, onSuccess func(token string) error) {
	tr := self.c.Tr.GithubActions

	self.c.Prompt(types.PromptOpts{
		Title: utils.ResolvePlaceholderString(tr.PasteTokenTitle, map[string]string{"host": host}),
		Mask:  true,
		HandleConfirm: func(input string) error {
			token := strings.TrimSpace(input)
			return self.c.WithWaitingStatus(tr.CheckingToken, func(gocui.Task) error {
				return self.completeLogin(host, token, onSuccess)
			})
		},
	})
}

// completeLogin runs on a worker. It checks that the token works, finds out
// who it belongs to, and stores it.
func (self *GithubLoginHelper) completeLogin(host string, token string, onSuccess func(token string) error) error {
	username, err := github_commands.FetchUsername(context.Background(), host, token)
	if err != nil {
		return self.loginFailed(err)
	}

	tr := self.c.Tr.GithubActions
	message := utils.ResolvePlaceholderString(tr.LoggedIn, map[string]string{"username": username})
	if err := github_commands.StoreToken(host, username, token); err != nil {
		// The token still works for this session, so carry on with it.
		self.c.Log.Error(err)
		message = utils.ResolvePlaceholderString(tr.LoggedInButNotStored, map[string]string{
			"username": username,
			"error":    err.Error(),
		})
	}

	self.c.OnUIThread(func() error {
		self.c.Toast(message)
		return onSuccess(token)
	})
	return nil
}

func (self *GithubLoginHelper) loginFailed(err error) error {
	self.c.OnUIThread(func() error {
		self.c.ErrorToast(utils.ResolvePlaceholderString(self.c.Tr.GithubActions.LoginFailed, map[string]string{
			"error": err.Error(),
		}))
		return nil
	})
	return nil
}
