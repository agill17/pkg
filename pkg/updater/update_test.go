package updater

import (
	"context"
	"errors"
	"testing"

	"github.com/gitops-tools/common/pkg/client/mock"
	"github.com/jenkins-x/go-scm/scm"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	testQuayRepo   = "mynamespace/repository"
	testGitHubRepo = "testorg/testrepo"
	testFilePath   = "environments/test/services/service-a/test.yaml"
	testBranch     = "main"
)

func TestUpdate(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar()

	updater := New(logger, m, NameGenerator(stubNameGenerator{"a"}))
	input := makeInput()
	pr, err := updater.UpdateYAML(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := "test:\n  image: test/my-test-image\n"
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertPullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title: input.PullRequest.Title,
		Body:  input.PullRequest.Body,
		Head:  "test-branch-a",
		Base:  testBranch,
	})
	if pr.Link != "https://example.com/pull-request/1" {
		t.Fatalf("link to PR is incorrect: got %#v, want %#v", pr.Link, "https://example.com/pull-request/1")

	}
}

func TestUpdaterWithMissingFile(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar()

	updater := New(logger, m, NameGenerator(stubNameGenerator{"a"}))
	testErr := errors.New("missing file")
	m.GetFileErr = testErr

	_, err := updater.UpdateYAML(context.Background(), makeInput())

	if err != testErr {
		t.Fatalf("got %s, want %s", err, testErr)
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithBranchCreationFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar()

	updater := New(logger, m, NameGenerator(stubNameGenerator{"a"}))
	testErr := errors.New("can't create branch")
	m.CreateBranchErr = testErr

	_, err := updater.UpdateYAML(context.Background(), makeInput())

	if err.Error() != "failed to create branch: can't create branch" {
		t.Fatalf("got %s, want %s", err, "failed to create branch: can't create branch")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertNoBranchesCreated()
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithUpdateFileFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar()

	updater := New(logger, m, NameGenerator(stubNameGenerator{"a"}))
	testErr := errors.New("can't update file")
	m.UpdateFileErr = testErr
	input := makeInput()

	_, err := updater.UpdateYAML(context.Background(), input)

	if err.Error() != "failed to update file: can't update file" {
		t.Fatalf("got %s, want %s", err, "failed to update file: can't update file")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	if s := string(updated); s != "" {
		t.Fatalf("update failed, got %#v, want %#v", s, "")
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithCreatePullRequestFailure(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, testBranch, []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, testBranch, testSHA)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar()

	updater := New(logger, m, NameGenerator(stubNameGenerator{"a"}))
	testErr := errors.New("can't create pull-request")
	m.CreatePullRequestErr = testErr
	input := makeInput()

	_, err := updater.UpdateYAML(context.Background(), input)

	if err.Error() != "failed to create a pull request: can't create pull-request" {
		t.Fatalf("got %s, want %s", err, "failed to create a pull request: can't create pull-request")
	}
	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := "test:\n  image: test/my-test-image\n"
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertNoPullRequestsCreated()
}

func TestUpdaterWithNonMasterSourceBranch(t *testing.T) {
	testSHA := "980a0d5f19a64b4b30a87d4206aade58726b60e3"
	m := mock.New(t)
	m.AddFileContents(testGitHubRepo, testFilePath, "staging", []byte("test:\n  image: old-image\n"))
	m.AddBranchHead(testGitHubRepo, "staging", testSHA)
	logger := zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar()

	input := makeInput()
	input.Branch = "staging"
	updater := New(logger, m, NameGenerator(stubNameGenerator{"a"}))

	_, err := updater.UpdateYAML(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	updated := m.GetUpdatedContents(testGitHubRepo, testFilePath, "test-branch-a")
	want := "test:\n  image: test/my-test-image\n"
	if s := string(updated); s != want {
		t.Fatalf("update failed, got %#v, want %#v", s, want)
	}
	m.AssertBranchCreated(testGitHubRepo, "test-branch-a", testSHA)
	m.AssertPullRequestCreated(testGitHubRepo, &scm.PullRequestInput{
		Title: input.PullRequest.Title,
		Body:  input.PullRequest.Body,
		Head:  "test-branch-a",
		Base:  "staging",
	})
}

type stubNameGenerator struct {
	name string
}

func (s stubNameGenerator) PrefixedName(p string) string {
	return p + s.name
}

func makeInput() *Input {
	return &Input{
		Repo:               testGitHubRepo,
		Filename:           testFilePath,
		Branch:             testBranch,
		Key:                "test.image",
		NewValue:           "test/my-test-image",
		BranchGenerateName: "test-branch-",
		CommitMessage:      "just a test commit",
		PullRequest: PullRequestInput{
			Title: "test pull-request",
			Body:  "test pull-request body",
		},
	}
}
