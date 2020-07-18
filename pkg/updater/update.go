package updater

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gitops-tools/common/pkg/client"
	"github.com/gitops-tools/common/pkg/names"
	"github.com/gitops-tools/common/pkg/syaml"
	"github.com/jenkins-x/go-scm/scm"
)

// Input is the configuration for updating a file in a repository.
type Input struct {
	Repo               string           // e.g. my-org/my-repo
	Filename           string           // relative path to the file in the repository
	Key                string           // key - the key within the YAML file to be updated, use a dotted path
	NewValue           string           // the new value to associate with the key
	Branch             string           // e.g. main
	BranchGenerateName string           // e.g. update-image-
	CommitMessage      string           // This will be used for the commit to update the file
	PullRequest        PullRequestInput // This is used to create the pull request.
}

// PullRequestInput provides configuration for the PullRequest to be opened.
type PullRequestInput struct {
	Title string
	Body  string
}

var timeSeed = rand.New(rand.NewSource(time.Now().UnixNano()))

type logger interface {
	Infow(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Debugw(msg string, keysAndValues ...interface{})
}

type updaterFunc func(u *Updater)

// NameGenerator is an option func for the Updater creation function.
func NameGenerator(g names.Generator) updaterFunc {
	return func(u *Updater) {
		u.nameGenerator = g
	}
}

// New creates and returns a new Updater.
func New(l logger, c client.GitClient, opts ...updaterFunc) *Updater {
	u := &Updater{gitClient: c, nameGenerator: names.New(timeSeed), log: l}
	for _, o := range opts {
		o(u)
	}
	return u
}

// Updater can update a Git repo with an updated version of a file.
type Updater struct {
	gitClient     client.GitClient
	nameGenerator names.Generator
	log           logger
}

// UpdateYAML does the job of fetching the existing file, updating it, and
// then optionally creating a PR.
func (u *Updater) UpdateYAML(ctx context.Context, input *Input) (*scm.PullRequest, error) {
	current, err := u.gitClient.GetFile(ctx, input.Repo, input.Branch, input.Filename)
	if err != nil {
		u.log.Errorw("failed to get file from repo", "error", err)
		return nil, err
	}
	u.log.Debugw("got existing file", "sha", current.Sha)
	updated, err := syaml.SetBytes(current.Data, input.Key, input.NewValue)
	if err != nil {
		return nil, err
	}
	branchRef, err := u.gitClient.GetBranchHead(ctx, input.Repo, input.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch head: %v", err)
	}
	newBranchName, err := u.createBranchIfNecessary(ctx, input, branchRef)
	if err != nil {
		return nil, err
	}
	err = u.gitClient.UpdateFile(ctx, input.Repo, newBranchName, input.Filename, input.CommitMessage, current.Sha, updated)
	if err != nil {
		return nil, fmt.Errorf("failed to update file: %w", err)
	}
	u.log.Debugw("updated file", "filename", input.Filename)
	return u.createPRIfNecessary(ctx, input, newBranchName)
}

func (u *Updater) createBranchIfNecessary(ctx context.Context, input *Input, sourceRef string) (string, error) {
	if input.BranchGenerateName == "" {
		u.log.Debugw("no branchGenerateName configured, reusing source branch", "branch", input.Branch)
		return input.Branch, nil
	}

	newBranchName := u.nameGenerator.PrefixedName(input.BranchGenerateName)
	u.log.Debugw("generating new branch", "name", newBranchName)
	err := u.gitClient.CreateBranch(ctx, input.Repo, newBranchName, sourceRef)
	if err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}
	u.log.Debugw("created branch", "branch", newBranchName, "ref", sourceRef)
	return newBranchName, nil
}

func (u *Updater) createPRIfNecessary(ctx context.Context, input *Input, newBranchName string) (*scm.PullRequest, error) {
	if input.Branch == newBranchName {
		return nil, nil
	}
	pr, err := u.gitClient.CreatePullRequest(ctx, input.Repo, &scm.PullRequestInput{
		Title: input.PullRequest.Title,
		Body:  input.PullRequest.Body,
		Head:  newBranchName,
		Base:  input.Branch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create a pull request: %w", err)
	}
	u.log.Debugw("created PullRequest", "number", pr.Number)
	return pr, nil
}
