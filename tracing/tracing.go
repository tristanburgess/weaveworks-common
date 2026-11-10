package tracing

import (
	"context"
	"io"

	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	jaeger "github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerprom "github.com/uber/jaeger-lib/metrics/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// ErrInvalidConfiguration is an error to notify client to provide valid trace report agent or config server
var (
	ErrBlankTraceConfiguration = errors.New("no trace report agent, config server, or collector endpoint specified")
)

// installJaeger registers Jaeger as the OpenTracing implementation.
func installJaeger(serviceName string, cfg *jaegercfg.Configuration, options ...jaegercfg.Option) (io.Closer, error) {
	metricsFactory := jaegerprom.New()

	// put the metricsFactory earlier so provided options can override it
	opts := append([]jaegercfg.Option{jaegercfg.Metrics(metricsFactory)}, options...)

	closer, err := cfg.InitGlobalTracer(serviceName, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize jaeger tracer")
	}
	return closer, nil
}

// NewFromEnv is a convenience function to allow tracing configuration
// via environment variables
//
// Tracing will be enabled if one (or more) of the following environment variables is used to configure trace reporting:
// - JAEGER_AGENT_HOST
// - JAEGER_SAMPLER_MANAGER_HOST_PORT
func NewFromEnv(serviceName string, options ...jaegercfg.Option) (io.Closer, error) {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		return nil, errors.Wrap(err, "could not load jaeger tracer configuration")
	}

	if cfg.Sampler.SamplingServerURL == "" && cfg.Reporter.LocalAgentHostPort == "" && cfg.Reporter.CollectorEndpoint == "" {
		return nil, ErrBlankTraceConfiguration
	}

	return installJaeger(serviceName, cfg, options...)
}

// ExtractTraceID extracts the trace id, if any from the context.
func ExtractTraceID(ctx context.Context) (string, bool) {
	traceID, _, ok := extractTraceID(ctx)
	return traceID, ok
}

// ExtractSampledTraceID works like ExtractTraceID but the returned bool is only
// true if the returned trace id is sampled.
func ExtractSampledTraceID(ctx context.Context) (string, bool) {
	traceID, sampled, _ := extractTraceID(ctx)
	return traceID, sampled
}

// extractTraceID reads a Jaeger span context first, then falls back to the OTel one.
// The fallback also covers spans started through the OTel OpenTracing bridge, whose
// span contexts aren't Jaeger ones but which also store the OTel span in the context.
func extractTraceID(ctx context.Context) (traceID string, sampled, ok bool) {
	if sp := opentracing.SpanFromContext(ctx); sp != nil {
		if sctx, ok := sp.Context().(jaeger.SpanContext); ok {
			return sctx.TraceID().String(), sctx.IsSampled(), true
		}
	}

	if sctx := trace.SpanContextFromContext(ctx); sctx.TraceID().IsValid() {
		return sctx.TraceID().String(), sctx.IsSampled(), true
	}

	return "", false, false
}
