package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/iyertrisha/fleetflow/internal/dispatch"
	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/events"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
	"github.com/iyertrisha/fleetflow/pkg/lock"
)

func main() {
	brokers := strings.Split(config.Env("KAFKA_BROKERS", "localhost:9092"), ",")
	driverURL := config.Env("DRIVER_SERVICE_URL", "http://localhost:8081")
	orderURL := config.Env("ORDER_SERVICE_URL", "http://localhost:8080")
	redisAddr := config.Env("REDIS_ADDR", "localhost:6379")
	useLock := os.Getenv("USE_REDIS_LOCK") != "false"
	group := config.Env("DISPATCH_GROUP", "dispatch-service")

	_ = kafkapkg.EnsureTopic(brokers, events.TopicOrderCreated, 3)
	_ = kafkapkg.EnsureTopic(brokers, events.TopicOrderAssigned, 3)

	svc := dispatch.New(driverURL, orderURL)
	svc.UseLock = useLock
	if useLock {
		rdb := lock.NewRedis(redisAddr, "", 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("redis unavailable, continuing without lock: %v", err)
			svc.UseLock = false
		} else {
			svc.Lock = lock.New(rdb, 5*time.Second)
			log.Printf("dispatch using Redis lock at %s", redisAddr)
		}
	}

	assigned := kafkapkg.NewProducer(brokers, events.TopicOrderAssigned)
	defer assigned.Close()
	svc.PublishAssigned = assigned.Publish

	consumer := kafkapkg.NewConsumer(brokers, events.TopicOrderCreated, group)
	defer consumer.Close()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		addr := config.Env("DISPATCH_ADDR", ":8082")
		log.Printf("dispatch-service health on %s", addr)
		_ = http.ListenAndServe(addr, mux)
	}()

	log.Printf("dispatch-service consuming %s (group=%s)", events.TopicOrderCreated, group)
	if err := consumer.Run(context.Background(), svc.HandleOrderCreated); err != nil {
		log.Fatal(err)
	}
}
