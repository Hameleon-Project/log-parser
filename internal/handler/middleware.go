package handler

import (
	"log/slog"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.code = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	if rec.code == 0 {
		rec.code = http.StatusOK
	}
	return rec.ResponseWriter.Write(b)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, code: 0}
		next.ServeHTTP(rec, r)
		status := rec.code
		if status == 0 {
			status = http.StatusOK
		}
		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
