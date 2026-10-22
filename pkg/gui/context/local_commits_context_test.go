package context

import (
	"testing"
	"time"

	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/style"
	"github.com/jesseduffield/lazygit/pkg/gui/types"
	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestAddCommitDropIndicator(t *testing.T) {
	pendingHeader := &NonModelItem{Index: 0, Content: "pending"}
	commitsHeader := &NonModelItem{Index: 3, Content: "commits"}
	indicator := &commitDropIndicator{insertionIndex: 3}
	spinnerConfig := config.SpinnerConfig{Frames: []string{"one", "two"}, Rate: 100}

	items := addCommitDropIndicator(
		[]*NonModelItem{pendingHeader}, indicator, "drop here", "moving commits here", spinnerConfig, time.UnixMilli(0),
	)
	items = append(items, commitsHeader)

	assert.Equal(t, []*NonModelItem{
		pendingHeader,
		{
			Index:   3,
			Content: style.FgCyan.SetBold().Sprint("━━━━━━ drop here ━━━━━━"),
			Column:  6,
		},
		commitsHeader,
	}, items)
	assert.Equal(t, 6, modelIndexToViewIndex(4, items, 3))
	assert.Equal(t, 3, viewIndexToModelIndex(4, items, 4))
}

func TestAddMovingCommitsIndicator(t *testing.T) {
	items := addCommitDropIndicator(
		nil,
		&commitDropIndicator{insertionIndex: 2, moving: true},
		"drop here",
		"moving commits here",
		config.SpinnerConfig{Frames: []string{"one", "two"}, Rate: 100},
		time.UnixMilli(100),
	)

	assert.Equal(t, []*NonModelItem{
		{
			Index:   2,
			Content: style.FgCyan.SetBold().Sprint("━━━━━━ moving commits here two ━━━━━━"),
			Column:  6,
		},
	}, items)
}

func TestCommitRangeShownInDiff(t *testing.T) {
	hashPool := &utils.StringPool{}
	newer := models.NewCommit(hashPool, models.NewCommitOpts{Hash: "newer"})
	older := models.NewCommit(hashPool, models.NewCommitOpts{Hash: "older"})

	scenarios := []struct {
		name          string
		refRange      *types.RefRange
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "a range is diffed as a whole",
			refRange:      &types.RefRange{From: older, To: newer},
			expectedStart: 1,
			expectedEnd:   3,
		},
		{
			name:          "without a range to diff, only the commit at the cursor is",
			expectedStart: 3,
			expectedEnd:   3,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			startIdx, endIdx := commitRangeShownInDiff(1, 3, 3, s.refRange)
			assert.Equal(t, s.expectedStart, startIdx)
			assert.Equal(t, s.expectedEnd, endIdx)
		})
	}
}

