// Package otel provides OpenTelemetry tracing integration using OTLP exporters.
//
// Note: This package was originally designed for Jaeger integration, but since
// OpenTelemetry dropped support for the Jaeger exporter in July 2023, it now
// uses OTLP (OpenTelemetry Protocol) exporters instead. Jaeger officially
// accepts and recommends using OTLP for sending traces.
//
// Supported exporter types:
//   - otlp-http: OTLP over HTTP (recommended for most use cases)
//   - otlp-grpc: OTLP over gRPC (for high-performance scenarios)
//
// The package maintains backward compatibility by keeping the same configuration
// structure and API, but internally uses OTLP exporters to send traces to
// Jaeger or other OTLP-compatible backends like Uptrace.
package otel

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	// "go.opentelemetry.io/otel/exporters/jaeger" // deprecated: use OTLP exporters instead
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

var (
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
	mu             sync.Mutex
	initialized    bool

	ErrOTELIsDisabled = errors.New("otel is disabled")
)

// Init initializes the OpenTelemetry tracer with OTLP exporters.
// This function replaces the deprecated Jaeger exporter with OTLP exporters
// that are compatible with Jaeger and other tracing backends.
func Init() error {
	cfg := config.App.OTEL
	if !cfg.Enable {
		logger.OTEL.Info("otel tracing is disabled")
		return nil
	}

	mu.Lock()
	defer mu.Unlock()
	if initialized {
		return nil
	}

	// Create exporter
	exporter, err := createExporter(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create exporter")
	}

	// Create resource
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create resource")
	}

	// Create sampler
	sampler := createSampler(cfg)

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			exporter,
			sdktrace.WithBatchTimeout(cfg.BufferFlushInterval),
			sdktrace.WithMaxExportBatchSize(cfg.ReporterQueueSize),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	// Set global propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create tracer
	tracer = otel.Tracer(cfg.ServiceName)

	// Store tracer provider for cleanup
	tracerProvider = tp

	initialized = true
	logger.OTEL.Info(
		"otel tracing initialized",
		zap.String("service_name", cfg.ServiceName),
		zap.String("exporter_type", string(cfg.ExporterType)),
		zap.String("sampler_type", string(cfg.SamplerType)),
	)

	return nil
}

// Close closes the Jaeger tracer
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if !initialized || tracerProvider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := tracerProvider.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "failed to shutdown tracer provider")
	}

	initialized = false
	logger.OTEL.Info("otel tracer closed")
	return nil
}

// GetTracer returns the global tracer
func GetTracer() trace.Tracer {
	if !initialized {
		return noop.NewTracerProvider().Tracer("noop")
	}
	return tracer
}

// IsEnabled returns whether Jaeger tracing is enabled
func IsEnabled() bool {
	return config.App.OTEL.Enable && initialized
}

// StartSpan starts a new span with the given name and options
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if !IsEnabled() {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name, opts...)
}

// SpanFromContext returns the span from the context
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// createExporter creates an exporter based on configuration
func createExporter(cfg config.OTEL) (sdktrace.SpanExporter, error) {
	switch cfg.ExporterType {
	case config.ExportTypeOtlpHTTP:
		// Create OTLP HTTP exporter
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
		}

		if cfg.OTLPInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}

		// Prepare headers with Uptrace DSN support
		headers := make(map[string]string)

		// Copy existing headers
		maps.Copy(headers, cfg.OTLPHeaders)

		// Add Uptrace DSN header if present in headers
		// This allows users to set uptrace-dsn in the OTLPHeaders configuration
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}

		// Enable compression for better performance (recommended by Uptrace)
		opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))

		return otlptracehttp.New(context.Background(), opts...)

	case config.ExportTypeOtlpGRPC:
		// Create OTLP gRPC exporter
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		}

		if cfg.OTLPInsecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}

		// Prepare headers with Uptrace DSN support
		headers := make(map[string]string)

		// Copy existing headers
		maps.Copy(headers, cfg.OTLPHeaders)

		// Add Uptrace DSN header if present in headers
		// This allows users to set uptrace-dsn in the OTLPHeaders configuration
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}

		// Enable compression for better performance (recommended by Uptrace)
		opts = append(opts, otlptracegrpc.WithCompressor("gzip"))

		return otlptracegrpc.New(context.Background(), opts...)

	// NOTE: Jaeger exporter is deprecated since OpenTelemetry dropped support in July 2023
	// Jaeger officially accepts and recommends using OTLP instead
	// Use otlp-http or otlp-grpc exporter types instead
	//
	// case "jaeger":
	// 	fallthrough
	// default:
	// 	// Use Jaeger exporter (default behavior)
	// 	if cfg.CollectorURL != "" {
	// 		// Use HTTP collector
	// 		return otel.New(otel.WithCollectorEndpoint(
	// 			otel.WithEndpoint(cfg.CollectorURL),
	// 		))
	// 	}
	//
	// 	// Use UDP agent
	// 	return otel.New(otel.WithAgentEndpoint(
	// 		otel.WithAgentHost(cfg.AgentEndpoint),
	// 	))

	default:
		return nil, errors.Errorf("unsupported exporter type: %s. Use 'otlp-http' or 'otlp-grpc' instead", cfg.ExporterType)
	}
}

// createSampler creates a sampler based on configuration
func createSampler(cfg config.OTEL) sdktrace.Sampler {
	switch cfg.SamplerType {
	case config.SamplerTypeConst:
		if cfg.SamplerParam >= 1.0 {
			return sdktrace.AlwaysSample()
		}
		return sdktrace.NeverSample()
	case config.SamplerTypeProbabilistic:
		return sdktrace.TraceIDRatioBased(cfg.SamplerParam)
	case config.SamplerTypeRateLimiting:
		// Note: OpenTelemetry doesn't have built-in rate limiting sampler
		// This would need to be implemented separately
		return sdktrace.TraceIDRatioBased(cfg.SamplerParam)
	default:
		return sdktrace.AlwaysSample()
	}
}

// AddSpanTags adds tags to the current span
func AddSpanTags(span trace.Span, tags map[string]any) {
	if span == nil || !span.IsRecording() {
		return
	}

	for key, value := range tags {
		switch v := value.(type) {
		case string:
			span.SetAttributes(attribute.String(key, v))
		case int:
			span.SetAttributes(attribute.Int(key, v))
		case int64:
			span.SetAttributes(attribute.Int64(key, v))
		case float64:
			span.SetAttributes(attribute.Float64(key, v))
		case bool:
			span.SetAttributes(attribute.Bool(key, v))
		default:
			span.SetAttributes(attribute.String(key, "unsupported_type"))
		}
	}
}

// AddSpanEvent adds an event to the current span
func AddSpanEvent(span trace.Span, name string, attrs ...attribute.KeyValue) {
	if span == nil || !span.IsRecording() {
		return
	}
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// RecordError records an error in the current span
func RecordError(span trace.Span, err error) {
	if span == nil || !span.IsRecording() || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
