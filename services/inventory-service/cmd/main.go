package main

import (
	"context"
	"github.com/logicore-wms/logicore-wms/libs/platform"
	service "github.com/logicore-wms/logicore-wms/services/inventory-service"
	"github.com/logicore-wms/logicore-wms/services/inventory-service/internal/application"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	r, e := platform.Open(ctx, "inventory-service", service.Migrations)
	if e != nil {
		slog.Error("startup failed", "error", e)
		os.Exit(1)
	}
	defer r.Close()
	app := application.New(r)
	mux := r.Mux()
	app.Routes(mux)
	go r.Outbox(ctx)
	go r.Consume(ctx, app.Handle)
	go app.Work(ctx)
	if e = platform.Serve(ctx, mux); e != nil {
		slog.Error("server failed", "error", e)
		os.Exit(1)
	}
}
