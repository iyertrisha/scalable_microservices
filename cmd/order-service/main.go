package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iyertrisha/fleetflow/internal/order"
	"github.com/iyertrisha/fleetflow/pkg/auth"
	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/db"
	"github.com/iyertrisha/fleetflow/pkg/events"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
	"github.com/iyertrisha/fleetflow/pkg/otelx"
)

func main() {
	ctx := context.Background()
	shutdown, err := otelx.InitTracer(ctx, "order-service")
	if err != nil {
		log.Printf("otel init: %v", err)
	} else {
		defer func() { _ = shutdown(context.Background()) }()
	}

	addr := config.Env("ORDER_ADDR", ":8080")
	dsn := config.Env("ORDER_DATABASE_URL", "")
	brokers := strings.Split(config.Env("KAFKA_BROKERS", ""), ",")

	var store order.Store
	var publisher order.Publisher = order.NopPublisher{}

	if dsn != "" {
		sqlDB, err := db.Open(dsn)
		if err != nil {
			log.Fatalf("postgres: %v", err)
		}
		defer sqlDB.Close()
		pg := order.NewPostgresStore(sqlDB)
		mctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		if err := pg.Migrate(mctx); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		store = pg
		log.Printf("order-service using PostgreSQL")
	} else {
		store = order.NewMemoryStore()
		log.Printf("order-service using in-memory store (set ORDER_DATABASE_URL for Postgres)")
	}

	if len(brokers) > 0 && brokers[0] != "" {
		_ = kafkapkg.EnsureTopic(brokers, events.TopicOrderCreated, 3)
		_ = kafkapkg.EnsureTopic(brokers, events.TopicOrderAssigned, 3)
		prod := kafkapkg.NewProducer(brokers, events.TopicOrderCreated)
		defer prod.Close()
		publisher = order.NewKafkaPublisher(prod.Publish)
		log.Printf("order-service publishing to Kafka topic %s", events.TopicOrderCreated)
	} else {
		publisher = order.LoggingPublisher{}
	}

	mux := http.NewServeMux()
	order.NewHandler(store, publisher).Register(mux)

	log.Printf("order-service listening on %s", addr)
	if err := http.ListenAndServe(addr, auth.Middleware(mux)); err != nil {
		log.Fatal(err)
	}
}
