package api

import (
	"fmt"
	"net/http"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	ddtrace_ext "gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/opentracer"
)

type tracingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newTracingResponseWriter(w http.ResponseWriter) *tracingResponseWriter {
	return &tracingResponseWriter{w, http.StatusOK}
}

func (trw *tracingResponseWriter) WriteHeader(code int) {
	trw.statusCode = code
	trw.ResponseWriter.WriteHeader(code)
}

func tracer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientContext, _ := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(r.Header))
		span, traceCtx := opentracing.StartSpanFromContext(r.Context(), "http.handler",
			ext.RPCServerOption(clientContext),
			opentracer.SpanType(ddtrace_ext.AppTypeWeb),
		)
		defer span.Finish()

		ext.HTTPMethod.Set(span, r.Method)
		ext.HTTPUrl.Set(span, r.URL.Path)
		resourceName := r.URL.Path
		resourceName = r.Method + " " + resourceName
		span.SetTag("resource.name", resourceName)

		if reqID := getRequestID(r.Context()); reqID != "" {
			span.SetTag("http.request_id", reqID)
		}

		trw := newTracingResponseWriter(w)
		next.ServeHTTP(trw, r.WithContext(traceCtx))

		status := trw.statusCode
		ext.HTTPStatusCode.Set(span, uint16(status))
		if status >= 500 && status < 600 {
			ext.Error.Set(span, true)
			span.SetTag("error.type", fmt.Sprintf("%d: %s", status, http.StatusText(status)))
			span.LogKV(
				"event", "error",
				"message", fmt.Sprintf("%d: %s", status, http.StatusText(status)),
			)
		}
	})
}
