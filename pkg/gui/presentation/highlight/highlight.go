// Package highlight renders file contents with syntax highlighting for display
// in a terminal.
package highlight

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// ansiStyle colours tokens by referring to the terminal's own 16 ANSI slots, so
// that highlighting follows whatever theme the terminal has been given rather
// than imposing a palette of its own.
//
// The hex values are not rendered literally: the terminal16 formatter maps each
// to its nearest entry in a fixed 16-colour table, and these are exactly that
// table's anchors, so each one selects its slot precisely. Token types left out
// here emit no colour at all and inherit the terminal's foreground, which is why
// there is deliberately no Text or Background entry — defining one would make
// every unclassified token, including plain identifiers and whitespace, override
// the user's foreground colour.
var ansiStyle = chroma.MustNewStyle("lazygit-ansi", chroma.StyleEntries{
	chroma.Comment:         "#555555", // bright black: dim, so comments recede
	chroma.CommentPreproc:  "#ff00ff", // bright magenta
	chroma.Keyword:         "#ff00ff", // bright magenta
	chroma.KeywordType:     "#ffff00", // bright yellow
	chroma.NameBuiltin:     "#00ffff", // bright cyan
	chroma.NameClass:       "#ffff00", // bright yellow
	chroma.NameFunction:    "#0000ff", // bright blue
	chroma.NameTag:         "#ff00ff", // bright magenta
	chroma.NameAttribute:   "#00ffff", // bright cyan
	chroma.NameDecorator:   "#00ffff", // bright cyan
	chroma.LiteralString:   "#00ff00", // bright green
	chroma.LiteralNumber:   "#ff0000", // bright red
	chroma.Operator:        "#00ffff", // bright cyan
	chroma.GenericInserted: "#00ff00", // bright green
	chroma.GenericDeleted:  "#ff0000", // bright red
	chroma.GenericHeading:  "#ffff00", // bright yellow
	chroma.Error:           "#ff0000", // bright red
})

// maxSizeForHighlighting caps the input we're willing to tokenise. Chroma is
// regex-driven and costs roughly a millisecond per kilobyte, so tokenising a
// file far larger than anyone can read in the main view would burn seconds of
// CPU for no benefit. Above this size we render it unhighlighted.
const maxSizeForHighlighting = 512 * 1024

// StyleNames returns the names of every style that can be configured, sorted.
func StyleNames() []string {
	return styles.Names()
}

// IsValidStyle reports whether name refers to a style we can render with.
func IsValidStyle(name string) bool {
	_, ok := styles.Registry[name]
	return ok
}

// FileContent returns source annotated with ANSI escape codes, choosing a lexer
// from the file's path and falling back to sniffing the content itself. Source
// is returned unchanged when no lexer fits it, when it's too large to tokenise,
// or when highlighting fails, so callers can render the result either way.
//
// An empty styleName means "use the terminal's colours", which keeps
// highlighting in step with the terminal's theme. Naming a style instead renders
// that style's own colours at 256-colour fidelity, ignoring the terminal theme.
func FileContent(path string, source string, styleName string) string {
	if len(source) > maxSizeForHighlighting {
		return source
	}

	lexer := lexers.Match(path)
	if lexer == nil {
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		return source
	}

	// Without this, every token gets its own escape sequence, including runs of
	// plain whitespace.
	lexer = chroma.Coalesce(lexer)

	formatter, style := formatters.Get("terminal16"), ansiStyle
	if styleName != "" {
		formatter, style = formatters.Get("terminal256"), styles.Get(styleName)
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}

	var sb strings.Builder
	if err := formatter.Format(&sb, style, iterator); err != nil {
		return source
	}

	return sb.String()
}
