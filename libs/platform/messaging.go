package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"log/slog"
	"time"
)

type Handler func(context.Context, pgx.Tx, Event) error

func (r *Runtime) Outbox(ctx context.Context) {
	tick := time.NewTicker(300 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			err := r.Transaction(ctx, func(tx pgx.Tx) error {
				rows, e := tx.Query(ctx, "select id::text,aggregate_id,payload from outbox_events where published_at is null order by sequence limit 50 for update skip locked")
				if e != nil {
					return e
				}
				type pending struct {
					id, key string
					body    []byte
				}
				batch := []pending{}
				for rows.Next() {
					var p pending
					if e = rows.Scan(&p.id, &p.key, &p.body); e != nil {
						rows.Close()
						return e
					}
					batch = append(batch, p)
				}
				rows.Close()
				if rows.Err() != nil {
					return rows.Err()
				}
				for _, p := range batch {
					if e = r.Writer.WriteMessages(ctx, kafka.Message{Topic: "wms.events", Key: []byte(p.key), Value: p.body}); e != nil {
						return e
					}
					if _, e = tx.Exec(ctx, "update outbox_events set published_at=now() where id=$1", p.id); e != nil {
						return e
					}
				}
				return nil
			})
			if err != nil {
				slog.Error("outbox retry", "error", err)
			}
		}
	}
}
func (r *Runtime) Consume(ctx context.Context, handler Handler) {
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: r.Brokers, Topic: "wms.events", GroupID: r.Name, StartOffset: kafka.FirstOffset, CommitInterval: 0, MaxBytes: 10e6})
	defer reader.Close()
	for ctx.Err() == nil {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() == nil {
				slog.Error("Kafka fetch", "error", err)
			}
			continue
		}
		var event Event
		err = json.Unmarshal(msg.Value, &event)
		if err == nil && (event.Version != 1 || ID(event.ID) != nil) {
			err = fmt.Errorf("invalid event envelope")
		}
		parseErr := err
		for attempt := 0; ctx.Err() == nil; attempt++ {
			err = parseErr
			if err == nil {
				carrier := propagation.MapCarrier{"traceparent": event.Traceparent}
				child := otel.GetTextMapPropagator().Extract(ctx, carrier)
				child = context.WithValue(child, contextKey("correlation"), event.Correlation)
				child, span := otel.Tracer(r.Name).Start(child, "consume "+event.Type)
				err = r.Process(child, event, handler)
				span.End()
			}
			if err == nil {
				r.Count("kafka", "success")
				break
			}
			r.Count("kafka", "failure")
			slog.Error("event retry", "eventId", event.ID, "correlationId", event.Correlation, "error", err)
			if attempt >= 4 {
				dltErr := r.Writer.WriteMessages(ctx, kafka.Message{Topic: "wms.events.DLT", Key: msg.Key, Value: msg.Value, Headers: []kafka.Header{{Key: "consumer", Value: []byte(r.Name)}, {Key: "error", Value: []byte(err.Error())}}})
				if dltErr == nil {
					err = nil
					break
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}
		if err == nil {
			if e := reader.CommitMessages(ctx, msg); e != nil {
				slog.Error("Kafka commit", "error", e)
			}
		}
	}
}

// Process commits the inbox marker and domain changes together.
func (r *Runtime) Process(ctx context.Context, event Event, handler Handler) error {
	return r.Transaction(ctx, func(tx pgx.Tx) error {
		tag, e := tx.Exec(ctx, "insert into processed_events(event_id,consumer) values($1,$2) on conflict do nothing", event.ID, r.Name)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return nil
		}
		return handler(ctx, tx, event)
	})
}
