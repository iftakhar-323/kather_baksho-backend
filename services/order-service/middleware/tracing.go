package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var Tracer trace.Tracer

// InitTracer configures OpenTelemetry exporter and returns a shutdown function.
func InitTracer(serviceName string) func(context.Context) error {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://jaeger:4318"
	}

	ctx := context.Background()

	// Configure OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint+"/v1/traces"),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(3*time.Second),
	)
	if err != nil {
		log.Printf("[OpenTelemetry] Warning: Failed to create OTLP trace exporter: %v. Continuing in standalone mode.", err)
		return func(context.Context) error { return nil }
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			attribute.String("environment", os.Getenv("GIN_MODE")),
			attribute.String("service.version", "2.0.0"),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	Tracer = tp.Tracer(serviceName)
	log.Printf("[OpenTelemetry] Tracing initialized for '%s' -> %s", serviceName, endpoint)

	return tp.Shutdown
}

// OpenTelemetryMiddleware wraps Gin requests in OpenTelemetry spans and propagates W3C headers.
func OpenTelemetryMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.GetTracerProvider().Tracer(serviceName)

	return func(c *gin.Context) {
		// Extract incoming W3C TraceContext
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if c.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.client_ip", c.ClientIP()),
				attribute.String("http.user_agent", c.Request.UserAgent()),
			),
		)
		defer span.End()

		// Inject traceparent into response headers so clients & frontend can correlate traces
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(c.Writer.Header()))

		// Pass traced context down Gin handlers
		c.Request = c.Request.WithContext(ctx)

		traceID := span.SpanContext().TraceID().String()
		spanID := span.SpanContext().SpanID().String()

		c.Set("trace_id", traceID)
		c.Set("span_id", spanID)
		c.Header("X-Trace-ID", traceID)

		c.Next()

		status := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.status_code", status))

		if status >= http.StatusInternalServerError {
			span.SetAttributes(attribute.Bool("error", true))
		}
	}
}
