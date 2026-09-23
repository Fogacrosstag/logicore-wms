package platform

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Endpoint func(http.ResponseWriter, *http.Request) (any, error)

func Wrap(fn Endpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, q *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(q.Context(), propagation.HeaderCarrier(q.Header))
		ctx, span := otel.Tracer("http").Start(ctx, q.Method+" "+q.URL.Path)
		defer span.End()
		c := q.Header.Get("X-Correlation-ID")
		if len(c) == 0 || len(c) > 80 {
			c = uuid.NewString()
		}
		ctx = context.WithValue(ctx, contextKey("correlation"), c)
		q = q.WithContext(ctx)
		w.Header().Set("X-Correlation-ID", c)
		w.Header().Set("Content-Type", "application/json")
		value, err := fn(w, q)
		if err != nil {
			status, code, message := 500, "INTERNAL_ERROR", "Unexpected server error"
			var fault *Fault
			if errors.As(err, &fault) {
				status, code, message = fault.Status, fault.Code, fault.Message
			} else {
				slog.Error("request failed", "error", err, "traceId", span.SpanContext().TraceID().String(), "correlationId", c)
			}
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]any{"code": code, "message": message, "details": map[string]any{}, "timestamp": time.Now().UTC(), "traceId": trace.SpanContextFromContext(ctx).TraceID().String()})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"data": value, "timestamp": time.Now().UTC()})
	}
}
func Body[T any](w http.ResponseWriter, q *http.Request) (T, error) {
	var v T
	q.Body = http.MaxBytesReader(w, q.Body, 1<<20)
	d := json.NewDecoder(q.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(&v); e != nil {
		return v, &Fault{"VALIDATION_ERROR", e.Error(), 400}
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return v, &Fault{"VALIDATION_ERROR", "Expected a single JSON value", 400}
	}
	return v, nil
}
func (r *Runtime) Mux() *http.ServeMux {
	m := http.NewServeMux()
	m.Handle("GET /metrics", promhttp.Handler())
	m.HandleFunc("GET /health", Wrap(func(_ http.ResponseWriter, q *http.Request) (any, error) {
		ctx, c := context.WithTimeout(q.Context(), 3*time.Second)
		defer c()
		if e := r.DB.Ping(ctx); e != nil {
			return nil, &Fault{"UNAVAILABLE", "PostgreSQL unavailable", 503}
		}
		if e := r.Redis.Ping(ctx).Err(); e != nil {
			return nil, &Fault{"UNAVAILABLE", "Redis unavailable", 503}
		}
		conn, e := kafka.DialContext(ctx, "tcp", r.Brokers[0])
		if e != nil {
			return nil, &Fault{"UNAVAILABLE", "Kafka unavailable", 503}
		}
		conn.Close()
		return map[string]string{"status": "UP"}, nil
	}))
	return m
}
func Get[T any](ctx context.Context, url string) (T, error) {
	var value T
	req, e := http.NewRequestWithContext(ctx, "GET", url, nil)
	if e != nil {
		return value, e
	}
	req.Header.Set("X-Correlation-ID", Correlation(ctx))
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	response, e := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if e != nil {
		return value, e
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		return value, &Fault{"NOT_FOUND", "Referenced resource not found", 404}
	}
	if response.StatusCode != 200 {
		return value, fmtStatus(response.StatusCode)
	}
	var envelope struct {
		Data T `json:"data"`
	}
	e = json.NewDecoder(response.Body).Decode(&envelope)
	return envelope.Data, e
}
func fmtStatus(status int) error {
	return &Fault{"DEPENDENCY_UNAVAILABLE", http.StatusText(status), 503}
}
func Serve(ctx context.Context, mux *http.ServeMux) error {
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		c, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(c)
	}()
	e := server.ListenAndServe()
	if errors.Is(e, http.ErrServerClosed) {
		return nil
	}
	return e
}
