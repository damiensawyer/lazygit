package git_commands

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/stretchr/testify/assert"
)

func TestParseTreeFiles(t *testing.T) {
	tests := []struct {
		testName    string
		input       string
		output      []*models.TreeFile
		expectedErr string
	}{
		{
			testName: "no files",
			input:    "",
			output:   []*models.TreeFile{},
		},
		{
			testName: "one file",
			input:    "100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\tfile.txt\x00",
			output: []*models.TreeFile{
				{
					Path:     "file.txt",
					BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
					Mode:     0o100644,
				},
			},
		},
		{
			testName: "several files with special modes and characters",
			input: "100755 blob 5716ca5987cbf97d6bb54920bea6adde242d87e6\tscripts/run.sh\x00" +
				"120000 blob 8f7b8b8f5a3ff5f0d4c69c58f0f0f0f0f0f0f0f0\tlink\x00" +
				"160000 commit 46c21e9e5eb2b1eb9a4d1e6a1a7c6b1e9d1f1e1d\tvendor/submodule\x00" +
				"100644 blob 5716ca5987cbf97d6bb54920bea6adde242d87e6\tdir with spaces/file\nwith newline\x00",
			output: []*models.TreeFile{
				{
					Path:     "scripts/run.sh",
					BlobHash: "5716ca5987cbf97d6bb54920bea6adde242d87e6",
					Mode:     0o100755,
				},
				{
					Path:     "link",
					BlobHash: "8f7b8b8f5a3ff5f0d4c69c58f0f0f0f0f0f0f0f0",
					Mode:     0o120000,
				},
				{
					Path:     "vendor/submodule",
					BlobHash: "46c21e9e5eb2b1eb9a4d1e6a1a7c6b1e9d1f1e1d",
					Mode:     0o160000,
				},
				{
					Path:     "dir with spaces/file\nwith newline",
					BlobHash: "5716ca5987cbf97d6bb54920bea6adde242d87e6",
					Mode:     0o100644,
				},
			},
		},
		{
			testName:    "malformed entry",
			input:       "100644 blob deadbeef no tab here\x00",
			expectedErr: "malformed ls-tree entry: \"100644 blob deadbeef no tab here\"",
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			result, err := parseTreeFiles(test.input)
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.output, result)
			}
		})
	}
}

func TestTreeFileFlags(t *testing.T) {
	assert.True(t, (&models.TreeFile{Mode: 0o160000}).IsSubmodule())
	assert.True(t, (&models.TreeFile{Mode: 0o120000}).IsSymlink())
	assert.False(t, (&models.TreeFile{Mode: 0o100644}).IsSubmodule())
	assert.False(t, (&models.TreeFile{Mode: 0o100755}).IsSymlink())
}
