package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"

	"dagger/hello-dagger/internal/dagger"
)

type HelloDagger struct{}

// Publish the application container after building and testing it on-the-fly
func (m *HelloDagger) Publish(
	ctx context.Context,
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {
	_, err := m.Test(ctx, source)
	if err != nil {
		return "", err
	}
	return m.Build(source).Publish(ctx, fmt.Sprintf("ttl.sh/hello-dagger-%0.f", math.Floor(rand.Float64()*1000000))) //#nosec
}

// Build the application container
func (m *HelloDagger) Build(
	// +defaultPath="/"
	source *dagger.Directory,
) *dagger.Container {
	build := m.BuildEnv(source).
		WithExec([]string{"npm", "run", "build"}).
		Directory("./dist")
	return dag.Container().
		From("nginx:1.25-alpine").
		WithDirectory("/usr/share/nginx/html", build).
		WithExposedPort(80)
}

func (m *HelloDagger) Test(
	ctx context.Context,
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {
	return m.BuildEnv(source).
		WithExec([]string{"npm", "run", "test:unit", "run"}).
		Stdout(ctx)
}

// Build a development environment
func (m *HelloDagger) BuildEnv(
	// +defaultPath="/"
	source *dagger.Directory,
) *dagger.Container {
	nodeCache := dag.CacheVolume("wtfisthis") // The cache volumes enables caching layers (nice for pip install...).
	return dag.Container().
		From("node:21-slim").
		WithDirectory("/src", source).
		WithMountedCache("/root/.npm", nodeCache).
		WithWorkdir("/src").
		WithExec([]string{"npm", "install"})
}

// A coding agent for developing new features
func (m *HelloDagger) Develop(
	ctx context.Context,
	// Assignment to complete
	assignment string,
	// +defaultPath="/"
	source *dagger.Directory,
) (*dagger.Directory, error) {

	// Environment with agent inputs and outputs
	environment := dag.Env(dagger.EnvOpts{Privileged: true}). // privileged lets the agent use the existing test function
									WithStringInput("assignment", assignment, "the assignment to complete").
									WithWorkspaceInput("workspace", dag.Workspace(source), "the workspace with tools to edit code").
									WithWorkspaceOutput("completed", "the workspace with the completed assignment")

	// Detailed prompts stored in markdown files
	promptFile := dag.CurrentModule().Source().File("prompts/develop.md")

	// Pull it all together to form the agent
	work := dag.LLM().
		WithEnv(environment).
		WithPromptFile(promptFile)

	// Get the output from the agent
	completed := work.
		Env().
		Output("completed").
		AsWorkspace()

	completedDirectory := completed.GetSource().WithoutDirectory("node_modules")

	// Make sure the tests really pass
	_, err := m.Test(ctx, completedDirectory)
	if err != nil {
		return nil, err
	}

	// Return the Directory with the assignment completed
	return completedDirectory, nil
}

// Develop with a GitHub issue as the assignment and open a pull request
func (m *HelloDagger) DevelopIssue(
	ctx context.Context,
	// GitHub token with permissions to write issues and contents
	githubToken *dagger.Secret,
	// GitHub issue number
	issueID int,
	// GitHub repository URL
	repository string,
	// +defaultPath="/"
	source *dagger.Directory,
) (string, error) {
	// Get the GitHub issue
	issueClient := dag.GithubIssue(dagger.GithubIssueOpts{Token: githubToken})
	issue := issueClient.Read(repository, issueID)

	// Get information from the issue
	assignment, err := issue.Body(ctx)
	if err != nil {
		return "", err
	}

	// Solve the issue with the developer agent
	feature, err := m.Develop(ctx, assignment, source)
	if err != nil {
		return "", err
	}

	// Open a pull request
	title, err := issue.Title(ctx)
	if err != nil {
		return "", err
	}
	url, err := issue.URL(ctx)
	if err != nil {
		return "", err
	}
	body := assignment + "\n\nCloses " + url
	pr := issueClient.CreatePullRequest(repository, title, body, feature)

	return pr.URL(ctx)
}
