package helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsBinary(t *testing.T) {
	scenarios := []struct {
		name     string
		content  string
		expected bool
	}{
		{"empty", "", false},
		{"text", "hello\nworld\n", false},
		{"invalid UTF-8 without a NUL", "\xff\xfe\xfd", false},
		{"NUL at the start", "\x00abc", true},
		{"NUL in the middle", "abc\x00def", true},
		{"NUL as the last sniffed byte", strings.Repeat("a", binarySniffLength-1) + "\x00", true},
		// Like git, only the start of the file is inspected.
		{"NUL just past the sniffed length", strings.Repeat("a", binarySniffLength) + "\x00", false},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			assert.Equal(t, s.expected, isBinary(s.content))
		})
	}
}
