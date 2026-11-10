package tracing

import (
	"context"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/require"
	jaeger "github.com/uber/jaeger-client-go"
	"go.opentelemetry.io/otel/trace"
)

func TestExtractTraceID(t *testing.T) {
	tracer, closer := jaeger.NewTracer("test", jaeger.NewConstSampler(true), jaeger.NewNullReporter())
	defer func() { _ = closer.Close() }()
	jaegerSpan := tracer.StartSpan("op")
	defer jaegerSpan.Finish()
	jaegerID := jaegerSpan.Context().(jaeger.SpanContext).TraceID().String()

	otelCtx := func(flags trace.TraceFlags) context.Context {
		return trace.ContextWithSpanContext(context.Background(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID{15: 1},
			SpanID:     trace.SpanID{7: 1},
			TraceFlags: flags,
		}))
	}
	const otelID = "00000000000000000000000000000001"

	for _, tc := range []struct {
		name        string
		ctx         context.Context
		wantID      string
		wantOK      bool
		wantSampled bool
	}{
		{name: "no span", ctx: context.Background()},
		{name: "jaeger span", ctx: opentracing.ContextWithSpan(context.Background(), jaegerSpan), wantID: jaegerID, wantOK: true, wantSampled: true},
		{name: "sampled otel span", ctx: otelCtx(trace.FlagsSampled), wantID: otelID, wantOK: true, wantSampled: true},
		{name: "unsampled otel span", ctx: otelCtx(0), wantID: otelID, wantOK: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := ExtractTraceID(tc.ctx)
			require.Equal(t, tc.wantID, id)
			require.Equal(t, tc.wantOK, ok)

			id, sampled := ExtractSampledTraceID(tc.ctx)
			require.Equal(t, tc.wantID, id)
			require.Equal(t, tc.wantSampled, sampled)
		})
	}
}
