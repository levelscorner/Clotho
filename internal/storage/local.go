package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	dirPerm  os.FileMode = 0o755
	filePerm os.FileMode = 0o644

	unsortedBucket = "unsorted"
)

// LocalStore writes under a base directory on the local filesystem.
// Directories are created lazily with 0o755 and files written with 0o644.
// The zero value is not usable; construct via NewLocal.
type LocalStore struct {
	base string
}

// NewLocal constructs a LocalStore rooted at base. If base is relative it is
// resolved against the current working directory. The base dir is created
// lazily on first Write so construction is side-effect-free.
func NewLocal(base string) *LocalStore {
	abs := base
	if !filepath.IsAbs(abs) {
		if cwd, err := os.Getwd(); err == nil {
			abs = filepath.Join(cwd, abs)
		}
	}
	return &LocalStore{base: filepath.Clean(abs)}
}

// Base returns the absolute root directory.
func (s *LocalStore) Base() string { return s.base }

// Write persists data under {base}/{projectSlug}/{pipelineSlug}/{executionID}/{filename}.
// If loc is incomplete (empty slugs or nil UUIDs) the file is placed under
// {base}/unsorted/{executionID}/{filename} so a broken provider cannot fail
// the whole execution.
func (s *LocalStore) Write(ctx context.Context, loc Location, filename string, data []byte) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if !isSafeFilename(filename) {
		return "", "", fmt.Errorf("%w: filename %q", ErrInvalidPath, filename)
	}

	relDir := resolveRelDir(loc)
	relPath := filepath.ToSlash(filepath.Join(relDir, filename))

	absDir := filepath.Join(s.base, filepath.FromSlash(relDir))
	absPath := filepath.Join(absDir, filename)

	if err := os.MkdirAll(absDir, dirPerm); err != nil {
		return "", "", fmt.Errorf("storage: create dir %q: %w", absDir, err)
	}
	if err := os.WriteFile(absPath, data, filePerm); err != nil {
		return "", "", fmt.Errorf("storage: write file %q: %w", absPath, err)
	}
	return relPath, absPath, nil
}

// Read opens the file at relativePath (as returned by Write). The caller must
// Close the returned ReadCloser. contentType is inferred from the extension.
func (s *LocalStore) Read(ctx context.Context, relativePath string) (io.ReadCloser, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	clean, err := s.safeJoin(relativePath)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(clean)
	if err != nil {
		return nil, "", fmt.Errorf("storage: open %q: %w", relativePath, err)
	}
	ct := ContentTypeForExtension(filepath.Ext(clean))
	return f, ct, nil
}

// safeJoin resolves relativePath against the base directory and rejects any
// path that escapes the root or contains disallowed segments.
func (s *LocalStore) safeJoin(relativePath string) (string, error) {
	rp := strings.TrimSpace(relativePath)
	if rp == "" {
		return "", fmt.Errorf("%w: empty path", ErrInvalidPath)
	}
	// Reject absolute paths and Windows-style drive markers up front.
	if filepath.IsAbs(rp) || strings.HasPrefix(rp, "/") || strings.HasPrefix(rp, `\`) {
		return "", fmt.Errorf("%w: absolute path %q", ErrInvalidPath, relativePath)
	}
	// Normalise separators and reject any traversal segment.
	normalized := filepath.FromSlash(rp)
	for _, seg := range strings.Split(normalized, string(filepath.Separator)) {
		if seg == ".." {
			return "", fmt.Errorf("%w: traversal in %q", ErrInvalidPath, relativePath)
		}
	}
	joined := filepath.Join(s.base, normalized)
	cleaned := filepath.Clean(joined)
	// Defense in depth: final path must stay under base.
	baseWithSep := s.base + string(filepath.Separator)
	if cleaned != s.base && !strings.HasPrefix(cleaned, baseWithSep) {
		return "", fmt.Errorf("%w: escapes base %q", ErrInvalidPath, relativePath)
	}
	return cleaned, nil
}

// RelDir returns the relative directory path the Store uses for the given
// Location — the same path that every file writes resolve against. Callers
// that need to surface "open this folder" in the UI (e.g. the engine
// publishing artifact_dir on execution_completed) use this so the path
// they ship to the frontend exactly matches what ends up on disk.
func RelDir(loc Location) string {
	return resolveRelDir(loc)
}

// resolveRelDir builds the relative directory for a Location, falling back to
// the unsorted bucket when any required field is zero-valued.
func resolveRelDir(loc Location) string {
	projectSlug := Slugify(loc.ProjectSlug)
	pipelineSlug := Slugify(loc.PipelineSlug)
	execID := loc.ExecutionID

	if projectSlug == "" || pipelineSlug == "" || execID == uuid.Nil {
		// Still try to preserve the execution id if we have one — it's the most
		// useful bit for diagnostics.
		id := execID.String()
		if execID == uuid.Nil {
			id = "unknown"
		}
		return filepath.ToSlash(filepath.Join(unsortedBucket, id))
	}
	return filepath.ToSlash(filepath.Join(projectSlug, pipelineSlug, execID.String()))
}
