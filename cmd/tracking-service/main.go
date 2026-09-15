package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iyertrisha/fleetflow/internal/tracking"
	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/events"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
	"github.com/iyertrisha/fleetflow/pkg/lock"
)

func main() {
	addr := config.Env("TRACKING_ADDR", ":8083")
	brokers := strings.Split(config.Env("KAFKA_BROKERS", "localhost:9092"), ",")
	redisAddr := config.Env("REDIS_ADDR", "localhost:6379")

	rdb := lock.NewRedis(redisAddr, "", 0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis ping failed (continuing): %v", err)
	}

	svc := tracking.New(rdb)
	mux := http.NewServeMux()
	svc.Register(mux)

	go func() {
		_ = kafkapkg.EnsureTopic(brokers, events.TopicOrderAssigned, 3)
		c := kafkapkg.NewConsumer(brokers, events.TopicOrderAssigned, "tracking-assigned")
		defer c.Close()
		log.Printf("tracking consuming %s", events.TopicOrderAssigned)
		if err := c.Run(context.Background(), svc.HandleAssigned); err != nil {
			log.Printf("assigned consumer stopped: %v", err)
		}
	}()

	go func() {
		_ = kafkapkg.EnsureTopic(brokers, events.TopicDriverLocationUpdated, 6)
		c := kafkapkg.NewConsumer(brokers, events.TopicDriverLocationUpdated, "tracking-location")
		defer c.Close()
		log.Printf("tracking consuming %s", events.TopicDriverLocationUpdated)
		if err := c.Run(context.Background(), svc.HandleLocation); err != nil {
			log.Printf("location consumer stopped: %v", err)
		}
	}()

	log.Printf("tracking-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
