package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/user/clotho/internal/storage"
)

// mockStore is a minimal storage.Store for handler tests. It only needs to
// satisfy Read + Base; Write is unused here but must exist to satisfy the
// interface contract.
type mockStore struct {
	readFn  func(ctx context.Context, rel string) (io.ReadCloser, string, error)
	baseDir string
}

func (m *mockStore) Write(_ context.Context, _ storage.Location, _ string, _ []byte) (string, string, error) {
	return "", "", errors.New("not implemented")
}

func (m *mockStore) Read(ctx context.Context, rel string) (io.ReadCloser, string, error) {
	return m.readFn(ctx, rel)
}

func (m *mockStore) Base() string { return m.baseDir }

// newRouter mounts the FilesHandler routes for a given store. The route
// pattern mirrors what Wave 4 will wire into the real router.
func newRouter(store storage.Store) chi.Router {
	r := chi.NewRouter()
	NewFilesHandler(store).Routes(r)
	return r
}

func TestFilesHandler_Get_Happy(t *testing.T) {
	want := []byte("PNG-BYTES")
	store := &mockStore{
		readFn: func(_ context.Context, rel string) (io.ReadCloser, string, error) {
			if rel != "projects/p1/pipe/exec/img.png" {
				t.Errorf("unexpected rel path: %q", rel)
			}
			return io.NopCloser(bytes.NewReader(want)), "image/png", nil
		},
	}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/files/projects/p1/pipe/exec/img.png", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=3600" {
		t.Errorf("Cache-Control = %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), want) {
		t.Errorf("body = %q, want %q", rec.Body.Bytes(), want)
	}
}

func TestFilesHandler_Get_TraversalRejected(t *testing.T) {
	// Store should never be called — handler rejects before reaching storage.
	store := &mockStore{
		readFn: func(context.Context, string) (io.ReadCloser, string, error) {
			t.Fatalf("store.Read should not be called on traversal")
			return nil, "", nil
		},
	}

	// chi cleans the URL path before matching routes, so embedding `..` in
	// the request URL usually gets normalised away. Drive the handler
	// directly with a chi route context that pins the wildcard value to a
	// traversal string — this is the exact shape the handler sees after
	// the router extracts the wildcard.
	traversals := []string{
		"../secret",
		"a/../../b",
		"/etc/passwd",
	}
	for _, tp := range traversals {
		t.Run(tp, func(t *testing.T) {
			h := NewFilesHandler(store)
			req := httptest.NewRequest(http.MethodGet, "/api/files/x", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("*", tp)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()
			h.Get(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestFilesHandler_Get_MissingPath(t *testing.T) {
	store := &mockStore{
		readFn: func(context.Context, string) (io.ReadCloser, string, error) {
			return nil, "", nil
		},
	}
	r := newRouter(store)

	// No wildcard content — chi's `*` match yields empty string.
	req := httptest.NewRequest(http.MethodGet, "/api/files/", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestFilesHandler_Get_NotFound(t *testing.T) {
	store := &mockStore{
		readFn: func(context.Context, string) (io.ReadCloser, string, error) {
			return nil, "", os.ErrNotExist
		},
	}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/files/missing.png", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestFilesHandler_Get_StorageInvalidPath(t *testing.T) {
	store := &mockStore{
		readFn: func(context.Context, string) (io.ReadCloser, string, error) {
			return nil, "", storage.ErrInvalidPath
		},
	}
	r := newRouter(store)

	// Handler lets the path through (no .. / leading /), but storage rejects.
	req := httptest.NewRequest(http.MethodGet, "/api/files/ok/path.png", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestFilesHandler_Get_GenericError(t *testing.T) {
	store := &mockStore{
		readFn: func(context.Context, string) (io.ReadCloser, string, error) {
			return nil, "", errors.New("disk on fire")
		},
	}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/api/files/x.png", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestFilesHandler_Reveal_InvalidBody(t *testing.T) {
	store := &mockStore{baseDir: "/tmp/clotho-test"}
	r := newRouter(store)

	cases := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"malformed JSON", "{not json"},
		{"empty path", `{"path":""}`},
		{"absolute path", `{"path":"/etc/passwd"}`},
		{"traversal", `{"path":"../../etc/passwd"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/files/reveal",
				strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestFilesHandler_Reveal_Darwin(t *testing.T) {
	// Capture the command args without actually spawning `open`.
	var captured []string
	origExec := execCommand
	origGOOS := runtimeGOOS
	t.Cleanup(func() {
		execCommand = origExec
		runtimeGOOS = origGOOS
	})
	execCommand = func(name string, args ...string) *exec.Cmd {
		captured = append([]string{name}, args...)
		// `true` exits 0 on macOS/Linux. On Windows this test is skipped below.
		return exec.Command("true")
	}
	runtimeGOOS = func() string { return "darwin" }

	store := &mockStore{baseDir: "/tmp/clotho-test"}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/api/files/reveal",
		strings.NewReader(`{"path":"projects/p1/pipe/exec/img.png"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if len(captured) == 0 || captured[0] != "open" {
		t.Fatalf("expected `open` command, got %v", captured)
	}
	if len(captured) < 3 || captured[1] != "-R" {
		t.Fatalf("expected `-R` flag, got %v", captured)
	}
	if !strings.HasSuffix(captured[2], "/img.png") {
		t.Errorf("expected abs path to end in /img.png, got %q", captured[2])
	}
	if !strings.HasPrefix(captured[2], "/tmp/clotho-test") {
		t.Errorf("expected abs path rooted at base, got %q", captured[2])
	}
}

func TestFilesHandler_Reveal_Linux(t *testing.T) {
	var captured []string
	origExec := execCommand
	origGOOS := runtimeGOOS
	t.Cleanup(func() {
		execCommand = origExec
		runtimeGOOS = origGOOS
	})
	execCommand = func(name string, args ...string) *exec.Cmd {
		captured = append([]string{name}, args...)
		return exec.Command("true")
	}
	runtimeGOOS = func() string { return "linux" }

	store := &mockStore{baseDir: "/tmp/clotho-test"}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/api/files/reveal",
		strings.NewReader(`{"path":"a/b/c.png"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if captured[0] != "xdg-open" {
		t.Fatalf("expected xdg-open, got %v", captured)
	}
	// xdg-open opens the parent directory, not the file itself.
	if strings.HasSuffix(captured[1], "c.png") {
		t.Errorf("expected parent dir, got %q", captured[1])
	}
}

func TestFilesHandler_Reveal_UnsupportedPlatform(t *testing.T) {
	origGOOS := runtimeGOOS
	t.Cleanup(func() { runtimeGOOS = origGOOS })
	runtimeGOOS = func() string { return "plan9" }

	store := &mockStore{baseDir: "/tmp/clotho-test"}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/api/files/reveal",
		strings.NewReader(`{"path":"a/b/c.png"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
}

func TestFilesHandler_Reveal_SpawnFailure(t *testing.T) {
	origExec := execCommand
	origGOOS := runtimeGOOS
	t.Cleanup(func() {
		execCommand = origExec
		runtimeGOOS = origGOOS
	})
	execCommand = func(name string, args ...string) *exec.Cmd {
		// Point to a definitely-missing binary so Start() fails.
		return exec.Command("/nonexistent/clotho-test-binary-xyz")
	}
	runtimeGOOS = func() string { return "darwin" }

	store := &mockStore{baseDir: "/tmp/clotho-test"}
	r := newRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/api/files/reveal",
		strings.NewReader(`{"path":"a/b/c.png"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}
