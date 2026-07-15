package models

// TreeFile : an entry in a commit's tree, as listed by `git ls-tree -r`
type TreeFile struct {
	Path string

	// hash of the blob (or of the commit, for a submodule entry)
	BlobHash string

	// file mode as reported by ls-tree, e.g. 0o100644 for a regular file
	Mode int32
}

const (
	treeFileModeDir       int32 = 0o040000
	treeFileModeSymlink   int32 = 0o120000
	treeFileModeSubmodule int32 = 0o160000
)

func (f *TreeFile) ID() string {
	return f.Path
}

func (f *TreeFile) Description() string {
	return f.Path
}

func (f *TreeFile) GetPath() string {
	return f.Path
}

func (f *TreeFile) IsSubmodule() bool {
	return f.Mode == treeFileModeSubmodule
}

// Only directory *entries* (from a non-recursive ls-tree) have this mode; the
// trees making up a recursive listing are never returned as entries themselves.
func (f *TreeFile) IsDir() bool {
	return f.Mode == treeFileModeDir
}

func (f *TreeFile) IsSymlink() bool {
	return f.Mode == treeFileModeSymlink
}
