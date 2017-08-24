package api

import (
	"fmt"
	"net/http"
	"time"

	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
)

func newStructuredLogger(logger *logrus.Logger) func(next http.Handler) http.Handler {
	return chimiddleware.RequestLogger(&structuredLogger{logger})
}

type structuredLogger struct {
	Logger *logrus.Logger
}

func (l *structuredLogger) NewLogEntry(r *http.Request) chimiddleware.LogEntry {
	entry := &structuredLoggerEntry{Logger: logrus.NewEntry(l.Logger)}
	logFields := logrus.Fields{
		"component":   "api",
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
		"referer":     r.Referer(),
	}

	if reqID := getRequestID(r.Context()); reqID != "" {
		logFields["request_id"] = reqID
	}

	entry.Logger = entry.Logger.WithFields(logFields)
	entry.Logger.Infoln("request started")
	return entry
}

type structuredLoggerEntry struct {
	Logger logrus.FieldLogger
}

func (l *structuredLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
	l.Logger = l.Logger.WithFields(logrus.Fields{
		"status":   status,
		"duration": elapsed.Nanoseconds(),
	})

	l.Logger.Info("request completed")
}

func (l *structuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.Logger.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	}).Panic("unhandled request panic")
}

func getLogEntry(r *http.Request) logrus.FieldLogger {
	entry, _ := chimiddleware.GetLogEntry(r).(*structuredLoggerEntry)
	if entry == nil {
		return logrus.NewEntry(logrus.StandardLogger())
	}
	return entry.Logger
}

func logEntrySetField(r *http.Request, key string, value interface{}) logrus.FieldLogger {
	if entry, ok := r.Context().Value(chimiddleware.LogEntryCtxKey).(*structuredLoggerEntry); ok {
		entry.Logger = entry.Logger.WithField(key, value)
		return entry.Logger
	}
	return nil
}

func logEntrySetFields(r *http.Request, fields logrus.Fields) logrus.FieldLogger {
	if entry, ok := r.Context().Value(chimiddleware.LogEntryCtxKey).(*structuredLoggerEntry); ok {
		entry.Logger = entry.Logger.WithFields(fields)
		return entry.Logger
	}
	return nil
}
