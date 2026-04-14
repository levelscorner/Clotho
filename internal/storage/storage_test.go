package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func newLoc(t *testing.T) Location {
	t.Helper()
	return Location{
		ProjectID:    uuid.New(),
		ProjectSlug:  "demo-project",
		PipelineID:   uuid.New(),
		PipelineSlug: "my-pipeline",
		ExecutionID:  uuid.MustParse("11111111-2222-3333-4444-555555555555"),
		NodeID:       "node-1",
	}
}

func TestLocalStore_WriteHappyPath(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)

	loc := newLoc(t)
	rel, abs, err := s.Write(context.Background(), loc, "image-abc.png", []byte("png-bytes"))
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	wantRel := "demo-project/my-pipeline/" + loc.ExecutionID.String() + "/image-abc.png"
	if rel != wantRel {
		t.Errorf("rel = %q, want %q", rel, wantRel)
	}
	wantAbs := filepath.Join(base, "demo-project", "my-pipeline", loc.ExecutionID.String(), "image-abc.png")
	if abs != wantAbs {
		t.Errorf("abs = %q, want %q", abs, wantAbs)
	}

	got, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if !bytes.Equal(got, []byte("png-bytes")) {
		t.Errorf("file contents = %q, want %q", got, "png-bytes")
	}
}

func TestLocalStore_WriteFallsBackToUnsorted(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)

	cases := []struct {
		name string
		loc  Location
	}{
		{
			name: "missing project slug",
			loc:  Location{PipelineSlug: "p", ExecutionID: uuid.New()},
		},
		{
			name: "missing pipeline slug",
			loc:  Location{ProjectSlug: "proj", ExecutionID: uuid.New()},
		},
		{
			name: "nil execution id",
			loc:  Location{ProjectSlug: "proj", PipelineSlug: "p"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rel, _, err := s.Write(context.Background(), tc.loc, "out.bin", []byte("x"))
			if err != nil {
				t.Fatalf("Write returned error: %v", err)
			}
			if !strings.HasPrefix(rel, "unsorted/") {
				t.Errorf("rel = %q, want unsorted/ prefix", rel)
			}
		})
	}
}

func TestLocalStore_WriteRejectsUnsafeFilename(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)
	loc := newLoc(t)

	cases := []struct {
		name     string
		filename string
	}{
		{"leading dot", ".hidden"},
		{"path separator slash", "foo/bar.png"},
		{"path separator backslash", `foo\bar.png`},
		{"traversal segment", "..png"},
		{"double dot", ".."},
		{"embedded traversal", "foo..bar"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := s.Write(context.Background(), loc, tc.filename, []byte("x"))
			if !errors.Is(err, ErrInvalidPath) {
				t.Errorf("err = %v, want ErrInvalidPath", err)
			}
		})
	}
}

func TestLocalStore_ReadRoundTrip(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)
	loc := newLoc(t)

	payload := []byte("round-trip-bytes")
	rel, _, err := s.Write(context.Background(), loc, "note.json", payload)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	rc, ct, err := s.Read(context.Background(), rel)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	t.Cleanup(func() { _ = rc.Close() })

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload = %q, want %q", got, payload)
	}
	if ct != "application/json" {
		t.Errorf("contentType = %q, want application/json", ct)
	}
}

func TestLocalStore_ReadRejectsUnsafePath(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)

	cases := []struct {
		name string
		path string
	}{
		{"traversal", "../etc/passwd"},
		{"mid-traversal", "demo/../../etc/passwd"},
		{"absolute unix", "/etc/passwd"},
		{"empty", ""},
		{"whitespace only", "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := s.Read(context.Background(), tc.path)
			if !errors.Is(err, ErrInvalidPath) {
				t.Errorf("err = %v, want ErrInvalidPath", err)
			}
		})
	}
}

func TestLocalStore_Base(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)
	if got := s.Base(); got != filepath.Clean(base) {
		t.Errorf("Base = %q, want %q", got, filepath.Clean(base))
	}
}

func TestLocalStore_NewLocalResolvesRelative(t *testing.T) {
	t.Parallel()
	s := NewLocal("some/relative/path")
	if !filepath.IsAbs(s.Base()) {
		t.Errorf("Base = %q, want absolute path", s.Base())
	}
}

func TestSlugify(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"Weird Project! With Spaces", "Weird-Project-With-Spaces"},
		{"hello_world", "hello_world"},
		{"already-safe.ext", "already-safe.ext"},
		{"   spaced   ", "spaced"},
		{"!!!???", ""},
		{"", ""},
		{"кириллица", ""},
		{"mix-Ed_123", "mix-Ed_123"},
		{"a//b\\c", "a-b-c"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := Slugify(tc.in)
			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestExtensionForMIME(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"image/png", "png"},
		{"image/jpeg", "jpg"},
		{"audio/mpeg", "mp3"},
		{"audio/mp3", "mp3"},
		{"video/mp4", "mp4"},
		{"application/octet-stream", "bin"},
		{"text/html; charset=utf-8", "bin"},
		{"", "bin"},
		{"  IMAGE/PNG  ", "png"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := ExtensionForMIME(tc.in)
			if got != tc.want {
				t.Errorf("ExtensionForMIME(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestContentTypeForExtension(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"png", "image/png"},
		{".png", "image/png"},
		{"JPG", "image/jpeg"},
		{"mp3", "audio/mpeg"},
		{"mp4", "video/mp4"},
		{"unknown", "application/octet-stream"},
		{"", "application/octet-stream"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := ContentTypeForExtension(tc.in)
			if got != tc.want {
				t.Errorf("ContentTypeForExtension(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLocationContextRoundTrip(t *testing.T) {
	t.Parallel()
	loc := Location{
		ProjectID:    uuid.New(),
		ProjectSlug:  "p",
		PipelineID:   uuid.New(),
		PipelineSlug: "q",
		ExecutionID:  uuid.New(),
		NodeID:       "n1",
	}

	ctx := WithLocation(context.Background(), loc)
	got, ok := LocationFromContext(ctx)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	if got != loc {
		t.Errorf("loc = %+v, want %+v", got, loc)
	}

	// Bare context has no Location.
	if _, ok := LocationFromContext(context.Background()); ok {
		t.Error("ok = true on bare context, want false")
	}

	// Nil context is handled defensively.
	//nolint:staticcheck // intentional nil-context test
	if _, ok := LocationFromContext(nil); ok {
		t.Error("ok = true on nil context, want false")
	}
}

func TestLocalStore_WriteContextCancelled(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	s := NewLocal(base)
	loc := newLoc(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := s.Write(ctx, loc, "f.txt", []byte("x"))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
