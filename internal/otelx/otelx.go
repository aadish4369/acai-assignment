package otelx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const defaultTelemetryFile = "telemetry.log"


func Setup(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	res := resource.NewSchemaless(attribute.String("service.name", serviceName))

	path := os.Getenv("OTEL_LOG_FILE")
	if path == "" {
		path = defaultTelemetryFile
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	out := &syncWriter{w: file}

	metricExporter, err := stdoutmetric.New(
		stdoutmetric.WithWriter(out),
		stdoutmetric.WithPrettyPrint(),
		stdoutmetric.WithoutTimestamps(),
	)
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(60*time.Second))),
	)
	otel.SetMeterProvider(meterProvider)

	traceExporter, err := stdouttrace.New(
		stdouttrace.WithWriter(out),
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)
	otel.SetTracerProvider(tracerProvider)

	slog.InfoContext(ctx, "Telemetry configured", "file", path)

	shutdown := func(ctx context.Context) error {
		return errors.Join(
			meterProvider.Shutdown(ctx),
			tracerProvider.Shutdown(ctx),
			file.Close(),
		)
	}

	return shutdown, nil
}

type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