func TestPullRequestDiff(t *testing.T) {
	hashPool := &utils.StringPool{}
	commit := func(hash string, parent string, status models.CommitStatus) *models.Commit {
		return models.NewCommit(hashPool, models.NewCommitOpts{
			Hash:    hash,
			Parents: lo.Ternary(parent == "", []string{}, []string{parent}),
			Status:  status,
		})
	}

	// A stack of two branches, as the panel lists them: newest first. The checked-out
	// branch "upper" has had its tip amended since it was pushed; it is based on
	// "lower", which is based on a commit that is in a main branch already.
	upperAmended := commit("upper-amended", "upper-2", models.StatusUnpushed)
	upper2 := commit("upper-2", "upper-1", models.StatusPushed)
	upper1 := commit("upper-1", "lower-2", models.StatusPushed)
	lower2 := commit("lower-2", "lower-1", models.StatusPushed)
	lower1 := commit("lower-1", "merged", models.StatusPushed)
	merged := commit("merged", "ancient", models.StatusMerged)
	ancient := commit("ancient", "", models.StatusMerged)
	allCommits := []*models.Commit{upperAmended, upper2, upper1, lower2, lower1, merged, ancient}
	elsewhere := commit("elsewhere", "unlisted", models.StatusPushed)

	branches := []*models.Branch{
		{Name: "upper", CommitHash: "upper-amended"},
		{Name: "lower", CommitHash: "lower-2"},
		// A branch without a pull request in the middle of "upper", and one whose
		// pull request was merged.
		{Name: "no-pull-request", CommitHash: "upper-1"},
		{Name: "merged-feature", CommitHash: "merged"},
	}
	pullRequests := map[string]*models.GithubPullRequest{
		"upper":          {Number: 2},
		"lower":          {Number: 1},
		"merged-feature": {Number: 0},
	}

	scenarios := []struct {
		name                string
		commits             []*models.Commit
		startIdx            int
		endIdx              int
		listsNoBranch       bool
		withoutPullRequests bool
		expected            types.PullRequestDiff
	}{
		{
			name:     "a commit of the checked-out branch starts after its parent",
			startIdx: 1,
			endIdx:   1,
			expected: types.PullRequestDiff{Branch: "upper", Commits: []*models.Commit{upper2}, BaseHash: "upper-1"},
		},
		{
			name:     "a range starts after the parent of its oldest commit",
			startIdx: 0,
			endIdx:   1,
			expected: types.PullRequestDiff{
				Branch: "upper", Commits: []*models.Commit{upperAmended, upper2}, BaseHash: "upper-1",
			},
		},
		{
			name:     "a commit of the branch below is in that branch's pull request",
			startIdx: 3,
			endIdx:   3,
			expected: types.PullRequestDiff{Branch: "lower", Commits: []*models.Commit{lower2}, BaseHash: "lower-1"},
		},
		{
			name:     "the first commit of a branch starts where its pull request does",
			startIdx: 4,
			endIdx:   4,
			expected: types.PullRequestDiff{Branch: "lower", Commits: []*models.Commit{lower1}},
		},
		{
			name:     "and so does the first commit of a branch based on another",
			startIdx: 2,
			endIdx:   2,
			expected: types.PullRequestDiff{Branch: "upper", Commits: []*models.Commit{upper1}},
		},
		{
			name:     "so does a range reaching down to it",
			startIdx: 1,
			endIdx:   2,
			expected: types.PullRequestDiff{Branch: "upper", Commits: []*models.Commit{upper2, upper1}},
		},
		{
			name:     "a range reaching down into the branch below spans both",
			startIdx: 2,
			endIdx:   3,
			expected: types.PullRequestDiff{
				Branch: "upper", SpansBranches: true, Commits: []*models.Commit{upper1, lower2},
			},
		},
		{
			name:     "a range down from the head of the branch below is that branch's",
			startIdx: 3,
			endIdx:   4,
			expected: types.PullRequestDiff{Branch: "lower", Commits: []*models.Commit{lower2, lower1}},
		},
		{
			name:     "the head of a branch that is in a main branch already doesn't count",
			startIdx: 5,
			endIdx:   5,
			expected: types.PullRequestDiff{Branch: "lower", Commits: []*models.Commit{merged}},
		},
		{
			name:     "the first commit of the repository starts where the pull request does",
			startIdx: 6,
			endIdx:   6,
			expected: types.PullRequestDiff{Branch: "lower", Commits: []*models.Commit{ancient}},
		},
		{
			name:     "a parent the panel doesn't list is none of the pull request's",
			commits:  []*models.Commit{elsewhere},
			startIdx: 0,
			endIdx:   0,
			expected: types.PullRequestDiff{Branch: "upper", Commits: []*models.Commit{elsewhere}},
		},
		{
			name:                "without any pull request, a commit is the listed branch's",
			startIdx:            3,
			endIdx:              3,
			withoutPullRequests: true,
			expected:            types.PullRequestDiff{Branch: "upper", Commits: []*models.Commit{lower2}, BaseHash: "lower-1"},
		},
		{
			name:          "a panel listing no local branch's commits has no pull request",
			startIdx:      1,
			endIdx:        1,
			listsNoBranch: true,
		},
		{
			name:     "nothing is shown",
			commits:  []*models.Commit{},
			startIdx: -1,
			endIdx:   -1,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			commits := lo.Ternary(s.commits == nil, allCommits, s.commits)
			listedBranch := lo.Ternary(s.listsNoBranch, "", "upper")
			prs := lo.Ternary(s.withoutPullRequests, nil, pullRequests)
			assert.Equal(t, s.expected, pullRequestDiff(commits, s.startIdx, s.endIdx, listedBranch, branches, prs))
		})
	}
}

// overviewModelGuiCommon serves just the model, which is all that
// overviewDisplayIndices reads.
type overviewModelGuiCommon struct {
	types.IGuiCommon
	model *types.Model
}

func (self *overviewModelGuiCommon) Model() *types.Model {
	return self.model
}

