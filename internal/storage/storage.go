// Package storage persists generated media artifacts (images, audio, video)
// produced by pipeline executions and serves them back via the Read API.
//
// The package is intentionally small: providers obtain a Location from
// request context (attached by the engine before invoking the provider) and
// call Store.Write with the bytes they produced. A relative path is returned
// which is persisted alongside the StepResult so the file can be served later
// via the files HTTP handler.
package storage

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
)

// Location identifies where a generated artifact belongs. All fields are
// required for a well-formed write; providers obtain the Location from request
// context (see WithLocation / LocationFromContext). Implementations fall back
// to an "unsorted" bucket when fields are missing so a provider cannot panic
// a worker by forgetting to attach context.
type Location struct {
	ProjectID    uuid.UUID
	ProjectSlug  string
	PipelineID   uuid.UUID
	PipelineSlug string
	ExecutionID  uuid.UUID
	NodeID       string
}

// Store persists generated media artifacts and serves them back.
// Implementations may be local filesystem, S3, etc. All methods must be safe
// for concurrent use across goroutines.
type Store interface {
	// Write persists data under {Base()}/{projectSlug}/{pipelineSlug}/{executionID}/{filename}.
	// It returns the relative path (persisted in the DB / served via URL) and the
	// absolute path (for logging / "reveal in Finder"). If the Location is
	// incomplete, the implementation SHOULD fall back to an "unsorted" bucket
	// rather than fail.
	Write(ctx context.Context, loc Location, filename string, data []byte) (rel string, abs string, err error)

	// Read opens the content under relativePath (as returned by Write). The
	// returned ReadCloser MUST be closed by the caller. contentType is
	// best-effort based on the file extension.
	Read(ctx context.Context, relativePath string) (content io.ReadCloser, contentType string, err error)

	// Base returns the configured root directory (for diagnostics /
	// "reveal in Finder" UX).
	Base() string
}

// ErrInvalidPath is returned when a path escapes the storage root, contains
// disallowed characters (e.g. ".." segments, path separators in a filename),
// or is empty.
var ErrInvalidPath = errors.New("storage: invalid path")
