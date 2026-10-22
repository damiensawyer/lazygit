package controllers

import (
	"strings"

	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
)

// githubCopyItem is something the GitHub Actions views offer to copy: a label
// for it, and a function that produces it, passing it on to the given
// callback, since some of them have to be loaded first.
type githubCopyItem struct {
	label string
	get   func(func(string) error) error
}

func githubCopyValue(label string, value string) githubCopyItem {
	return githubCopyItem{label: label, get: func(f func(string) error) error { return f(value) }}
}

func githubCopyMenu(c *ControllerCommon, items []githubCopyItem) error {
	return c.Menu(types.CreateMenuOptions{
		Title: c.Tr.GithubActions.CopyToClipboard,
		Items: lo.Map(items, func(item githubCopyItem, _ int) *types.MenuItem {
			return &types.MenuItem{
				Label: item.label,
				OnPress: func() error {
					return item.get(func(value string) error {
						if err := c.OS().CopyToClipboard(utils.Decolorise(value)); err != nil {
							return err
						}
						c.Toast(utils.ResolvePlaceholderString(c.Tr.GithubActions.CopiedToClipboard, map[string]string{
							"what": item.label,
						}))
						return nil
					})
				},
			}
		}),
	})
}

func joinLogLines(lines []string) string {
	return strings.Join(lines, "\n") + "\n"
}
