package highlight

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const goSource = "package main\n\n// a comment\nfunc greet(name string) int {\n\treturn 42\n}\n"

func TestFileContentHighlightsKnownLanguages(t *testing.T) {
	output := FileContent("main.go", goSource, "")

	assert.NotEqual(t, goSource, output, "expected the Go source to be highlighted")
	assert.Contains(t, output, "\x1b[95m", "expected keywords to use the bright magenta ANSI slot")
	assert.Contains(t, output, "\x1b[90m", "expected comments to use the bright black ANSI slot")
	assert.Equal(t, goSource, stripAnsi(output), "highlighting must not alter the text itself")
}

// The default style refers to the terminal's ANSI slots rather than naming
// colours outright, which is what keeps highlighting in step with the user's
// terminal theme.
func TestFileContentUsesAnsiSlotsByDefault(t *testing.T) {
	output := FileContent("main.go", goSource, "")

	assert.NotContains(t, output, "\x1b[38;5;", "expected no 256-colour codes")
	assert.NotContains(t, output, "\x1b[38;2;", "expected no truecolor codes")
}

// Anything we don't deliberately colour must emit no escape code, so that it
// inherits the terminal's foreground instead of overriding it.
func TestFileContentLeavesPlainTextUncoloured(t *testing.T) {
	output := FileContent("main.go", goSource, "")

	// "name" is an unclassified identifier and the parentheses around it are
	// punctuation; neither should carry an escape code of its own.
	assert.Contains(t, output, "(name ", "expected identifiers and punctuation to be left uncoloured")
	assert.NotContains(t, output, "\x1b[97m", "expected no token to force the foreground to bright white")
}

func TestFileContentWithNamedStyleUses256Colours(t *testing.T) {
	output := FileContent("main.go", goSource, "monokai")

	assert.Contains(t, output, "\x1b[38;5;", "expected a named style to render at 256-colour fidelity")
	assert.Equal(t, goSource, stripAnsi(output))
}

func TestFileContentFallsBackToPlainText(t *testing.T) {
	scenarios := []struct {
		name   string
		path   string
		source string
	}{
		{"unknown extension", "notes.xyzzy", "just some prose\n"},
		{"no extension", "LICENSE", "All rights reserved.\n"},
		{"binary content", "thing.bin", "\x00\x01\x02\xff\xfe"},
		{"empty file", "empty.go", ""},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.source, FileContent(s.path, s.source, ""))
		})
	}
}

// Files big enough to stall the UI thread are rendered unhighlighted.
func TestFileContentSkipsHighlightingLargeFiles(t *testing.T) {
	large := strings.Repeat("var x = 1\n", maxSizeForHighlighting/10+1)

	assert.Equal(t, large, FileContent("main.go", large, ""))
}

// A file with no extension still gets highlighted if its content identifies it.
func TestFileContentDetectsLanguageFromContent(t *testing.T) {
	source := "#!/bin/bash\necho hello\n"

	assert.NotEqual(t, source, FileContent("runme", source, ""))
}

// The lexer is picked from the file name alone where it can be, which includes
// well-known names without an extension and paths in subdirectories.
func TestFileContentChoosesLexerFromPath(t *testing.T) {
	scenarios := []struct {
		name   string
		path   string
		source string
	}{
		{"nested path", "pkg/sub/main.go", "func f() {}\n"},
		{"upper-case extension", "MAIN.GO", "func f() {}\n"},
		{"extensionless well-known name", "Dockerfile", "FROM alpine\n"},
		{"well-known name in a subdirectory", "build/Dockerfile", "FROM alpine\n"},
		{"Makefile", "Makefile", "all:\n\techo hi\n"},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			output := FileContent(s.path, s.source, "")

			assert.NotEqual(t, s.source, output, "expected the source to be highlighted")
			assert.Equal(t, s.source, stripAnsi(output))
		})
	}
}

// Content sniffing is only a fallback: when the path already names a language,
// the content doesn't get to override it.
func TestFileContentPrefersPathOverContent(t *testing.T) {
	source := "#!/bin/bash\necho hello\n"

	assert.Equal(t, source, FileContent("notes.txt", source, ""))
}

// The configured style is validated when the config is loaded, but an unknown
// name reaching us anyway must still render the file intact.
func TestFileContentWithUnknownStyleKeepsText(t *testing.T) {
	output := FileContent("main.go", goSource, "not-a-real-style")

	assert.Equal(t, goSource, stripAnsi(output))
}

func TestFileContentWithNamedStyleAndEmptySource(t *testing.T) {
	assert.Equal(t, "", FileContent("main.go", "", "monokai"))
}

func TestFileContentKeepsLongLinesIntact(t *testing.T) {
	source := "package main\n\nvar s = \"" + strings.Repeat("a", 100_000) + "\"\n"

	output := FileContent("main.go", source, "")

	assert.Contains(t, output, "\x1b[92m", "expected the long string literal to be coloured")
	assert.Equal(t, source, stripAnsi(output))
}

func TestFileContentKeepsMultiByteCharactersIntact(t *testing.T) {
	source := "package main\n\nvar s = \"日本語\" // ünïcödé\n"

	assert.Equal(t, source, stripAnsi(FileContent("main.go", source, "")))
}

// Chroma decodes its input as UTF-8, so a byte that isn't valid UTF-8 comes
// out as a replacement character, just as the view would display it anyway.
// What matters is that the rest of the file is still highlighted and kept.
func TestFileContentWithInvalidUTF8(t *testing.T) {
	source := "package main\n\n// \xff\xfe\nvar s = \"\xc3\x28\"\n"

	output := FileContent("main.go", source, "")

	assert.Contains(t, output, "\x1b[95m", "expected keywords to still be coloured")
	assert.Equal(t, "package main\n\n// ��\nvar s = \"�(\"\n", stripAnsi(output))
}

// The size cap is inclusive: a file of exactly the maximum size is still
// highlighted.
func TestFileContentHighlightsAFileOfExactlyTheMaximumSize(t *testing.T) {
	source := strings.Repeat("var x = 1\n", maxSizeForHighlighting/10)
	source += strings.Repeat("\n", maxSizeForHighlighting-len(source))

	assert.NotEqual(t, source, FileContent("main.go", source, ""))
}

// Every style we offer in the config has to be one we can render with.
func TestEveryStyleNameRenders(t *testing.T) {
	names := StyleNames()
	assert.NotEmpty(t, names)
	assert.IsIncreasing(t, names)

	for _, name := range names {
		assert.True(t, IsValidStyle(name), name)
		assert.Equal(t, goSource, stripAnsi(FileContent("main.go", goSource, name)), name)
	}
}

func TestIsValidStyle(t *testing.T) {
	assert.True(t, IsValidStyle("monokai"))
	assert.False(t, IsValidStyle("not-a-real-style"))
	assert.False(t, IsValidStyle(""))
}

func TestStyleNames(t *testing.T) {
	assert.Contains(t, StyleNames(), "monokai")
}

func stripAnsi(s string) string {
	var sb strings.Builder
	for {
		start := strings.Index(s, "\x1b[")
		if start == -1 {
			sb.WriteString(s)
			return sb.String()
		}
		sb.WriteString(s[:start])
		end := strings.Index(s[start:], "m")
		if end == -1 {
			return sb.String()
		}
		s = s[start+end+1:]
	}
}
