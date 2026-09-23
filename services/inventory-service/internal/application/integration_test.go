//go:build integration

package application

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	p "github.com/logicore-wms/logicore-wms/libs/platform"
	root "github.com/logicore-wms/logicore-wms/services/inventory-service"
	d "github.com/logicore-wms/logicore-wms/services/inventory-service/internal/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestInventoryTransactions(t *testing.T) {
	ctx := context.Background()
	container, e := postgres.Run(ctx, "postgres:16.9-alpine", postgres.WithDatabase("test"), postgres.WithUsername("test"), postgres.WithPassword("test"), postgres.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { container.Terminate(ctx) })
	dsn, e := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	pool, e := pgxpool.New(ctx, dsn)
	require.NoError(t, e)
	defer pool.Close()
	for _, file := range []string{"migrations/000001_messaging.up.sql", "migrations/000002_inventory.up.sql"} {
		b, e := root.Migrations.ReadFile(file)
		require.NoError(t, e)
		_, e = pool.Exec(ctx, string(b))
		require.NoError(t, e)
	}
	r := &p.Runtime{DB: pool, Name: "inventory-service", Metrics: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "test_ops", Help: "test"}, []string{"service", "operation", "outcome"})}
	app := New(r)
	warehouse, product, location, full, blocked := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, q *http.Request) {
		var data any
		switch q.URL.Path {
		case "/products/" + product:
			data = Product{ID: product, Weight: 2, Volume: 1, StorageType: "NORMAL", Active: true}
		case "/warehouses/" + warehouse:
			data = map[string]any{"status": "ACTIVE"}
		default:
			id := q.URL.Path[len("/locations/"):]
			maxWeight, status := 100.0, "AVAILABLE"
			if id == full {
				maxWeight = 1
			}
			if id == blocked {
				status = "BLOCKED"
			}
			data = Location{ID: id, WarehouseID: warehouse, MaxWeight: maxWeight, MaxVolume: 100, StorageType: "NORMAL", ZoneType: "STORAGE", Status: status}
		}
		json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	defer api.Close()
	app.ProductURL = api.URL
	app.WarehouseURL = api.URL
	var stock d.Stock
	require.NoError(t, r.Transaction(ctx, func(tx pgx.Tx) error {
		x, e := app.Receive(ctx, tx, Receive{ProductID: product, WarehouseID: warehouse, Quantity: 20, Batch: "B1"})
		if e == nil {
			stock = x.(d.Stock)
		}
		return e
	}))
	event := p.Event{ID: uuid.NewString(), Type: "ProductPlacementAssigned", Version: 1, Payload: map[string]any{"stockId": stock.ID, "warehouseId": warehouse, "locationId": location}}
	require.NoError(t, r.Process(ctx, event, app.Handle))
	require.NoError(t, r.Process(ctx, event, app.Handle))
	var weight float64
	require.NoError(t, pool.QueryRow(ctx, "select current_weight from capacities where location_id=$1", location).Scan(&weight))
	require.Equal(t, 40.0, weight)
	for _, target := range []string{full, blocked} {
		e = r.Transaction(ctx, func(tx pgx.Tx) error {
			_, e := app.Change(ctx, tx, "move", Change{StockID: stock.ID, ToLocationID: target, Quantity: 1})
			return e
		})
		require.Error(t, e)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	ids := []string{uuid.NewString(), uuid.NewString()}
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			event := p.Event{ID: uuid.NewString(), Type: "ReservationRequested", Version: 1, Payload: map[string]any{"reservationId": id, "warehouseId": warehouse, "expiresAt": time.Now().Add(time.Minute).Format(time.RFC3339Nano), "items": []d.Line{{ProductID: product, Quantity: 15}}}}
			errs <- r.Process(ctx, event, app.Handle)
		}(id)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	var confirmed, rejected int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) filter(where status='CONFIRMED'),count(*) filter(where status='REJECTED') from reservations").Scan(&confirmed, &rejected))
	require.Equal(t, 1, confirmed)
	require.Equal(t, 1, rejected)
	var reserved, quantity int64
	require.NoError(t, pool.QueryRow(ctx, "select quantity,reserved_quantity from stocks where id=$1", stock.ID).Scan(&quantity, &reserved))
	require.Equal(t, int64(20), quantity)
	require.Equal(t, int64(15), reserved)
	var winner string
	require.NoError(t, pool.QueryRow(ctx, "select id::text from reservations where status='CONFIRMED'").Scan(&winner))
	shipment := p.Event{ID: uuid.NewString(), Type: "ShipmentRequested", Version: 1, Payload: map[string]any{"warehouseId": warehouse, "reservationId": winner, "shipmentId": uuid.NewString()}}
	require.NoError(t, r.Process(ctx, shipment, app.Handle))
	require.NoError(t, r.Process(ctx, shipment, app.Handle))
	shipment.ID = uuid.NewString()
	require.NoError(t, r.Process(ctx, shipment, app.Handle))
	require.NoError(t, pool.QueryRow(ctx, "select quantity,reserved_quantity from stocks where id=$1", stock.ID).Scan(&quantity, &reserved))
	require.Equal(t, int64(5), quantity)
	require.Zero(t, reserved)
	// HTTP idempotency persists the first result and rejects changed bodies.
	calls := 0
	fn := func(tx pgx.Tx) (any, error) { calls++; return map[string]any{"ok": true}, nil }
	_, e = r.Command(ctx, "key", map[string]any{"n": 1}, fn)
	require.NoError(t, e)
	_, e = r.Command(ctx, "key", map[string]any{"n": 1}, fn)
	require.NoError(t, e)
	_, e = r.Command(ctx, "key", map[string]any{"n": 2}, fn)
	require.Error(t, e)
	require.Equal(t, 1, calls)
	// A failed handler must not leave an inbox marker.
	bad := p.Event{ID: uuid.NewString()}
	require.Error(t, r.Process(ctx, bad, func(context.Context, pgx.Tx, p.Event) error { return p.Fail("FAIL", "rollback") }))
	var n int
	require.NoError(t, pool.QueryRow(ctx, "select count(*) from processed_events where event_id=$1", bad.ID).Scan(&n))
	require.Zero(t, n)
}
