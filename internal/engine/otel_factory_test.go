package engine

import (
	"context"
	"testing"
)

func TestNewTracer_DefaultsToNoOp(t *testing.T) {
	t.Setenv("OTEL_EXPORTER", "")
	tr, shutdown, err := NewTracer(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	if tr == nil {
		t.Fatal("tracer should never be nil")
	}
	// A no-op tracer creates non-recording spans. Easiest assertion:
	// the SpanContext is invalid for a span started from no-op.
	_, span := tr.Start(context.Background(), "probe")
	defer span.End()
	if span.SpanContext().IsValid() {
		t.Errorf("no-op tracer should produce invalid SpanContext")
	}
}

func TestNewTracer_NoneIsExplicitlyNoOp(t *testing.T) {
	t.Setenv("OTEL_EXPORTER", "none")
	tr, shutdown, err := NewTracer(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	_, span := tr.Start(context.Background(), "probe")
	defer span.End()
	if span.SpanContext().IsValid() {
		t.Errorf("explicit none should produce invalid SpanContext")
	}
}

func TestNewTracer_StdoutProducesRecordingSpans(t *testing.T) {
	t.Setenv("OTEL_EXPORTER", "stdout")
	tr, shutdown, err := NewTracer(context.Background())
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	_, span := tr.Start(context.Background(), "probe")
	defer span.End()
	if !span.SpanContext().IsValid() {
		t.Errorf("stdout exporter should produce a valid SpanContext")
	}
	if !span.IsRecording() {
		t.Errorf("stdout exporter should produce recording spans")
	}
}

func TestNewTracer_UnknownExporterReturnsError(t *testing.T) {
	t.Setenv("OTEL_EXPORTER", "jaeger") // not implemented yet
	_, _, err := NewTracer(context.Background())
	if err == nil {
		t.Fatal("expected error for unknown exporter")
	}
}
