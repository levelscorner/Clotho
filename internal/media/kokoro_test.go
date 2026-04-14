package media

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKokoro_Submit_HappyPath(t *testing.T) {
	t.Parallel()

	// Fake Kokoro-FastAPI server — returns a short MP3-like byte sequence.
	fakeMP3 := []byte{0xFF, 0xFB, 0x90, 0x44, 0x00} // MP3 sync bytes + a few bytes

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			http.Error(w, "expected json content type", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeMP3)
	}))
	defer srv.Close()

	k := NewKokoro(srv.URL)
	jobID, err := k.Submit(context.Background(), MediaRequest{
		Prompt: "hello world",
		Voice:  "af_bella",
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if jobID == "" {
		t.Fatal("empty jobID")
	}

	status, err := k.Poll(context.Background(), jobID)
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if status.State != "succeeded" {
		t.Errorf("expected state=succeeded, got %q", status.State)
	}
	if !strings.HasPrefix(status.Output, "data:audio/mp3;base64,") {
		t.Errorf("expected data URI prefix, got %q", status.Output[:40])
	}
	// Decode + compare bytes.
	encoded := strings.TrimPrefix(status.Output, "data:audio/mp3;base64,")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if string(decoded) != string(fakeMP3) {
		t.Errorf("output bytes mismatch: got %v want %v", decoded, fakeMP3)
	}
}

func TestKokoro_Submit_DefaultVoice(t *testing.T) {
	t.Parallel()

	var receivedVoice string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body kokoroSpeechRequest
		_ = decodeJSON(r, &body)
		receivedVoice = body.Voice
		_, _ = w.Write([]byte{0xFF, 0xFB})
	}))
	defer srv.Close()

	k := NewKokoro(srv.URL)
	_, err := k.Submit(context.Background(), MediaRequest{Prompt: "x"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if receivedVoice != "af_bella" {
		t.Errorf("expected default voice af_bella, got %q", receivedVoice)
	}
}

func TestKokoro_Submit_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"detail":"voice unknown: foo"}`))
	}))
	defer srv.Close()

	k := NewKokoro(srv.URL)
	_, err := k.Submit(context.Background(), MediaRequest{Prompt: "x", Voice: "foo"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "voice unknown") {
		t.Errorf("expected error to surface API detail, got: %v", err)
	}
}

func TestKokoro_Submit_Unreachable(t *testing.T) {
	t.Parallel()

	// Use a port that nothing listens on.
	k := NewKokoro("http://127.0.0.1:1")
	_, err := k.Submit(context.Background(), MediaRequest{Prompt: "x"})
	if err == nil {
		t.Fatal("expected error when server unreachable")
	}
	if !strings.Contains(err.Error(), "kokoro:") {
		t.Errorf("expected error namespaced with 'kokoro:', got: %v", err)
	}
}

func TestKokoro_Poll_UnknownJob(t *testing.T) {
	t.Parallel()

	k := NewKokoro("http://example.invalid")
	_, err := k.Poll(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown jobID")
	}
}
