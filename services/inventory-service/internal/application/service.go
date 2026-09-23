package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	p "github.com/logicore-wms/logicore-wms/libs/platform"
	d "github.com/logicore-wms/logicore-wms/services/inventory-service/internal/domain"
	"math"
	"net/http"
	"time"
)

type Service struct {
	R                        *p.Runtime
	WarehouseURL, ProductURL string
}

func New(r *p.Runtime) *Service {
	return &Service{r, p.Env("WAREHOUSE_URL", "http://warehouse-service:8080"), p.Env("PRODUCT_URL", "http://product-service:8080")}
}

type Product struct {
	ID             string `json:"id"`
	Weight, Volume float64
	StorageType    string
	Active         bool
}
type Location struct {
	ID          string  `json:"id"`
	WarehouseID string  `json:"warehouseId"`
	MaxWeight   float64 `json:"maxWeight"`
	MaxVolume   float64 `json:"maxVolume"`
	StorageType string  `json:"storageType"`
	ZoneType    string  `json:"zoneType"`
	Status      string  `json:"status"`
}
type Receive struct {
	ProductID   string `json:"productId"`
	WarehouseID string `json:"warehouseId"`
	Quantity    int64  `json:"quantity"`
	Batch       string `json:"batchNumber"`
	Expiration  string `json:"expirationDate"`
}
type Change struct {
	StockID      string `json:"stockId"`
	ToLocationID string `json:"toLocationId"`
	Quantity     int64  `json:"quantity"`
	Reason       string `json:"reason"`
}

