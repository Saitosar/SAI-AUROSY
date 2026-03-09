package observability

import (
	"context"
	"net/http"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/sai-aurosy/platform/control-plane"

var defaultTracer = otel.Tracer(tracerName)

// InitTracer initializes the OpenTelemetry tracer. If otlpEndpoint is empty, tracing is disabled (no-op).
// otlpEndpoint can be "host:port" or "http://host:port" (scheme is stripped for OTLP HTTP).
func InitTracer(serviceName, otlpEndpoint string) (func(), error) {
	if otlpEndpoint == "" {
		otlpEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	if otlpEndpoint == "" {
		return func() {}, nil
	}
	// Strip scheme for WithEndpoint (expects host:port)
	if strings.HasPrefix(otlpEndpoint, "http://") {
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "http://")
	} else if strings.HasPrefix(otlpEndpoint, "https://") {
		otlpEndpoint = strings.TrimPrefix(otlpEndpoint, "https://")
	}
	if serviceName == "" {
		serviceName = os.Getenv("OTEL_SERVICE_NAME")
	}
	if serviceName == "" {
		serviceName = "sai-aurosy-control-plane"
	}

	ctx := context.Background()
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func() {
		_ = tp.Shutdown(ctx)
	}, nil
}

// TracingMiddleware wraps the handler with OpenTelemetry HTTP instrumentation.
// Extracts W3C Trace Context, creates a span per request, sets http.method, http.route, http.status_code.
// Uses default tracer provider; no-op if InitTracer was not called with an endpoint.
func TracingMiddleware(next http.Handler) http.Handler {
	return otelhttp.NewHandler(next, "control-plane",
		otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		otelhttp.WithPropagators(otel.GetTextMapPropagator()),
	)
}

// StartSpan starts a child span with the given name and attributes.
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, func()) {
	ctx, span := defaultTracer.Start(ctx, name, trace.WithAttributes(attrs...))
	return ctx, func() { span.End() }
}

// Tracer returns the package tracer for manual instrumentation.
func Tracer() trace.Tracer {
	return defaultTracer
}
