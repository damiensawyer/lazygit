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
