package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	appmetrics "infinite-experiment/politburo/internal/metrics"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const maxErrorBodyCapture = 512

func AccessLog(metrics *appmetrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			base := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			capture := &errorBodyCapture{WrapResponseWriter: base}
			next.ServeHTTP(capture, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			status := capture.Status()
			metrics.Requests.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
			metrics.RequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(started).Seconds())

			args := []any{
				"request_id", chimiddleware.GetReqID(r.Context()),
				"method", r.Method,
				"route", route,
				"status", status,
				"duration_ms", time.Since(started).Milliseconds(),
			}
            if route == "unmatched" || status >= 400 {
				args = append(args, "request_path", r.URL.Path)
			}
			if code := capture.errorCode(); code != "" {
				args = append(args, "error_code", code)
			}
			slog.Info("http request", args...)
		})
	}
}

type errorBodyCapture struct {
	chimiddleware.WrapResponseWriter
	body []byte
}

func (c *errorBodyCapture) Write(b []byte) (int, error) {
	if c.Status() == 0 {
		c.WriteHeader(http.StatusOK)
	}
	if c.Status() >= 400 && len(c.body) < maxErrorBodyCapture {
		remain := maxErrorBodyCapture - len(c.body)
		if len(b) > remain {
			c.body = append(c.body, b[:remain]...)
		} else {
			c.body = append(c.body, b...)
		}
	}
	return c.WrapResponseWriter.Write(b)
}

func (c *errorBodyCapture) errorCode() string {
	if c.Status() < 400 || len(c.body) == 0 {
		return ""
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(c.body, &envelope); err != nil {
		return ""
	}
	return envelope.Error.Code
}
