package git_commands

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
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
			testName: "last entry without trailing NUL",
			input: "100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\ta\x00" +
				"100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\tb",
			output: []*models.TreeFile{
				{Path: "a", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
				{Path: "b", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
			},
		},
		{
			testName: "empty entries are skipped",
			input:    "\x00\x00100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\ta\x00\x00",
			output: []*models.TreeFile{
				{Path: "a", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
			},
		},
		{
			// with -z, git doesn't quote paths, so the path is everything
			// after the first tab, including any further tabs
			testName: "unicode and tabs in path",
			input: "100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\tdïr/日本語 ファイル.txt\x00" +
				"100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\twith\ttab\x00",
			output: []*models.TreeFile{
				{Path: "dïr/日本語 ファイル.txt", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
				{Path: "with\ttab", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
			},
		},
		{
			testName: "directory entry from a non-recursive listing",
			input:    "040000 tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\tdir\x00",
			output: []*models.TreeFile{
				{Path: "dir", BlobHash: "4b825dc642cb6eb9a060e54bf8d69288fbee4904", Mode: 0o040000},
			},
		},
		{
			testName:    "entry without any space",
			input:       "100644\x00",
			expectedErr: "malformed ls-tree entry: \"100644\"",
		},
		{
			testName:    "entry without a space after the type",
			input:       "100644 blob\tfile\x00",
			expectedErr: "malformed ls-tree entry: \"100644 blob\\tfile\"",
		},
		{
			testName:    "non-octal mode",
			input:       "100948 blob deadbeef\tfile\x00",
			expectedErr: "malformed ls-tree entry: \"100948 blob deadbeef\\tfile\"",
		},
		{
			testName:    "malformed entry after valid ones",
			input:       "100644 blob deadbeef\tok\x00garbage\x00",
			expectedErr: "malformed ls-tree entry: \"garbage\"",
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

func TestGetTreeFiles(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"ls-tree", "-z", "-r", "abc123"},
			"100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\tdir/file.txt\x00", nil)
	loader := NewTreeFileLoader(common.NewDummyCommon(), oscommands.NewDummyCmdObjBuilder(runner))

	files, err := loader.GetTreeFiles("abc123")
	assert.NoError(t, err)
	assert.Equal(t, []*models.TreeFile{
		{Path: "dir/file.txt", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
	}, files)
	runner.CheckForMissingCalls()
}

func TestGetTreeFilesError(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"ls-tree", "-z", "-r", "bad"}, "", errors.New("fatal: not a tree object"))
	loader := NewTreeFileLoader(common.NewDummyCommon(), oscommands.NewDummyCmdObjBuilder(runner))

	files, err := loader.GetTreeFiles("bad")
	assert.EqualError(t, err, "fatal: not a tree object")
	assert.Nil(t, files)
	runner.CheckForMissingCalls()
}

func TestGetTreeDirEntries(t *testing.T) {
	// the trailing slash makes ls-tree list the directory's entries rather
	// than the directory entry itself
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"ls-tree", "-z", "abc123", "--", "dir with spaces/"},
			"040000 tree 4b825dc642cb6eb9a060e54bf8d69288fbee4904\tdir with spaces/sub\x00"+
				"100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\tdir with spaces/file.txt\x00", nil)
	loader := NewTreeFileLoader(common.NewDummyCommon(), oscommands.NewDummyCmdObjBuilder(runner))

	entries, err := loader.GetTreeDirEntries("abc123", "dir with spaces")
	assert.NoError(t, err)
	assert.Equal(t, []*models.TreeFile{
		{Path: "dir with spaces/sub", BlobHash: "4b825dc642cb6eb9a060e54bf8d69288fbee4904", Mode: 0o040000},
		{Path: "dir with spaces/file.txt", BlobHash: "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", Mode: 0o100644},
	}, entries)
	assert.True(t, entries[0].IsDir())
	assert.False(t, entries[1].IsDir())
	runner.CheckForMissingCalls()
}

func TestGetTreeDirEntriesError(t *testing.T) {
	runner := oscommands.NewFakeRunner(t).
		ExpectGitArgs([]string{"ls-tree", "-z", "abc123", "--", "dir/"}, "", errors.New("fatal: error"))
	loader := NewTreeFileLoader(common.NewDummyCommon(), oscommands.NewDummyCmdObjBuilder(runner))

	entries, err := loader.GetTreeDirEntries("abc123", "dir")
	assert.EqualError(t, err, "fatal: error")
	assert.Nil(t, entries)
	runner.CheckForMissingCalls()
}

// Exercises the parser at the scale of a very large repository (300k files),
// which is the scale the repo-files panel is designed to handle.
func BenchmarkParseTreeFiles(b *testing.B) {
	var sb strings.Builder
	for i := range 300_000 {
		fmt.Fprintf(&sb, "100644 blob e69de29bb2d1d6434b8b29ae775ad8c2e48c5391\tdir%d/sub%d/file%d.txt\x00",
			i%1000, i%37, i)
	}
	output := sb.String()

	b.ResetTimer()
	for b.Loop() {
		treeFiles, err := parseTreeFiles(output)
		if err != nil || len(treeFiles) != 300_000 {
			b.Fatal("unexpected parse result")
		}
	}
}

func TestTreeFileFlags(t *testing.T) {
	assert.True(t, (&models.TreeFile{Mode: 0o160000}).IsSubmodule())
	assert.True(t, (&models.TreeFile{Mode: 0o120000}).IsSymlink())
	assert.False(t, (&models.TreeFile{Mode: 0o100644}).IsSubmodule())
	assert.False(t, (&models.TreeFile{Mode: 0o100755}).IsSymlink())
	assert.True(t, (&models.TreeFile{Mode: 0o040000}).IsDir())
	assert.False(t, (&models.TreeFile{Mode: 0o100644}).IsDir())
	assert.False(t, (&models.TreeFile{Mode: 0o160000}).IsDir())
	assert.False(t, (&models.TreeFile{Mode: 0o040000}).IsSubmodule())
	assert.False(t, (&models.TreeFile{Mode: 0o040000}).IsSymlink())

	file := &models.TreeFile{Path: "dir/file.txt"}
	assert.Equal(t, "dir/file.txt", file.ID())
	assert.Equal(t, "dir/file.txt", file.Description())
	assert.Equal(t, "dir/file.txt", file.GetPath())
}
