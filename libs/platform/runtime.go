package platform

import (
	"context"
	"database/sql"
	"errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"io/fs"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Runtime struct {
	DB      *pgxpool.Pool
	Redis   *redis.Client
	Writer  *kafka.Writer
	Name    string
	Brokers []string
	Metrics *prometheus.CounterVec
	Tracer  *sdktrace.TracerProvider
}

func Open(ctx context.Context, name string, migrations fs.FS) (*Runtime, error) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", name))
	dsn := Env("DATABASE_URL", "")
	db, e := sql.Open("pgx", dsn)
	if e != nil {
		return nil, e
	}
	driver, e := postgres.WithInstance(db, &postgres.Config{})
	if e != nil {
		return nil, e
	}
	source, e := iofs.New(migrations, "migrations")
	if e != nil {
		return nil, e
	}
	m, e := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if e != nil {
		return nil, e
	}
	e = m.Up()
	m.Close()
	if e != nil && !errors.Is(e, migrate.ErrNoChange) {
		return nil, e
	}
	pool, e := pgxpool.New(ctx, dsn)
	if e != nil {
		return nil, e
	}
	if e = pool.Ping(ctx); e != nil {
		return nil, e
	}
	brokers := strings.Split(Env("KAFKA_BROKERS", "kafka:9092"), ",")
	r := &Runtime{DB: pool, Redis: redis.NewClient(&redis.Options{Addr: Env("REDIS_ADDR", "redis:6379")}), Name: name, Brokers: brokers}
	r.Writer = &kafka.Writer{Addr: kafka.TCP(brokers...), Balancer: &kafka.Hash{}, RequiredAcks: kafka.RequireAll, WriteTimeout: 10 * time.Second}
	r.Metrics = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "wms_operations_total", Help: "Committed operations and worker outcomes"}, []string{"service", "operation", "outcome"})
	prometheus.MustRegister(r.Metrics)
	exp, e := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(Env("OTEL_ENDPOINT", "otel-collector:4318")), otlptracehttp.WithInsecure())
	if e != nil {
		return nil, e
	}
	r.Tracer = sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp), sdktrace.WithResource(resource.NewWithAttributes("", attribute.String("service.name", name))))
	otel.SetTracerProvider(r.Tracer)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return r, nil
}
func (r *Runtime) Close() {
	ctx, c := context.WithTimeout(context.Background(), 5*time.Second)
	defer c()
	r.Writer.Close()
	r.Redis.Close()
	r.DB.Close()
	r.Tracer.Shutdown(ctx)
}
func (r *Runtime) Count(op, outcome string) { r.Metrics.WithLabelValues(r.Name, op, outcome).Inc() }
