package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
	integrationTypes "github.com/jesseduffield/lazygit/pkg/integration/types"
	"github.com/samber/lo"
)

type TestDriver struct {
	gui        integrationTypes.GuiDriver
	keys       config.KeybindingConfig
	inputDelay int
	mouseX     int
	mouseY     int
	*assertionHelper
	shell      *Shell
	fakeGithub *FakeGithub
	// only set in demos, which show the keys being pressed
	keycast *keycast
}

func NewTestDriver(gui integrationTypes.GuiDriver, shell *Shell, keys config.KeybindingConfig, inputDelay int, isDemo bool) *TestDriver {
	return &TestDriver{
		gui:             gui,
		keys:            keys,
		inputDelay:      inputDelay,
		assertionHelper: &assertionHelper{gui: gui},
		shell:           shell,
		keycast:         lo.Ternary(isDemo, &keycast{}, nil),
	}
}

// key is something like 'w' or '<space>'. It's best not to pass a direct value,
// but instead to go through the default user config to get a more meaningful key name
func (self *TestDriver) press(keyStr string) {
	self.SetCaption(self.pressingCaption(keyStr))
	self.castKeys(func(k *keycast, now time.Time) { k.press(keyStr, now) })
	self.gui.PressKey(keyStr)
	self.Wait(self.inputDelay)
}

// for use when typing or navigating, because in demos we want that to happen
// faster
func (self *TestDriver) pressFast(keyStr string) {
	self.castKeys(func(k *keycast, now time.Time) { k.press(keyStr, now) })
	self.pressFastWithoutKeycast(keyStr)
}

func (self *TestDriver) pressFastWithoutKeycast(keyStr string) {
	self.SetCaption("")
	self.gui.PressKey(keyStr)
	self.Wait(self.inputDelay / 5)
}

// presses the keys in immediate succession, without waiting for lazygit to
// become idle in between, to simulate a user typing faster than lazygit
// processes the input
func (self *TestDriver) pressRapidly(keyStrs []string) {
	self.SetCaption(self.pressingCaption(strings.Join(keyStrs, ", ")))
	self.castKeys(func(k *keycast, now time.Time) {
		for _, keyStr := range keyStrs {
			k.press(keyStr, now)
		}
	})
	self.gui.PressKeysRapidly(keyStrs...)
	self.Wait(self.inputDelay)
}

// Demos show the keys being pressed in the keycast box, so their caption
// needn't repeat them; it keeps showing the step that the demo is at.
func (self *TestDriver) pressingCaption(keys string) string {
	if self.keycast != nil {
		return ""
	}

	return fmt.Sprintf("Pressing %s", keys)
}

// updates the keys that a demo shows as having been pressed, before the
// keypress takes effect, so that the viewer sees the key together with what it
// does
func (self *TestDriver) castKeys(record func(k *keycast, now time.Time)) {
	if self.keycast == nil {
		return
	}

	record(self.keycast, time.Now())
	self.gui.SetKeycast(self.keycast.render())
}

func (self *TestDriver) click(x, y int) {
	self.SetCaption(fmt.Sprintf("Clicking %d, %d", x, y))
	self.gui.Click(x, y)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) doubleClick(x, y int) {
	self.SetCaption(fmt.Sprintf("Double-clicking %d, %d", x, y))
	self.gui.DoubleClick(x, y)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) clickWithModifier(x, y int, modifier gocui.Modifier, what string) {
	self.SetCaption(fmt.Sprintf("%s-clicking %d, %d", what, x, y))
	self.gui.ClickWithModifier(x, y, modifier)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) clickAndHold(x, y int) {
	self.SetCaption(fmt.Sprintf("Clicking and holding %d, %d", x, y))
	self.mouseX, self.mouseY = x, y
	self.gui.ClickAndHold(x, y)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) mouseMove(x, y int) {
	self.SetCaption(fmt.Sprintf("Moving mouse to %d, %d", x, y))
	self.mouseX, self.mouseY = x, y
	self.gui.MouseMove(x, y)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) repeatMouseMove() {
	self.mouseMove(self.mouseX, self.mouseY)
}

func (self *TestDriver) scrollWheelDown(x, y int) {
	self.SetCaption(fmt.Sprintf("Scrolling down at %d, %d", x, y))
	self.gui.ScrollWheelDown(x, y)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) mouseRelease() {
	self.SetCaption(fmt.Sprintf("Releasing mouse at %d, %d", self.mouseX, self.mouseY))
	self.gui.MouseRelease(self.mouseX, self.mouseY)
	self.Wait(self.inputDelay)
}

