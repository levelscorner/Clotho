package engine

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// NewTracer returns the tracer the engine should use, plus a shutdown
// function the caller MUST defer until program exit. The exporter is
// chosen via the OTEL_EXPORTER env var:
//
//	"stdout"  → human-readable spans on stdout (dev / debugging)
//	"none"    → no-op tracer (default; spans aren't recorded anywhere)
//
// We deliberately avoid auto-OTLP detection because that drags a config
// surface (endpoint, headers, sampling) the local-first deployment
// doesn't need yet. Add OTLP as a third value when a collector exists.
func NewTracer(ctx context.Context) (trace.Tracer, func(context.Context) error, error) {
	exporterKind := os.Getenv("OTEL_EXPORTER")
	if exporterKind == "" {
		exporterKind = "none"
	}

	switch exporterKind {
	case "none":
		return noop.NewTracerProvider().Tracer("clotho/noop"),
			func(context.Context) error { return nil }, nil

	case "stdout":
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, nil, fmt.Errorf("init stdouttrace exporter: %w", err)
		}
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
		)
		return tp.Tracer("clotho"), tp.Shutdown, nil
	}

	return nil, nil, fmt.Errorf("unsupported OTEL_EXPORTER %q (want: none | stdout)", exporterKind)
}
