package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/logger"
	pkgzap "github.com/forbearing/gst/logger/zap"
	gstotel "github.com/forbearing/gst/provider/otel"
	"github.com/forbearing/gst/types/consts"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestTracingUsesIncomingTraceparent(t *testing.T) {
	setupTracingTest(t)

	const incomingTraceID = "11111111111111111111111111111111"

	router := gin.New()
	router.Use(Tracing())
	router.GET("/api/ping", func(c *gin.Context) {
		spanContext := oteltrace.SpanFromContext(c.Request.Context()).SpanContext()
		require.True(t, spanContext.HasTraceID())
		require.Equal(t, incomingTraceID, spanContext.TraceID().String())
		require.Equal(t, incomingTraceID, c.GetString(consts.TRACE_ID))
		require.Equal(t, incomingTraceID, c.GetString(consts.REQUEST_ID))
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("traceparent", "00-"+incomingTraceID+"-2222222222222222-01")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, incomingTraceID, w.Header().Get(consts.HEADER_TRACE_ID))
}

func TestTracingUsesIncomingTraceIDHeader(t *testing.T) {
	setupTracingTest(t)

	const incomingTraceID = "33333333333333333333333333333333"

	router := gin.New()
	router.Use(Tracing())
	router.GET("/api/ping", func(c *gin.Context) {
		spanContext := oteltrace.SpanFromContext(c.Request.Context()).SpanContext()
		require.True(t, spanContext.HasTraceID())
		require.Equal(t, incomingTraceID, spanContext.TraceID().String())
		require.Equal(t, incomingTraceID, c.GetString(consts.TRACE_ID))
		require.Equal(t, incomingTraceID, c.GetString(consts.REQUEST_ID))
		c.Status(http.StatusNoContent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set(consts.HEADER_TRACE_ID, incomingTraceID)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNoContent, w.Code)
	require.Equal(t, incomingTraceID, w.Header().Get(consts.HEADER_TRACE_ID))
}

func TestTracingMarksHTTPSpanAsRequestRoot(t *testing.T) {
	source := readMiddlewareSource(t, "tracing.go")
	require.Contains(t, source, "ctx = gstotel.ContextWithRequestRootSpan(ctx)")
}

func TestMiddlewareWrapperStartsMiddlewareSpansFromRequestRoot(t *testing.T) {
	source := readMiddlewareSource(t, "wrapper.go")
	require.Contains(t, source, "parentCtx := gstotel.RequestRootContext(originalCtx)")
}

func setupTracingTest(t *testing.T) {
	t.Helper()

	setupTracingTestWithEndpoint(t, "127.0.0.1:1", 0)
}

func setupTracingTestWithEndpoint(t *testing.T, endpoint string, samplerParam float64) {
	t.Helper()

	gin.SetMode(gin.TestMode)

	originalConfig := config.App
	config.App = new(config.Config)
	config.App.OTEL.Enable = true
	config.App.OTEL.ServiceName = "gst-test"
	config.App.OTEL.ExporterType = config.ExportTypeOtlpHTTP
	config.App.OTEL.OTLPEndpoint = endpoint
	config.App.OTEL.OTLPInsecure = true
	config.App.OTEL.SamplerType = config.SamplerTypeConst
	config.App.OTEL.SamplerParam = samplerParam
	config.App.OTEL.BufferFlushInterval = 10 * time.Millisecond
	config.App.OTEL.ReporterQueueSize = 100
	t.Cleanup(func() {
		config.App = originalConfig
	})

	originalOTELLogger := logger.OTEL
	logger.OTEL = pkgzap.New("/dev/null")
	t.Cleanup(func() {
		logger.OTEL = originalOTELLogger
	})

	gstotel.Close()
	require.NoError(t, gstotel.Init())
	t.Cleanup(func() {
		gstotel.Close()
	})
}

func readMiddlewareSource(t *testing.T, filename string) string {
	t.Helper()

	source, err := os.ReadFile(filename)
	require.NoError(t, err)
	return string(source)
}
