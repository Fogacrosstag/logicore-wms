package platform

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"time"
)

func (r *Runtime) Transaction(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, e := r.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func Lock(ctx context.Context, tx pgx.Tx, warehouse string) error {
	if e := ID(warehouse); e != nil {
		return e
	}
	_, e := tx.Exec(ctx, "select pg_advisory_xact_lock(hashtextextended($1,0))", warehouse)
	return e
}
func (r *Runtime) Emit(ctx context.Context, tx pgx.Tx, kind string, payload map[string]any) error {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	event := Event{uuid.NewString(), kind, 1, time.Now().UTC(), r.Name, Correlation(ctx), carrier.Get("traceparent"), payload}
	body, e := json.Marshal(event)
	if e != nil {
		return e
	}
	key := String(payload, "warehouseId")
	if key == "" {
		key = event.ID
	}
	_, e = tx.Exec(ctx, "insert into outbox_events(id,aggregate_id,event_type,payload) values($1,$2,$3,$4)", event.ID, key, kind, body)
	return e
}
func (r *Runtime) Command(ctx context.Context, key string, request any, fn func(pgx.Tx) (any, error)) (any, error) {
	if len(key) == 0 || len(key) > 128 {
		return nil, &Fault{"VALIDATION_ERROR", "Idempotency-Key is required (1-128 characters)", 400}
	}
	raw, e := json.Marshal(request)
	if e != nil {
		return nil, e
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	var result any
	e = r.Transaction(ctx, func(tx pgx.Tx) error {
		if _, e := tx.Exec(ctx, "select pg_advisory_xact_lock(hashtextextended($1,1))", key); e != nil {
			return e
		}
		var previousHash string
		var saved []byte
		err := tx.QueryRow(ctx, "select fingerprint,response from http_commands where id=$1", key).Scan(&previousHash, &saved)
		if err == nil {
			if previousHash != hash {
				return Fail("IDEMPOTENCY_CONFLICT", "Key reused with different request")
			}
			return json.Unmarshal(saved, &result)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		result, err = fn(tx)
		if err != nil {
			return err
		}
		body, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "insert into http_commands(id,fingerprint,response) values($1,$2,$3)", key, hash, body)
		return err
	})
	return result, e
}
func Query(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, sql string, args ...any) ([]map[string]any, error) {
	rows, e := db.Query(ctx, sql, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var b []byte
		if e = rows.Scan(&b); e != nil {
			return nil, e
		}
		var x map[string]any
		if e = json.Unmarshal(b, &x); e != nil {
			return nil, e
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
