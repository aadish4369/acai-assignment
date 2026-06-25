package httpx

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)


func Metrics(meter metric.Meter) (func(http.Handler) http.Handler, error) {
	requests, err := meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total number of HTTP requests handled."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, err
	}

	duration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP requests in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			saw := &statusAwareResponseWriter{ResponseWriter: w}

			handler.ServeHTTP(saw, r)

			status := saw.statusOrDefault()
			attrs := metric.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.route", r.URL.Path),
				attribute.Int("http.status_code", status),
				attribute.Bool("error", status >= http.StatusInternalServerError),
			)

			requests.Add(r.Context(), 1, attrs)
			duration.Record(r.Context(), float64(time.Since(start).Microseconds())/1000.0, attrs)
		})
	}, nil
}
