package otel

import (
	"context"
	"testing"
	"time"

	"github.com/forbearing/gst/config"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestNormalizeConfigUsesOTELDefaults(t *testing.T) {
	cfg, err := normalizeConfig(config.OTEL{})
	require.NoError(t, err)

	require.Equal(t, config.OTLPProtocolHTTPProtobuf, cfg.ExporterOTLPProtocol)
	require.Equal(t, "http://localhost:4318/v1/traces", cfg.ExporterOTLPTracesEndpoint)
	require.Equal(t, config.OTLPCompressionNone, cfg.ExporterOTLPCompression)
	require.Equal(t, config.TracesSamplerParentBasedAlwaysOn, cfg.TracesSampler)
	require.Equal(t, "", cfg.TracesSamplerArg)
	require.Equal(t, sdktrace.DefaultMaxQueueSize, cfg.BSPMaxQueueSize)
	require.Equal(t, sdktrace.DefaultMaxExportBatchSize, cfg.BSPMaxExportBatchSize)
	require.Equal(t, time.Duration(sdktrace.DefaultScheduleDelay)*time.Millisecond, cfg.BSPScheduleDelay)
	require.Equal(t, time.Duration(sdktrace.DefaultExportTimeout)*time.Millisecond, cfg.BSPExportTimeout)
}

func TestNormalizeConfigUsesGRPCDefaultEndpoint(t *testing.T) {
	cfg, err := normalizeConfig(config.OTEL{
		ExporterOTLPProtocol: config.OTLPProtocolGRPC,
	})
	require.NoError(t, err)

	require.Equal(t, "http://localhost:4317", cfg.ExporterOTLPTracesEndpoint)
}

func TestNewSamplerUsesParentBasedTraceIDRatio(t *testing.T) {
	sampler, err := newSampler(config.OTEL{
		TracesSampler:    config.TracesSamplerParentBasedTraceIDRatio,
		TracesSamplerArg: "1",
	})
	require.NoError(t, err)

	require.Equal(t, sdktrace.RecordAndSample, sampleDecision(context.Background(), t, sampler))
	require.Equal(t, sdktrace.RecordAndSample, sampleDecision(parentContext(t, true), t, sampler))
	require.Equal(t, sdktrace.Drop, sampleDecision(parentContext(t, false), t, sampler))
}

func TestNewSamplerRejectsInvalidTraceIDRatio(t *testing.T) {
	for _, arg := range []string{"-0.1", "1.1", "not-a-number"} {
		t.Run(arg, func(t *testing.T) {
			_, err := newSampler(config.OTEL{
				TracesSampler:    config.TracesSamplerParentBasedTraceIDRatio,
				TracesSamplerArg: arg,
			})
			require.Error(t, err)
		})
	}
}

func TestNormalizeConfigRejectsOversizedExportBatch(t *testing.T) {
	_, err := normalizeConfig(config.OTEL{
		BSPMaxQueueSize:       100,
		BSPMaxExportBatchSize: 101,
	})
	require.Error(t, err)
}

func sampleDecision(parent context.Context, t *testing.T, sampler sdktrace.Sampler) sdktrace.SamplingDecision {
	t.Helper()

	traceID, err := oteltrace.TraceIDFromHex("11111111111111111111111111111111")
	require.NoError(t, err)

	result := sampler.ShouldSample(sdktrace.SamplingParameters{
		ParentContext: parent,
		TraceID:       traceID,
		Name:          "GET /api/ping",
		Kind:          oteltrace.SpanKindServer,
	})
	return result.Decision
}

func parentContext(t *testing.T, sampled bool) context.Context {
	t.Helper()

	traceID, err := oteltrace.TraceIDFromHex("22222222222222222222222222222222")
	require.NoError(t, err)

	spanID, err := oteltrace.SpanIDFromHex("3333333333333333")
	require.NoError(t, err)

	var flags oteltrace.TraceFlags
	if sampled {
		flags = oteltrace.FlagsSampled
	}

	spanContext := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: flags,
		Remote:     true,
	})
	return oteltrace.ContextWithRemoteSpanContext(context.Background(), spanContext)
}
