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

func (f *TreeFile) IsSymlink() bool {
	return f.Mode == treeFileModeSymlink
}
