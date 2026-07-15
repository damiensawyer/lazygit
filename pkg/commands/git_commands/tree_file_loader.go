package git_commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/commands/oscommands"
	"github.com/jesseduffield/lazygit/pkg/common"
)

type TreeFileLoader struct {
	*common.Common
	cmd oscommands.ICmdObjBuilder
}

func NewTreeFileLoader(common *common.Common, cmd oscommands.ICmdObjBuilder) *TreeFileLoader {
	return &TreeFileLoader{
		Common: common,
		cmd:    cmd,
	}
}

// GetTreeFiles returns all files in the tree of the given ref (usually a
// commit hash), i.e. the full state of the repository at that ref.
func (self *TreeFileLoader) GetTreeFiles(ref string) ([]*models.TreeFile, error) {
	cmdArgs := NewGitCmd("ls-tree").
		Arg("-z").
		Arg("-r").
		Arg(ref).
		ToArgv()

	output, err := self.cmd.New(cmdArgs).DontLog().RunWithOutput()
	if err != nil {
		return nil, err
	}

	return parseTreeFiles(output)
}

// Parses the output of `git ls-tree -z -r`, i.e. NUL-separated entries of the
// form "<mode> <type> <hash>\t<path>". Trees this large can have hundreds of
// thousands of entries, so we scan the string by hand rather than using a
// regex or strings.Split.
func parseTreeFiles(output string) ([]*models.TreeFile, error) {
	treeFiles := make([]*models.TreeFile, 0, strings.Count(output, "\x00"))

	for len(output) > 0 {
		entry := output
		if nul := strings.IndexByte(output, '\x00'); nul >= 0 {
			entry = output[:nul]
			output = output[nul+1:]
		} else {
			output = ""
		}
		if entry == "" {
			continue
		}

		treeFile, err := parseTreeEntry(entry)
		if err != nil {
			return nil, err
		}
		treeFiles = append(treeFiles, treeFile)
	}

	return treeFiles, nil
}

func parseTreeEntry(entry string) (*models.TreeFile, error) {
	modeEnd := strings.IndexByte(entry, ' ')
	if modeEnd < 0 {
		return nil, fmt.Errorf("malformed ls-tree entry: %q", entry)
	}
	typeEnd := strings.IndexByte(entry[modeEnd+1:], ' ')
	if typeEnd < 0 {
		return nil, fmt.Errorf("malformed ls-tree entry: %q", entry)
	}
	typeEnd += modeEnd + 1
	hashEnd := strings.IndexByte(entry[typeEnd+1:], '\t')
	if hashEnd < 0 {
		return nil, fmt.Errorf("malformed ls-tree entry: %q", entry)
	}
	hashEnd += typeEnd + 1

	mode, err := strconv.ParseInt(entry[:modeEnd], 8, 32)
	if err != nil {
		return nil, fmt.Errorf("malformed ls-tree entry: %q", entry)
	}

	return &models.TreeFile{
		Path:     entry[hashEnd+1:],
		BlobHash: entry[typeEnd+1 : hashEnd],
		Mode:     int32(mode),
	}, nil
}
