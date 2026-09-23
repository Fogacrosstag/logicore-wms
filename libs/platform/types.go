package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"os"
	"time"
)

type Event struct {
	ID          string         `json:"eventId"`
	Type        string         `json:"eventType"`
	Version     int            `json:"eventVersion"`
	Timestamp   time.Time      `json:"timestamp"`
	Source      string         `json:"source"`
	Correlation string         `json:"correlationId"`
	Traceparent string         `json:"traceparent"`
	Payload     map[string]any `json:"payload"`
}
type Fault struct {
	Code    string
	Message string
	Status  int
}

func (e *Fault) Error() string         { return e.Code + ": " + e.Message }
func Fail(code, message string) *Fault { return &Fault{code, message, 409} }
func Env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func String(p map[string]any, k string) string { v, _ := p[k].(string); return v }
func Number(p map[string]any, k string) float64 {
	switch v := p[k].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	}
	return 0
}
func Decode[T any](v any) (T, error) {
	var x T
	b, e := json.Marshal(v)
	if e != nil {
		return x, e
	}
	e = json.Unmarshal(b, &x)
	return x, e
}
func ID(v string) error {
	if _, e := uuid.Parse(v); e != nil {
		return &Fault{"VALIDATION_ERROR", fmt.Sprintf("Invalid UUID: %s", v), 400}
	}
	return nil
}

type contextKey string

func Correlation(ctx context.Context) string {
	if v, ok := ctx.Value(contextKey("correlation")).(string); ok {
		return v
	}
	return uuid.NewString()
}