func (s *Service) stock(ctx context.Context, tx pgx.Tx, id string) (d.Stock, error) {
	var x d.Stock
	err := tx.QueryRow(ctx, "select id::text,product_id::text,warehouse_id::text,location_id::text,quantity,reserved_quantity,batch_number,expiration_date,unit_weight,unit_volume,storage_type from stocks where id=$1", id).Scan(&x.ID, &x.ProductID, &x.WarehouseID, &x.LocationID, &x.Quantity, &x.Reserved, &x.Batch, &x.Expiration, &x.Weight, &x.Volume, &x.StorageType)
	if errors.Is(err, pgx.ErrNoRows) {
		err = &p.Fault{Code: "NOT_FOUND", Message: "Stock lot not found", Status: 404}
	}
	return x, err
}
func (s *Service) insert(ctx context.Context, tx pgx.Tx, x d.Stock) error {
	_, e := tx.Exec(ctx, "insert into stocks(id,product_id,warehouse_id,location_id,quantity,batch_number,expiration_date,unit_weight,unit_volume,storage_type) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)", x.ID, x.ProductID, x.WarehouseID, x.LocationID, x.Quantity, x.Batch, x.Expiration, x.Weight, x.Volume, x.StorageType)
	return e
}
func (s *Service) capacity(ctx context.Context, tx pgx.Tx, x d.Stock, location string, delta int64) error {
	var loc Location
	if delta > 0 {
		var e error
		loc, e = p.Get[Location](ctx, s.WarehouseURL+"/locations/"+location)
		if e != nil {
			return e
		}
		if loc.WarehouseID != x.WarehouseID {
			return p.Fail("WRONG_WAREHOUSE", "Location belongs to another warehouse")
		}
		if loc.Status == "BLOCKED" || loc.Status == "MAINTENANCE" {
			return p.Fail("LOCATION_BLOCKED", "Location is unavailable")
		}
		if loc.ZoneType != "STORAGE" || loc.StorageType != x.StorageType {
			return p.Fail("INCOMPATIBLE_LOCATION", "Storage restrictions do not match")
		}
	}
	if _, e := tx.Exec(ctx, "insert into capacities(location_id,warehouse_id) values($1,$2) on conflict do nothing", location, x.WarehouseID); e != nil {
		return e
	}
	var weight, volume float64
	var rev int64
	if e := tx.QueryRow(ctx, "select current_weight,current_volume,revision from capacities where location_id=$1 for update", location).Scan(&weight, &volume, &rev); e != nil {
		return e
	}
	weight += float64(delta) * x.Weight
	volume += float64(delta) * x.Volume
	if delta > 0 && (weight > loc.MaxWeight+1e-9 || volume > loc.MaxVolume+1e-9) {
		return p.Fail("LOCATION_FULL", "Location capacity exceeded")
	}
	if weight < -1e-7 || volume < -1e-7 {
		return fmt.Errorf("capacity invariant violated")
	}
	weight = math.Max(0, weight)
	volume = math.Max(0, volume)
	rev++
	if _, e := tx.Exec(ctx, "update capacities set current_weight=$2,current_volume=$3,revision=$4 where location_id=$1", location, weight, volume, rev); e != nil {
		return e
	}
	return s.R.Emit(ctx, tx, "LocationCapacityChanged", map[string]any{"locationId": location, "warehouseId": x.WarehouseID, "currentWeight": weight, "currentVolume": volume, "revision": rev})
}
func (s *Service) Receive(ctx context.Context, tx pgx.Tx, q Receive) (any, error) {
	if e := p.ID(q.ProductID); e != nil {
		return nil, e
	}
	if e := p.Lock(ctx, tx, q.WarehouseID); e != nil {
		return nil, e
	}
	if q.Quantity <= 0 || q.Quantity > 1_000_000_000 || q.Batch == "" {
		return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "Positive quantity <= 1 billion and batchNumber required", Status: 400}
	}
	product, e := p.Get[Product](ctx, s.ProductURL+"/products/"+q.ProductID)
	if e != nil {
		return nil, e
	}
	if !product.Active {
		return nil, p.Fail("INACTIVE_PRODUCT", "Product is inactive")
	}
	warehouse, e := p.Get[map[string]any](ctx, s.WarehouseURL+"/warehouses/"+q.WarehouseID)
	if e != nil {
		return nil, e
	}
	if warehouse["status"] != "ACTIVE" {
		return nil, p.Fail("WAREHOUSE_UNAVAILABLE", "Warehouse inactive")
	}
	var expires *time.Time
	if q.Expiration != "" {
		v, e := time.Parse("2006-01-02", q.Expiration)
		if e != nil || !v.After(time.Now().UTC().Truncate(24*time.Hour)) {
			return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "expirationDate must be a future ISO date", Status: 400}
		}
		expires = &v
	}
	x := d.Stock{ID: uuid.NewString(), ProductID: q.ProductID, WarehouseID: q.WarehouseID, Quantity: q.Quantity, Batch: q.Batch, Expiration: expires, Weight: product.Weight, Volume: product.Volume, StorageType: product.StorageType}
	if e = s.insert(ctx, tx, x); e != nil {
		return nil, e
	}
	e = s.R.Emit(ctx, tx, "GoodsReceived", map[string]any{"stockId": x.ID, "productId": x.ProductID, "warehouseId": x.WarehouseID, "quantity": x.Quantity, "weight": x.Weight, "volume": x.Volume, "storageType": x.StorageType})
	return x, e
}
func (s *Service) Change(ctx context.Context, tx pgx.Tx, action string, q Change) (any, error) {
	if e := p.ID(q.StockID); e != nil {
		return nil, e
	}
	x, e := s.stock(ctx, tx, q.StockID)
	if e != nil {
		return nil, e
	}
	if e = p.Lock(ctx, tx, x.WarehouseID); e != nil {
		return nil, e
	}
	x, e = s.stock(ctx, tx, q.StockID)
	if e != nil {
		return nil, e
	}
	if action == "adjust" {
		if q.Quantity < 0 || q.Quantity > 1_000_000_000 {
			return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "Target quantity must be 0..1 billion", Status: 400}
		}
		if q.Quantity < x.Reserved {
			return nil, p.Fail("RESERVED_STOCK", "Cannot adjust below reserved quantity")
		}
		delta := q.Quantity - x.Quantity
		if q.Reason == "" {
			return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "reason required", Status: 400}
		}
		if x.LocationID == nil && delta > 0 {
			return nil, p.Fail("UNPLACED_STOCK", "Receive new unplaced stock separately")
		}
		if x.LocationID != nil && delta != 0 {
			if e = s.capacity(ctx, tx, x, *x.LocationID, delta); e != nil {
				return nil, e
			}
		}
		_, e = tx.Exec(ctx, "update stocks set quantity=$2,updated_at=now() where id=$1", x.ID, q.Quantity)
		if e != nil {
			return nil, e
		}
		return map[string]any{"stockId": x.ID, "quantity": q.Quantity}, s.R.Emit(ctx, tx, "StockAdjusted", map[string]any{"stockId": x.ID, "productId": x.ProductID, "warehouseId": x.WarehouseID, "quantity": q.Quantity, "delta": delta, "reason": q.Reason})
	}
	if e = d.CheckQuantity(q.Quantity, d.Available(x.Quantity, x.Reserved)); e != nil {
		return nil, e
	}
	payload := map[string]any{"stockId": x.ID, "productId": x.ProductID, "warehouseId": x.WarehouseID, "quantity": q.Quantity}
	event := "StockWrittenOff"
	if action == "move" {
		if e = p.ID(q.ToLocationID); e != nil {
			return nil, e
		}
		if x.LocationID != nil && *x.LocationID == q.ToLocationID {
			return nil, p.Fail("SAME_LOCATION", "Source equals destination")
		}
		if e = s.capacity(ctx, tx, x, q.ToLocationID, q.Quantity); e != nil {
			return nil, e
		}
		copy := x
		copy.ID = uuid.NewString()
		copy.LocationID = &q.ToLocationID
		copy.Quantity = q.Quantity
		copy.Reserved = 0
		if e = s.insert(ctx, tx, copy); e != nil {
			return nil, e
		}
		payload["toLocationId"] = q.ToLocationID
		payload["destinationStockId"] = copy.ID
		payload["fromLocationId"] = x.LocationID
		event = "StockMoved"
	} else if q.Reason == "" {
		return nil, &p.Fault{Code: "VALIDATION_ERROR", Message: "reason required", Status: 400}
	} else {
		payload["reason"] = q.Reason
	}
	if x.LocationID != nil {
		if e = s.capacity(ctx, tx, x, *x.LocationID, -q.Quantity); e != nil {
			return nil, e
		}
	}
	if _, e = tx.Exec(ctx, "update stocks set quantity=quantity-$2,updated_at=now() where id=$1", x.ID, q.Quantity); e != nil {
		return nil, e
	}
	return payload, s.R.Emit(ctx, tx, event, payload)
}
func (s *Service) Routes(m *http.ServeMux) {
	m.HandleFunc("GET /inventory", p.Wrap(s.list))
	m.HandleFunc("GET /inventory/products/{product}", p.Wrap(s.list))
	m.HandleFunc("GET /inventory/warehouses/{warehouse}", p.Wrap(s.list))
	m.HandleFunc("GET /inventory/lots/{id}", p.Wrap(func(_ http.ResponseWriter, q *http.Request) (any, error) {
		if e := p.ID(q.PathValue("id")); e != nil {
			return nil, e
		}
		rows, e := p.Query(q.Context(), s.R.DB, "select to_jsonb(s) from stock_view s where id=$1", q.PathValue("id"))
		if e != nil {
			return nil, e
		}
		if len(rows) == 0 {
			return nil, &p.Fault{Code: "NOT_FOUND", Message: "Stock not found", Status: 404}
		}
		return rows[0], nil
	}))
	m.HandleFunc("POST /inventory/receive", p.Wrap(func(w http.ResponseWriter, q *http.Request) (any, error) {
		body, e := p.Body[Receive](w, q)
		if e != nil {
			return nil, e
		}
		return s.R.Command(q.Context(), q.Header.Get("Idempotency-Key"), map[string]any{"action": "receive", "body": body}, func(tx pgx.Tx) (any, error) { return s.Receive(q.Context(), tx, body) })
	}))
	for _, action := range []string{"move", "write-off", "adjust"} {
		m.HandleFunc("POST /inventory/"+action, p.Wrap(func(w http.ResponseWriter, q *http.Request) (any, error) {
			body, e := p.Body[Change](w, q)
			if e != nil {
				return nil, e
			}
			return s.R.Command(q.Context(), q.Header.Get("Idempotency-Key"), map[string]any{"action": action, "body": body}, func(tx pgx.Tx) (any, error) { return s.Change(q.Context(), tx, action, body) })
		}))
	}
}
func (s *Service) list(_ http.ResponseWriter, q *http.Request) (any, error) {
	product, warehouse := q.PathValue("product"), q.PathValue("warehouse")
	if product == "" {
		product = q.URL.Query().Get("productId")
	}
	if warehouse == "" {
		warehouse = q.URL.Query().Get("warehouseId")
	}
	for _, id := range []string{product, warehouse} {
		if id != "" {
			if e := p.ID(id); e != nil {
				return nil, e
			}
		}
	}
	return p.Query(q.Context(), s.R.DB, `select to_jsonb(s) from stock_view s where ($1='' or "productId"::text=$1) and ($2='' or "warehouseId"::text=$2) order by "createdAt",id limit 500`, product, warehouse)
}