func TestOverviewDisplayIndices(t *testing.T) {
	hashPool := &utils.StringPool{}
	source := lo.Map([]string{"a", "b", "c", "d", "e", "f"}, func(hash string, _ int) *models.Commit {
		return models.NewCommit(hashPool, models.NewCommitOpts{Hash: hash})
	})
	// the overview shows a, c, d, and f
	displayed := []*models.Commit{source[0], source[2], source[3], source[5]}
	indices := []int{0, 2, 3, 5}

	scenarios := []struct {
		name      string
		displayed []*models.Commit
		source    []*models.Commit
		indices   []int
		startIdx  int
		endIdx    int
		expected  []int
	}{
		{
			name:      "overview mode is off",
			displayed: source,
			startIdx:  0,
			endIdx:    6,
			expected:  nil,
		},
		{
			name:      "all displayed rows",
			displayed: displayed,
			source:    source,
			indices:   indices,
			startIdx:  0,
			endIdx:    4,
			expected:  []int{0, 2, 3, 5},
		},
		{
			name:      "a window of the displayed rows",
			displayed: displayed,
			source:    source,
			indices:   indices,
			startIdx:  1,
			endIdx:    3,
			expected:  []int{2, 3},
		},
		{
			name:      "a window reaching past the end of the displayed rows",
			displayed: displayed,
			source:    source,
			indices:   indices,
			startIdx:  2,
			endIdx:    10,
			expected:  []int{3, 5},
		},
		{
			name:      "a window starting past the end of the displayed rows",
			displayed: displayed,
			source:    source,
			indices:   indices,
			startIdx:  4,
			endIdx:    10,
			expected:  []int{},
		},
		{
			name:      "nothing is displayed",
			displayed: []*models.Commit{},
			source:    source,
			indices:   []int{},
			startIdx:  0,
			endIdx:    10,
			expected:  []int{},
		},
		{
			name:      "the displayed rows were replaced by a list of a different length",
			displayed: source[:3],
			source:    source,
			indices:   indices,
			startIdx:  0,
			endIdx:    3,
			expected:  nil,
		},
		{
			name:      "the displayed rows were replaced by different commits",
			displayed: []*models.Commit{source[0], source[1], source[3], source[5]},
			source:    source,
			indices:   indices,
			startIdx:  0,
			endIdx:    4,
			expected:  nil,
		},
		{
			name:      "an index points past the end of the full list",
			displayed: displayed,
			source:    source[:5],
			indices:   indices,
			startIdx:  0,
			endIdx:    4,
			expected:  nil,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			c := &ContextCommon{IGuiCommon: &overviewModelGuiCommon{model: &types.Model{
				Commits:               s.displayed,
				OverviewCommitsSource: s.source,
				OverviewCommitIndices: s.indices,
			}}}

			assert.Equal(t, s.expected, overviewDisplayIndices(c, s.startIdx, s.endIdx))
		})
	}
}

func TestLocalCommitsPullRequestDiffInOverviewMode(t *testing.T) {
	hashPool := &utils.StringPool{}
	commit := func(hash string, parent string) *models.Commit {
		return models.NewCommit(hashPool, models.NewCommitOpts{
			Hash: hash, Parents: []string{parent}, Status: models.StatusPushed,
		})
	}

	top := commit("top", "hidden")
	hidden := commit("hidden", "tagged")
	tagged := commit("tagged", "below")
	below := commit("below", "root")

	// The overview shows top and tagged. The diff of the range between them
	// takes in the hidden commit too, and starts after tagged's parent.
	c := &ContextCommon{IGuiCommon: &overviewModelGuiCommon{model: &types.Model{
		Commits:               []*models.Commit{top, tagged},
		OverviewCommitsSource: []*models.Commit{top, hidden, tagged, below},
		OverviewCommitIndices: []int{0, 2},
		CheckedOutBranch:      "feature",
		Branches:              []*models.Branch{{Name: "feature", CommitHash: "top"}},
		PullRequestsMap:       map[string]*models.GithubPullRequest{"feature": {Number: 1}},
	}}}

	assert.Equal(t,
		/* EXPECTED:
		types.PullRequestDiff{Branch: "feature", Commits: []*models.Commit{top, hidden, tagged}, BaseHash: "below"},
		ACTUAL: */
		types.PullRequestDiff{Branch: "feature", Commits: []*models.Commit{top, tagged}},
		localCommitsPullRequestDiff(c, 0, 1))
}
