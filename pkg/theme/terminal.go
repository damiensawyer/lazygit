package theme

import (
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gocui"
)

// ConfigForTerminal returns gui.theme from the user config, with the overrides
// for the terminal's background applied.
func ConfigForTerminal(guiConfig *config.GuiConfig, detected gocui.DetectedColorScheme) config.ThemeConfig {
	return guiConfig.ThemeForBackground(
		TerminalHasLightBackground(guiConfig.ColorScheme, detected),
		terminalBackgroundColor(guiConfig.ColorScheme, detected),
	)
}

// TerminalHasLightBackground goes by colorScheme, the gui.colorScheme config,
// or by what the terminal tells us if that is 'auto'.
func TerminalHasLightBackground(colorScheme string, detected gocui.DetectedColorScheme) bool {
	switch colorScheme {
	case "dark":
		return false
	case "light":
		return true
	default:
		return detected.ColorScheme == gocui.ColorSchemeLight
	}
}

// terminalBackgroundColor returns the background color that the terminal told
// us, as #rrggbb. It returns "" if the terminal didn't tell us, or if
// gui.colorScheme disagrees with it about whether the background is light.
func terminalBackgroundColor(colorScheme string, detected gocui.DetectedColorScheme) string {
	if (detected.ColorScheme == gocui.ColorSchemeLight) != TerminalHasLightBackground(colorScheme, detected) {
		return ""
	}
	return detected.Background
}
