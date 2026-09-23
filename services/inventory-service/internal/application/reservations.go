package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	p "github.com/logicore-wms/logicore-wms/libs/platform"
	d "github.com/logicore-wms/logicore-wms/services/inventory-service/internal/domain"
	"log/slog"
	"time"
)

func (s *Service) Handle(ctx context.Context, tx pgx.Tx, e p.Event) error {
	switch e.Type {
	case "ReservationRequested", "ReservationCancellationRequested", "ShipmentRequested", "ProductPlacementAssigned":
	default:
		return nil
	}
	warehouse := p.String(e.Payload, "warehouseId")
	if err := p.Lock(ctx, tx, warehouse); err != nil {
		return err
	}
	// A savepoint rolls back partial business changes before recording a rejection.
	nested, err := tx.Begin(ctx)
	if err != nil {
		return err
	}
	switch e.Type {
	case "ReservationRequested":
		err = s.reserve(ctx, nested, e.Payload)
	case "ReservationCancellationRequested":
		err = s.release(ctx, nested, p.String(e.Payload, "reservationId"), false)
	case "ShipmentRequested":
		err = s.ship(ctx, nested, e.Payload)
	case "ProductPlacementAssigned":
		err = s.place(ctx, nested, e.Payload)
	}
	if err == nil {
		return nested.Commit(ctx)
	}
	nested.Rollback(ctx)
	var fault *p.Fault
	if !errors.As(err, &fault) || fault.Status >= 500 {
		return err
	}
	kind := map[string]string{"ReservationRequested": "ReservationRejected", "ShipmentRequested": "ShipmentRejected", "ProductPlacementAssigned": "PlacementRejected", "ReservationCancellationRequested": "ReservationCancellationRejected"}[e.Type]
	payload := map[string]any{}
	for k, v := range e.Payload {
		payload[k] = v
	}
	payload["reason"] = fault.Code
	if e.Type == "ReservationRequested" {
		_, er := tx.Exec(ctx, "insert into reservations(id,warehouse_id,status,expires_at) values($1,$2,'REJECTED',now()) on conflict do nothing", p.String(e.Payload, "reservationId"), warehouse)
		if er != nil {
			return er
		}
	}
	s.R.Count(kind, "rejected")
	return s.R.Emit(ctx, tx, kind, payload)
}
func (s *Service) reserve(ctx context.Context, tx pgx.Tx, payload map[string]any) error {
	id, warehouse := p.String(payload, "reservationId"), p.String(payload, "warehouseId")
	if e := p.ID(id); e != nil {
		return e
	}
	var existing string
	e := tx.QueryRow(ctx, "select status from reservations where id=$1", id).Scan(&existing)
	if e == nil {
		return nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	expires, e := time.Parse(time.RFC3339Nano, p.String(payload, "expiresAt"))
	if e != nil {
		return &p.Fault{Code: "VALIDATION_ERROR", Message: "Invalid expiry", Status: 400}
	}
	if !expires.After(time.Now()) {
		return p.Fail("EXPIRED", "Reservation request expired")
	}
	lines, e := p.Decode[[]d.Line](payload["items"])
	if e != nil || len(lines) == 0 {
		return &p.Fault{Code: "VALIDATION_ERROR", Message: "items required", Status: 400}
	}
	if _, e = tx.Exec(ctx, "insert into reservations(id,warehouse_id,status,expires_at) values($1,$2,'CONFIRMED',$3)", id, warehouse, expires); e != nil {
		return e
	}
	seen := map[string]bool{}
	for _, line := range lines {
		if line.Quantity <= 0 || seen[line.ProductID] {
			return &p.Fault{Code: "VALIDATION_ERROR", Message: "Invalid or duplicate line", Status: 400}
		}
		seen[line.ProductID] = true
		if e = p.ID(line.ProductID); e != nil {
			return e
		}
		rows, e := tx.Query(ctx, "select id::text,quantity-reserved_quantity from stocks where warehouse_id=$1 and product_id=$2 and location_id is not null and quantity>reserved_quantity and (expiration_date is null or expiration_date>current_date) order by expiration_date nulls last,created_at,id for update", warehouse, line.ProductID)
		if e != nil {
			return e
		}
		type lot struct {
			id string
			n  int64
		}
		lots := []lot{}
		for rows.Next() {
			var l lot
			if e = rows.Scan(&l.id, &l.n); e != nil {
				rows.Close()
				return e
			}
			lots = append(lots, l)
		}
		rows.Close()
		if rows.Err() != nil {
			return rows.Err()
		}
		remaining := line.Quantity
		for _, l := range lots {
			take := min(remaining, l.n)
			if take <= 0 {
				break
			}
			if _, e = tx.Exec(ctx, "update stocks set reserved_quantity=reserved_quantity+$2,updated_at=now() where id=$1", l.id, take); e != nil {
				return e
			}
			if _, e = tx.Exec(ctx, "insert into allocations(reservation_id,stock_id,quantity) values($1,$2,$3)", id, l.id, take); e != nil {
				return e
			}
			remaining -= take
		}
		if remaining > 0 {
			return p.Fail("INSUFFICIENT_STOCK", "Not enough placed, unexpired stock")
		}
	}
	s.R.Count("reserve", "success")
	return s.R.Emit(ctx, tx, "StockReserved", payload)
}
func (s *Service) release(ctx context.Context, tx pgx.Tx, id string, expired bool) error {
	var status, warehouse string
	e := tx.QueryRow(ctx, "select status,warehouse_id::text from reservations where id=$1 for update", id).Scan(&status, &warehouse)
	if errors.Is(e, pgx.ErrNoRows) {
		return fmt.Errorf("reservation command not yet processed")
	}
	if e != nil {
		return e
	}
	if status == "COMPLETED" {
		return p.Fail("ALREADY_SHIPPED", "Reservation already shipped")
	}
	if status != "CONFIRMED" {
		return nil
	}
	if _, e = tx.Exec(ctx, "update stocks s set reserved_quantity=s.reserved_quantity-a.quantity,updated_at=now() from allocations a where a.reservation_id=$1 and s.id=a.stock_id", id); e != nil {
		return e
	}
	kind, next := "StockReservationCancelled", "CANCELLED"
	if expired {
		kind, next = "ReservationExpired", "EXPIRED"
	}
	if _, e = tx.Exec(ctx, "update reservations set status=$2 where id=$1", id, next); e != nil {
		return e
	}
	items, err := p.Query(ctx, tx, `select jsonb_build_object('productId',s.product_id,'stockId',s.id,'quantity',a.quantity) from allocations a join stocks s on s.id=a.stock_id where a.reservation_id=$1 order by s.id`, id)
	if err != nil {
		return err
	}
	return s.R.Emit(ctx, tx, kind, map[string]any{"reservationId": id, "warehouseId": warehouse, "items": items})
}
func (s *Service) ship(ctx context.Context, tx pgx.Tx, payload map[string]any) error {
	id, shipment := p.String(payload, "reservationId"), p.String(payload, "shipmentId")
	var status, warehouse string
	var expires time.Time
	var previous *string
	err := tx.QueryRow(ctx, "select status,expires_at,warehouse_id::text,shipment_id::text from reservations where id=$1 for update", id).Scan(&status, &expires, &warehouse, &previous)
	if errors.Is(err, pgx.ErrNoRows) {
		return p.Fail("RESERVATION_REQUIRED", "Unknown reservation")
	}
	if err != nil {
		return err
	}
	if status == "COMPLETED" && previous != nil && *previous == shipment {
		return nil
	}
	if status != "CONFIRMED" || !expires.After(time.Now()) {
		return p.Fail("RESERVATION_UNAVAILABLE", "Reservation is not active")
	}
	if warehouse != p.String(payload, "warehouseId") {
		return p.Fail("WRONG_WAREHOUSE", "Reservation warehouse mismatch")
	}
	rows, err := tx.Query(ctx, "select stock_id::text,quantity from allocations where reservation_id=$1 order by stock_id", id)
	if err != nil {
		return err
	}
	type allocation struct {
		id string
		n  int64
	}
	allocs := []allocation{}
	for rows.Next() {
		var a allocation
		if err = rows.Scan(&a.id, &a.n); err != nil {
			rows.Close()
			return err
		}
		allocs = append(allocs, a)
	}
	rows.Close()
	if rows.Err() != nil {
		return rows.Err()
	}
	items := []map[string]any{}
	for _, a := range allocs {
		x, e := s.stock(ctx, tx, a.id)
		if e != nil {
			return e
		}
		if x.Expiration != nil && !x.Expiration.After(time.Now().UTC().Truncate(24*time.Hour)) {
			return p.Fail("EXPIRED_STOCK", "Allocated lot expired")
		}
		if x.LocationID == nil {
			return fmt.Errorf("reserved stock is unplaced")
		}
		if e = s.capacity(ctx, tx, x, *x.LocationID, -a.n); e != nil {
			return e
		}
		if _, e = tx.Exec(ctx, "update stocks set quantity=quantity-$2,reserved_quantity=reserved_quantity-$2,updated_at=now() where id=$1", a.id, a.n); e != nil {
			return e
		}
		items = append(items, map[string]any{"productId": x.ProductID, "stockId": x.ID, "quantity": a.n})
	}
	if _, err = tx.Exec(ctx, "update reservations set status='COMPLETED',shipment_id=$2 where id=$1", id, shipment); err != nil {
		return err
	}
	payload["items"] = items
	s.R.Count("ship", "success")
	return s.R.Emit(ctx, tx, "StockShipped", payload)
}
func (s *Service) place(ctx context.Context, tx pgx.Tx, payload map[string]any) error {
	x, e := s.stock(ctx, tx, p.String(payload, "stockId"))
	if e != nil {
		return e
	}
	if x.WarehouseID != p.String(payload, "warehouseId") {
		return p.Fail("WRONG_WAREHOUSE", "Placement warehouse mismatch")
	}
	if x.LocationID != nil || x.Quantity == 0 {
		return nil
	}
	location := p.String(payload, "locationId")
	if e = p.ID(location); e != nil {
		return e
	}
	if e = s.capacity(ctx, tx, x, location, x.Quantity); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, "update stocks set location_id=$2,updated_at=now() where id=$1", x.ID, location); e != nil {
		return e
	}
	return s.R.Emit(ctx, tx, "StockAdded", map[string]any{"stockId": x.ID, "productId": x.ProductID, "warehouseId": x.WarehouseID, "locationId": location, "quantity": x.Quantity})
}
func (s *Service) Work(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, e := s.R.DB.Query(ctx, "select id::text,warehouse_id::text from reservations where status='CONFIRMED' and expires_at<=now() limit 100")
			if e != nil {
				slog.Error("expiry scan", "error", e)
				continue
			}
			type due struct{ id, warehouse string }
			batch := []due{}
			for rows.Next() {
				var x due
				if e = rows.Scan(&x.id, &x.warehouse); e != nil {
					break
				}
				batch = append(batch, x)
			}
			rows.Close()
			for _, x := range batch {
				e = s.R.Transaction(ctx, func(tx pgx.Tx) error {
					if e := p.Lock(ctx, tx, x.warehouse); e != nil {
						return e
					}
					return s.release(ctx, tx, x.id, true)
				})
				if e != nil {
					slog.Error("expiry retry", "error", e)
				}
			}
		}
	}
}
