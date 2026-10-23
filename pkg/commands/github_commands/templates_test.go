package github_commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for path, content := range files {
		full := filepath.Join(root, path)
		assert.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		assert.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
}

func TestFindPullRequestTemplates(t *testing.T) {
	scenarios := []struct {
		name     string
		files    map[string]string
		expected []PullRequestTemplate
	}{
		{
			name:     "none",
			files:    map[string]string{"README.md": "hi"},
			expected: []PullRequestTemplate{},
		},
		{
			name:     "a single template in .github, in any case and with dashes",
			files:    map[string]string{".github/pull-request-template.md": "## Summary\n"},
			expected: []PullRequestTemplate{{Name: "pull-request-template.md", Body: "## Summary\n"}},
		},
		{
			name: "a single template at the root, preferring .github",
			files: map[string]string{
				"PULL_REQUEST_TEMPLATE.md":         "root",
				".github/PULL_REQUEST_TEMPLATE.md": "github",
			},
			expected: []PullRequestTemplate{{Name: "PULL_REQUEST_TEMPLATE.md", Body: "github"}},
		},
		{
			name: "several templates in a directory, with front matter, taking precedence over a single one",
			files: map[string]string{
				".github/PULL_REQUEST_TEMPLATE.md":            "legacy",
				"docs/pull_request_template/feature.md":       "---\nname: Feature\nabout: A new feature\n---\n## Feature\n",
				"docs/pull_request_template/bug.md":           "## Bug\n",
				"docs/pull_request_template/not-markdown.txt": "nope",
			},
			expected: []PullRequestTemplate{
				{Name: "bug.md", Body: "## Bug\n"},
				{Name: "Feature", Body: "## Feature\n"},
			},
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, s.files)
			assert.Equal(t, s.expected, FindPullRequestTemplates(root))
		})
	}
}
