package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/user/clotho/internal/storage"
)

// execCommand is a package-level hook so tests can substitute the OS-level
// process spawn without actually opening Finder / Explorer / xdg-open.
var execCommand = exec.Command

// runtimeGOOS is a package-level hook so tests can pin the platform branch in
// the Reveal handler without relying on the host OS.
var runtimeGOOS = func() string { return runtime.GOOS }

// FilesHandler serves pipeline output files written by the storage layer
// and, on macOS (plus best-effort for Windows/Linux), can open their
// containing folder in the host file manager.
type FilesHandler struct {
	store storage.Store
}

// NewFilesHandler constructs a FilesHandler backed by the given storage.Store.
// The store is expected to be a LocalStore in dev/prod; any Store
// implementation that honours the Read + Base contract will work.
func NewFilesHandler(store storage.Store) *FilesHandler {
	return &FilesHandler{store: store}
}

// Routes registers:
//
//	GET  /api/files/*        — serve an on-disk artifact by relative path
//	POST /api/files/reveal   — reveal the given relative path in the host file manager
//
// Wave 4 of the sprint mounts these inside the protected route group.
func (h *FilesHandler) Routes(r chi.Router) {
	r.Post("/api/files/reveal", h.Reveal)
	r.Get("/api/files/*", h.Get)
}

// Get streams an artifact file to the client. The relative path is taken from
// chi's wildcard match. Path traversal is rejected with 400; missing files
// return 404. All successful responses carry a conservative private cache
// header and X-Content-Type-Options: nosniff.
func (h *FilesHandler) Get(w http.ResponseWriter, r *http.Request) {
	rel := chi.URLParam(r, "*")
	if rel == "" {
		writeError(w, http.StatusBadRequest, "missing path")
		return
	}
	if !isSafeRelPath(rel) {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	content, contentType, err := h.store.Read(r.Context(), rel)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrInvalidPath):
			writeError(w, http.StatusBadRequest, "invalid path")
		case errors.Is(err, os.ErrNotExist):
			writeError(w, http.StatusNotFound, "not found")
		default:
			writeError(w, http.StatusInternalServerError, "server error")
		}
		return
	}
	defer content.Close()

	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Body stream. io.Copy handles partial writes internally; any mid-stream
	// error is already too late to send a different status.
	_, _ = io.Copy(w, content)
}

// revealRequest is the JSON body for POST /api/files/reveal.
type revealRequest struct {
	Path string `json:"path"`
}

// Reveal tells the host OS to open the containing directory of the given
// relative path. macOS: `open -R <abs>` selects the file in Finder.
// Windows: `explorer /select,<abs>`. Linux: `xdg-open <dir>` (no "select"
// equivalent — opens the parent dir). Returns 501 on any other platform.
//
// Security: the request body is rejected on traversal / absolute paths.
// The absolute path is constructed by joining Store.Base() with the cleaned
// relative path; the storage package's Read flow uses the same validation
// for its own safety, but we double-check here because Reveal does not call
// Read.
func (h *FilesHandler) Reveal(w http.ResponseWriter, r *http.Request) {
	var body revealRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if body.Path == "" {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if !isSafeRelPath(body.Path) {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}

	abs := filepath.Join(h.store.Base(), filepath.FromSlash(body.Path))

	var cmd *exec.Cmd
	switch runtimeGOOS() {
	case "darwin":
		cmd = execCommand("open", "-R", abs)
	case "windows":
		cmd = execCommand("explorer", "/select,", abs)
	case "linux":
		// xdg-open lacks a select-file mode; open the containing directory.
		cmd = execCommand("xdg-open", filepath.Dir(abs))
	default:
		writeError(w, http.StatusNotImplemented, "reveal not supported on this platform")
		return
	}

	if err := cmd.Start(); err != nil {
		writeError(w, http.StatusInternalServerError, "reveal failed: "+err.Error())
		return
	}
	// Fire-and-forget: release the child so we don't accumulate zombies.
	// We deliberately don't Wait — the UI only cares that the file manager
	// was asked to open.
	go func() { _ = cmd.Wait() }()

	w.WriteHeader(http.StatusNoContent)
}

// isSafeRelPath rejects absolute paths and any traversal segments. The
// storage package performs its own validation; this is a fast-path so
// obvious attacks return 400 before touching disk.
//
// Defense in depth:
//   - Rejects absolute paths on Unix and Windows (including drive-letter
//     forms like "C:foo" that filepath.IsAbs returns false for on Unix).
//   - Rejects any ".." segment and NUL bytes (truncation attacks).
//   - Rejects sub-0x20 control characters so log-injection CRLF can't
//     smuggle through as a "safe" path segment.
func isSafeRelPath(p string) bool {
	if p == "" {
		return false
	}
	if strings.ContainsRune(p, 0) {
		return false
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) {
		return false
	}
	if filepath.IsAbs(p) {
		return false
	}
	if len(p) >= 2 && p[1] == ':' &&
		((p[0] >= 'A' && p[0] <= 'Z') || (p[0] >= 'a' && p[0] <= 'z')) {
		return false
	}
	for _, r := range p {
		if r < 0x20 {
			return false
		}
	}
	// Normalise and scan segments.
	for _, seg := range strings.Split(filepath.ToSlash(p), "/") {
		if seg == ".." {
			return false
		}
	}
	return true
}
