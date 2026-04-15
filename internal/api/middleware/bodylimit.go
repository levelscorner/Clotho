package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Body size caps — intentionally tight by default. Each number is what we
// reasonably expect a benign client to send; anything larger is a
// misconfiguration or an attack.
const (
	// DefaultMaxBodyBytes covers JSON mutations (project, pipeline
	// metadata, credential save). 1 MB is well above any hand-built
	// payload and trims the abuse surface down to "nothing weird here".
	DefaultMaxBodyBytes int64 = 1 << 20 // 1 MB

	// PipelineImportMaxBodyBytes covers /api/pipelines/{id}/import. A real
	// exported pipeline tops out at maybe 200 KB for a 50-node graph with
	// long prompts; 10 MB is the ceiling beyond which we assume abuse.
	PipelineImportMaxBodyBytes int64 = 10 << 20 // 10 MB

	// ExecuteMaxBodyBytes is for POST /api/pipelines/{id}/execute, which
	// only carries {from_node_id?: string}.
	ExecuteMaxBodyBytes int64 = 64 << 10 // 64 KB
)

// BodyLimit returns middleware that caps the request body at max bytes.
// Oversize bodies surface as a 413 response; handlers see a MaxBytesError
// when they read past the limit, so they can distinguish "too big" from
// "malformed" cleanly.
//
// Applied as a per-group middleware — each route group picks its own cap.
func BodyLimit(max int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, max)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WriteMaxBytesError translates a *http.MaxBytesError into a 413. Helpers
// that json-decode request bodies call this before returning their generic
// 400 so oversize clients get a truthful status code.
func WriteMaxBytesError(w http.ResponseWriter, err error) bool {
	var mbErr *http.MaxBytesError
	if !errors.As(err, &mbErr) {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "request body too large",
	})
	return true
}
