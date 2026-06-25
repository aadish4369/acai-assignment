package httpx

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.method", r.Method),
					attribute.String("http.route", r.URL.Path),
				),
			)
			defer span.End()

			saw := &statusAwareResponseWriter{ResponseWriter: w}
			handler.ServeHTTP(saw, r.WithContext(ctx))

			status := saw.statusOrDefault()
			span.SetAttributes(attribute.Int("http.status_code", status))
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}
		})
	}
}
