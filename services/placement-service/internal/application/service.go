package application

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	p "github.com/logicore-wms/logicore-wms/libs/platform"
	d "github.com/logicore-wms/logicore-wms/services/placement-service/internal/domain"
	"log/slog"
	"net/http"
	"time"
)

type Service struct {
	R            *p.Runtime
	WarehouseURL string
	InventoryURL string
}

func New(r *p.Runtime) *Service {
	return &Service{r, p.Env("WAREHOUSE_URL", "http://warehouse-service:8080"), p.Env("INVENTORY_URL", "http://inventory-service:8080")}
}
func (s *Service) Routes(m *http.ServeMux) {
	m.HandleFunc("GET /placements", p.Wrap(func(_ http.ResponseWriter, q *http.Request) (any, error) {
		return p.Query(q.Context(), s.R.DB, "select to_jsonb(j) from placement_jobs j order by updated_at desc limit 500")
	}))
	m.HandleFunc("GET /placements/{id}", p.Wrap(func(_ http.ResponseWriter, q *http.Request) (any, error) {
		if e := p.ID(q.PathValue("id")); e != nil {
			return nil, e
		}
		rows, e := p.Query(q.Context(), s.R.DB, "select to_jsonb(j) from placement_jobs j where stock_id=$1", q.PathValue("id"))
		if e != nil {
			return nil, e
		}
		if len(rows) == 0 {
			return nil, &p.Fault{Code: "NOT_FOUND", Message: "Placement job not found", Status: 404}
		}
		return rows[0], nil
	}))
}
func (s *Service) Handle(ctx context.Context, tx pgx.Tx, event p.Event) error {
	v := event.Payload
	switch event.Type {
	case "GoodsReceived":
		_, e := tx.Exec(ctx, "insert into placement_jobs(stock_id,warehouse_id,payload) values($1,$2,$3) on conflict do nothing", p.String(v, "stockId"), p.String(v, "warehouseId"), v)
		return e
	case "PlacementRejected":
		_, e := tx.Exec(ctx, "update placement_jobs set status='PENDING',last_error=$2,next_attempt=now()+interval '3 seconds',updated_at=now() where stock_id=$1 and status<>'PLACED'", p.String(v, "stockId"), p.String(v, "reason"))
		return e
	case "StockAdded":
		_, e := tx.Exec(ctx, "update placement_jobs set status='PLACED',last_error=null,updated_at=now() where stock_id=$1", p.String(v, "stockId"))
		return e
	}
	return nil
}
func (s *Service) Work(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e := s.assign(ctx); e != nil {
				slog.Error("placement retry", "error", e)
			}
		}
	}
}
func (s *Service) assign(ctx context.Context) error {
	return s.R.Transaction(ctx, func(tx pgx.Tx) error {
		var stock, warehouse string
		var raw []byte
		e := tx.QueryRow(ctx, "select stock_id::text,warehouse_id::text,payload from placement_jobs where status='PENDING' and next_attempt<=now() order by next_attempt limit 1 for update skip locked").Scan(&stock, &warehouse, &raw)
		if e == pgx.ErrNoRows {
			return nil
		}
		if e != nil {
			return e
		}
		token := uuid.NewString()
		key := "placement:" + warehouse
		ok, e := s.R.Redis.SetNX(ctx, key, token, 15*time.Second).Result()
		if e != nil {
			return e
		}
		if !ok {
			return nil
		}
		defer s.R.Redis.Eval(ctx, "if redis.call('get',KEYS[1])==ARGV[1] then return redis.call('del',KEYS[1]) else return 0 end", []string{key}, token)
		var payload map[string]any
		if e = json.Unmarshal(raw, &payload); e != nil {
			return e
		}
		lot, e := p.Get[map[string]any](ctx, s.InventoryURL+"/inventory/lots/"+stock)
		if e != nil {
			return e
		}
		quantity := p.Number(lot, "quantity")
		if quantity == 0 {
			_, e = tx.Exec(ctx, "update placement_jobs set status='CANCELLED',last_error='NO_REMAINING_STOCK',updated_at=now() where stock_id=$1", stock)
			return e
		}
		if lot["locationId"] != nil {
			_, e = tx.Exec(ctx, "update placement_jobs set status='PLACED',last_error=null,updated_at=now() where stock_id=$1", stock)
			return e
		}
		payload["quantity"] = quantity
		locations, e := p.Get[[]d.Candidate](ctx, s.WarehouseURL+"/warehouses/"+warehouse+"/locations")
		if e != nil {
			return e
		}
		best := -1e9
		var chosen *d.Candidate
		for _, candidate := range locations {
			score, eligible := d.Score(candidate, warehouse, p.String(payload, "storageType"), p.Number(payload, "weight")*p.Number(payload, "quantity"), p.Number(payload, "volume")*p.Number(payload, "quantity"))
			if eligible && (score > best || (score == best && chosen != nil && candidate.ID < chosen.ID)) {
				copy := candidate
				chosen = &copy
				best = score
			}
		}
		if chosen == nil {
			s.R.Count("placement", "no_capacity")
			_, e = tx.Exec(ctx, "update placement_jobs set attempts=attempts+1,last_error='NO_SUITABLE_LOCATION',next_attempt=now()+interval '10 seconds',updated_at=now() where stock_id=$1", stock)
			return e
		}
		payload["locationId"] = chosen.ID
		payload["score"] = best
		if _, e = tx.Exec(ctx, "update placement_jobs set status='ASSIGNED',location_id=$2,score=$3,attempts=attempts+1,updated_at=now() where stock_id=$1", stock, chosen.ID, best); e != nil {
			return e
		}
		s.R.Count("placement", "assigned")
		return s.R.Emit(ctx, tx, "ProductPlacementAssigned", payload)
	})
}
