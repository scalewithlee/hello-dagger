// a module for editing code
package main

import (
	"context"
	"dagger/workspace/internal/dagger"
)

type Workspace struct {
	// The source directory
	Source *dagger.Directory
}

func New(
	// where the codebase is
	source *dagger.Directory,
) *Workspace {
	return &Workspace{Source: source}
}

// Read a file in the workspace
func (w *Workspace) ReadFile(
	ctx context.Context,
	path string,
) (string, error) {
	return w.Source.File(path).Contents(ctx)
}

// Write a file to the workspace
func (w *Workspace) WriteFile(
	path string,
	contents string,
) *Workspace {
	w.Source = w.Source.WithNewFile(path, contents)
	return w
}

// List all files in the workspace
func (w *Workspace) ListFiles(
	ctx context.Context,
) (string, error) {
	return dag.Container().
		From("alpine:3").
		WithDirectory("/src", w.Source).
		WithWorkdir("/src").
		WithExec([]string{"tree", "./src"}).
		Stdout(ctx)
}

// Get the source code directory from the workspace
func (w *Workspace) GetSource() *dagger.Directory {
	return w.Source
}
