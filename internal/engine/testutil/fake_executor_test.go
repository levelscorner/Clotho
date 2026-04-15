package testutil

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/user/clotho/internal/domain"
)

func TestFakeExecutor_Execute_Success(t *testing.T) {
	t.Parallel()

	out := TextOutputWithCost("hello", 42, 0.001)
	fe := NewFakeExecutor(map[string]Script{
		"node-a": {Output: out},
	})

	got, err := fe.Execute(context.Background(), domain.NodeInstance{ID: "node-a", Type: domain.NodeTypeAgent}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var text string
	if err := json.Unmarshal(got.Data, &text); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if text != "hello" {
		t.Fatalf("output = %q, want hello", text)
	}
	if got.TokensUsed == nil || *got.TokensUsed != 42 {
		t.Fatalf("tokens = %v, want 42", got.TokensUsed)
	}
}

func TestFakeExecutor_Execute_MissingScript(t *testing.T) {
	t.Parallel()

	fe := NewFakeExecutor(nil)
	_, err := fe.Execute(context.Background(), domain.NodeInstance{ID: "no-such"}, nil)
	if err == nil {
		t.Fatal("expected error for missing script")
	}
}

func TestFakeExecutor_Execute_Error(t *testing.T) {
	t.Parallel()

	want := errors.New("provider blew up")
	fe := NewFakeExecutor(map[string]Script{"x": {Error: want}})

	_, err := fe.Execute(context.Background(), domain.NodeInstance{ID: "x"}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestFakeExecutor_ExecuteStream_ChunksAndCompletion(t *testing.T) {
	t.Parallel()

	fe := NewFakeExecutor(map[string]Script{
		"n": {
			Chunks: []string{"Hel", "lo, ", "world"},
			Output: TextOutput("Hello, world"),
		},
	})

	chunks, results, errs := fe.ExecuteStream(context.Background(), domain.NodeInstance{ID: "n"}, nil)

	var got []string
	for c := range chunks {
		got = append(got, c.Content)
	}
	if len(got) != 3 || got[0] != "Hel" || got[2] != "world" {
		t.Fatalf("chunks = %v, want [Hel lo, world]", got)
	}

	// The fake deliberately leaves resultCh + errCh open (only chunkCh is
	// closed) so the engine's `select { <-resultCh; <-errCh }` never races
	// on two closed channels. Here: receive from resultCh with a select
	// so we can detect "error fired on the success path" via errCh being
	// ready with a value.
	select {
	case r := <-results:
		var s string
		if err := json.Unmarshal(r.Data, &s); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if s != "Hello, world" {
			t.Fatalf("output = %q", s)
		}
	case e := <-errs:
		t.Fatalf("unexpected err on success path: %v", e)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for result")
	}
}

func TestFakeExecutor_ExecuteStream_ErrorPath(t *testing.T) {
	t.Parallel()

	want := errors.New("stream interrupted")
	fe := NewFakeExecutor(map[string]Script{
		"n": {Chunks: []string{"partial"}, Error: want},
	})

	chunks, results, errs := fe.ExecuteStream(context.Background(), domain.NodeInstance{ID: "n"}, nil)

	// Drain chunks first; then the error channel fires.
	got := 0
	for range chunks {
		got++
	}
	if got != 1 {
		t.Fatalf("chunks received = %d, want 1", got)
	}

	select {
	case r := <-results:
		t.Fatalf("unexpected result on error path: %+v", r)
	case e := <-errs:
		if !errors.Is(e, want) {
			t.Fatalf("err = %v, want %v", e, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for err")
	}
}

func TestFakeExecutor_RecordsCalls(t *testing.T) {
	t.Parallel()

	fe := NewFakeExecutor(map[string]Script{
		"a": {Output: TextOutput("x")},
		"b": {Output: TextOutput("y")},
	})

	inputsA := map[string]json.RawMessage{"in": json.RawMessage(`"hi"`)}
	_, _ = fe.Execute(context.Background(), domain.NodeInstance{ID: "a", Type: domain.NodeTypeAgent}, inputsA)
	_, _, _ = fe.ExecuteStream(context.Background(), domain.NodeInstance{ID: "b", Type: domain.NodeTypeMedia}, nil)

	calls := fe.Calls()
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}
	if calls[0].NodeID != "a" || calls[0].Stream {
		t.Errorf("call 0 = %+v", calls[0])
	}
	if calls[1].NodeID != "b" || !calls[1].Stream {
		t.Errorf("call 1 = %+v", calls[1])
	}
	// Input copy must be independent of the original map.
	delete(inputsA, "in")
	if len(calls[0].Inputs) != 1 {
		t.Errorf("recorded inputs should be a copy, got %v after mutation", calls[0].Inputs)
	}
}

func TestFileRefOutput_SerialisesClothoURL(t *testing.T) {
	t.Parallel()

	out := FileRefOutput("proj/pipe/exec/image-1.png")
	var s string
	if err := json.Unmarshal(out.Data, &s); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := "clotho://file/proj/pipe/exec/image-1.png"
	if s != want {
		t.Fatalf("got %q, want %q", s, want)
	}
}
