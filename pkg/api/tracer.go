package api

import (
	"context"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"github.com/stefanprodan/podinfo/pkg/version"

	"go.opentelemetry.io/contrib/detectors/aws/ecs"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gorilla/mux/otelmux"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const (
	instrumentationName = "github.com/stefanprodan/podinfo/pkg/api"
)

func (s *Server) initTracer(ctx context.Context) {
	if viper.GetString("otel-service-name") == "" {
		nop := trace.NewNoopTracerProvider()
		s.tracer = nop.Tracer(viper.GetString("otel-service-name"))
		return
	}

	client := otlptracegrpc.NewClient()
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		s.logger.Error("creating OTLP trace exporter", zap.Error(err))
	}

	ecsResourceDetector := ecs.NewResourceDetector()
	ecsResource, err := ecsResourceDetector.Detect(ctx)
	if err != nil {
		s.logger.Error("Detecting ECS Resource", zap.Error(err))
	}

	s.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(ecsResource),
		sdktrace.WithIDGenerator(xray.NewIDGenerator()),
	)

	otel.SetTracerProvider(s.tracerProvider)
	otel.SetTextMapPropagator(xray.Propagator{})

	s.tracer = s.tracerProvider.Tracer(
		instrumentationName,
		trace.WithInstrumentationVersion(version.VERSION),
		trace.WithSchemaURL(semconv.SchemaURL),
	)

}

func NewOpenTelemetryMiddleware() mux.MiddlewareFunc {
	return otelmux.Middleware(viper.GetString("otel-service-name"))
}
