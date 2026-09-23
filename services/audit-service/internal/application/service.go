package application

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	p "github.com/logicore-wms/logicore-wms/libs/platform"
	"net/http"
	"strconv"
)

type Service struct{ R *p.Runtime }

func New(r *p.Runtime) *Service             { return &Service{r} }
func (s *Service) Work(ctx context.Context) { <-ctx.Done() }
func (s *Service) Handle(ctx context.Context, tx pgx.Tx, e p.Event) error {
	kind, id := "event", e.ID
	for _, field := range []string{"stockId", "shipmentId", "reservationId", "locationId", "productId", "warehouseId", "id"} {
		if v := p.String(e.Payload, field); v != "" {
			kind, id = field, v
			break
		}
	}
	_, err := tx.Exec(ctx, "insert into audit_events(id,event_id,event_type,aggregate_type,aggregate_id,payload,timestamp,correlation_id,source) values($1,$1,$2,$3,$4,$5,$6,$7,$8) on conflict do nothing", e.ID, e.Type, kind, id, e.Payload, e.Timestamp, e.Correlation, e.Source)
	return err
}
func (s *Service) Routes(m *http.ServeMux) {
	m.HandleFunc("GET /audit/events", p.Wrap(s.list))
	m.HandleFunc("GET /audit/products/{product}", p.Wrap(s.list))
	m.HandleFunc("GET /audit/warehouses/{warehouse}", p.Wrap(s.list))
	m.HandleFunc("GET /audit/events/{id}", p.Wrap(func(_ http.ResponseWriter, q *http.Request) (any, error) {
		if e := p.ID(q.PathValue("id")); e != nil {
			return nil, e
		}
		rows, e := p.Query(q.Context(), s.R.DB, "select to_jsonb(a) from audit_events a where id=$1", q.PathValue("id"))
		if e != nil {
			return nil, e
		}
		if len(rows) == 0 {
			return nil, &p.Fault{Code: "NOT_FOUND", Message: "Event not found", Status: 404}
		}
		return rows[0], nil
	}))
}
func (s *Service) list(_ http.ResponseWriter, q *http.Request) (any, error) {
	product, warehouse := q.PathValue("product"), q.PathValue("warehouse")
	for _, id := range []string{product, warehouse} {
		if id != "" {
			if e := p.ID(id); e != nil {
				return nil, e
			}
		}
	}
	limit, offset := 100, 0
	if v := q.URL.Query().Get("limit"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > 500 {
			return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "limit 1..500", Status: 400}
		}
		limit = n
	}
	if v := q.URL.Query().Get("offset"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 0 {
			return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "offset >= 0", Status: 400}
		}
		offset = n
	}
	query := `select to_jsonb(a) from audit_events a where ($1='' or payload->>'productId'=$1 or exists(select 1 from jsonb_array_elements(coalesce(payload->'items','[]'::jsonb)) item where item->>'productId'=$1)) and ($2='' or payload->>'warehouseId'=$2) and ($3='' or event_type=$3) order by timestamp,id limit $4 offset $5`
	rows, e := p.Query(q.Context(), s.R.DB, query, product, warehouse, q.URL.Query().Get("eventType"), limit, offset)
	if e != nil {
		return nil, fmt.Errorf("audit query: %w", e)
	}
	return rows, nil
}