// Should only be used in specific cases where you're doing something weird!
// E.g. invoking a global keybinding from within a popup.
// You probably shouldn't use this function, and should instead go through a view like t.Views().Commit().Focus().Press(...)
func (self *TestDriver) GlobalPress(key config.Keybinding) {
	self.press(key[0])
}

// asserts that the terminal's text cursor is shown, i.e. that there is a text
// field to type into
func (self *TestDriver) CursorIsVisible() *TestDriver {
	self.assertWithRetries(func() (bool, string) {
		return self.gui.CursorVisible(), "Expected the cursor to be visible"
	})

	return self
}

func (self *TestDriver) CursorIsHidden() *TestDriver {
	self.assertWithRetries(func() (bool, string) {
		return !self.gui.CursorVisible(), "Expected the cursor to be hidden"
	})

	return self
}

// FocusIn simulates the terminal window regaining focus, which causes lazygit
// to reload any config files that changed while it was in the background.
func (self *TestDriver) FocusIn() {
	self.SetCaption("Focusing window")
	self.gui.FocusIn()
	self.Wait(self.inputDelay)
}

func (self *TestDriver) focusInAndClick(x, y int) {
	self.SetCaption(fmt.Sprintf("Focusing window and clicking %d, %d", x, y))
	self.gui.FocusInAndClick(x, y)
	self.Wait(self.inputDelay)
}

func (self *TestDriver) typeContent(content string) {
	for _, char := range content {
		self.castKeys(func(k *keycast, now time.Time) { k.typeChar(string(char), now) })
		self.pressFastWithoutKeycast(string(char))
	}
}

func (self *TestDriver) Common() *Common {
	return &Common{t: self}
}

// for when you want to allow lazygit to process something before continuing
func (self *TestDriver) Wait(milliseconds int) {
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}

func (self *TestDriver) SetCaption(caption string) {
	self.gui.SetCaption(caption)
}

func (self *TestDriver) SetCaptionPrefix(prefix string) {
	self.gui.SetCaptionPrefix(prefix)
}

func (self *TestDriver) LogUI(message string) {
	self.gui.LogUI(message)
}

func (self *TestDriver) Log(message string) {
	self.gui.LogUI(message)
}

// RefreshInBackground performs the refresh that lazygit's background routines
// perform on a timer, e.g. to pick up changes made by RunCommand. Tests use this
// rather than turning those routines on and waiting for them.
func (self *TestDriver) RefreshInBackground() {
	self.SetCaption("Refreshing in the background")
	self.gui.RefreshInBackground()
	self.Wait(self.inputDelay)
}

// allows the user to run shell commands during the test to emulate background activity
func (self *TestDriver) Shell() *Shell {
	return self.shell
}

// for making assertions on lazygit views
func (self *TestDriver) Views() *Views {
	return &Views{t: self}
}

// for interacting with popups
func (self *TestDriver) ExpectPopup() *Popup {
	return &Popup{t: self}
}

func (self *TestDriver) ExpectToast(matcher *TextMatcher) *TestDriver {
	t := self.gui.NextToast()
	if t == nil {
		self.gui.Fail("Expected toast, but didn't get one")
	} else {
		self.matchString(matcher, "Unexpected toast message",
			func() string {
				return *t
			},
		)
	}

	return self
}

func (self *TestDriver) ExpectSearch() *SearchDriver {
	self.inSearch()

	return &SearchDriver{t: self}
}

func (self *TestDriver) inSearch() {
	self.assertWithRetries(func() (bool, string) {
		currentView := self.gui.CurrentContext().GetView()
		return currentView.Name() == "search", "Expected search prompt to be focused"
	})
}

// for making assertions through git itself
func (self *TestDriver) Git() *Git {
	return &Git{assertionHelper: self.assertionHelper, shell: self.shell}
}

// for making assertions on the file system
func (self *TestDriver) FileSystem() *FileSystem {
	return &FileSystem{assertionHelper: self.assertionHelper}
}

// for making assertions on the requests made to the test's FakeGithub
func (self *TestDriver) FakeGithub() *FakeGithubDriver {
	if self.fakeGithub == nil {
		self.Fail("The test has no FakeGithub")
	}
	return &FakeGithubDriver{t: self, fake: self.fakeGithub}
}

// for when you just want to fail the test yourself.
// This runs callbacks to ensure we render the error after closing the gui.
func (self *TestDriver) Fail(message string) {
	self.assertionHelper.fail(message)
}
