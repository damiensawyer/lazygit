package diff

import (
	"fmt"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

// twentyLines returns the lines "line01" to "line20", with the ones whose
// (one-based) numbers are given upper-cased, so that a change to a single line
// leaves most of the file outside a diff's default context.
func twentyLines(upperCased ...int) string {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%02d", i+1)
	}
	for _, n := range upperCased {
		lines[n-1] = strings.ToUpper(lines[n-1])
	}
	return strings.Join(lines, "\n") + "\n"
}

var ToggleFullFileContext = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Toggle a working tree file's diff between the configured context size and the whole file",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig:  func(cfg *config.AppConfig) {},
	SetupRepo: func(shell *Shell) {
		shell.CreateFileAndAdd("file.txt", twentyLines())
		shell.Commit("initial")
		shell.UpdateFile("file.txt", twentyLines(10))
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Views().Files().
			IsFocused().
			SelectedLine(Contains("file.txt"))

		// Three lines of context either side of the change.
		t.Views().Main().
			Content(Contains("@@ -7,7 +7,7 @@")).
			Content(DoesNotContain(" line01"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Showing full file diff"))

		// The whole file, still as a diff.
		t.Views().Main().
			Content(Contains("@@ -1,20 +1,20 @@")).
			Content(Contains(" line01")).
			Content(Contains("-line10")).
			Content(Contains("+LINE10")).
			Content(Contains(" line20"))

		t.Views().Files().Press(keys.Universal.ToggleContextInDiffView)
		t.ExpectToast(Equals("Restored context size to 3"))

		t.Views().Main().
			Content(Contains("@@ -7,7 +7,7 @@")).
			Content(DoesNotContain(" line01"))
	},
})
