package pull_requests

import (
	"github.com/jesseduffield/lazygit/pkg/commands/models"
	"github.com/jesseduffield/lazygit/pkg/config"
	. "github.com/jesseduffield/lazygit/pkg/integration/components"
)

var CommentOnDiffLines = NewIntegrationTest(NewIntegrationTestArgs{
	Description:  "Comment on a range of lines of a pull request's diff, starting a pending review, and then submit the review",
	ExtraCmdArgs: []string{},
	Skip:         false,
	SetupConfig: func(config *config.AppConfig) {
		config.GetUserConfig().Gui.UseHunkModeInDiffView = false
	},
	SetupRepo: func(shell *Shell) {
		setupRepoWithGithubRemote(shell)
		shell.NewBranch("contribution").
			CreateFileAndAdd("widget.go", "package widget\n\nfunc New() *Widget {\n\treturn nil\n}\n").
			Commit("add a widget").
			RunCommand([]string{"git", "push", "origin", "contribution:refs/pull/12/head"}).
			Checkout("master").
			RunCommand([]string{"git", "branch", "-D", "contribution"})
	},
	FakeGithub: &FakeGithub{
		PullRequests: []*models.GithubPullRequest{pullRequest(12, "Add a widget", "octocat", "OPEN", "widget")},
	},
	Run: func(t *TestDriver, keys config.KeybindingConfig) {
		t.Shell().RunCommand([]string{"git", "fetch", "origin", "refs/pull/12/head:refs/pull/12/head"})
		prHead := t.Git().GetCommitHash("refs/pull/12/head")
		t.FakeGithub().Update(func(fake *FakeGithub) {
			fake.PullRequests[0].HeadRefOid = prHead
		})

		t.Views().PullRequests().
			Focus().
			Press(keys.Universal.Refresh).
			PressEnter()

		t.Views().SubCommits().
			IsFocused().
			Lines(
				Contains("add a widget").IsSelected(),
				Contains("initial commit"),
			).
			Press(keys.Universal.FocusMainView)

		t.Views().Main().
			IsFocused().
			NavigateToLine(Contains("+func New() *Widget {")).
			Press(keys.Universal.ToggleRangeSelect).
			NavigateToLine(Contains("return nil")).
			Press(keys.Main.AddReviewComment)

		t.ExpectPopup().CommitMessagePanel().
			Title(Equals("Comment on widget.go:3-4")).
			Type("Why a pointer?").
			Confirm()

		t.ExpectToast(Equals("Added the comment to your pending review"))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReviewStart {"input":{"pullRequestId":"PR_12"}}`))
		t.FakeGithub().ReceivedRequest(Equals(`POST /graphql PullRequestReviewThreadAdd {"input":{"body":"Why a pointer?","line":4,"path":"widget.go","pullRequestReviewId":"PRR_1","side":"RIGHT","startLine":3,"startSide":"RIGHT"}}`))
	},
})
