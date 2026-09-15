package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/events"
	"github.com/iyertrisha/fleetflow/pkg/httpx"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
)

// notification-service mocks SMS/email/push providers.
func main() {
	addr := config.Env("NOTIFICATION_ADDR", ":8086")
	brokers := strings.Split(config.Env("KAFKA_BROKERS", "localhost:9092"), ",")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	consume := func(topic, group string) {
		_ = kafkapkg.EnsureTopic(brokers, topic, 3)
		c := kafkapkg.NewConsumer(brokers, topic, group)
		defer c.Close()
		log.Printf("notification consuming %s", topic)
		_ = c.Run(context.Background(), func(_ context.Context, key, value []byte) error {
			var payload map[string]any
			_ = json.Unmarshal(value, &payload)
			log.Printf("[MOCK SMS/EMAIL/PUSH] topic=%s key=%s payload=%v", topic, string(key), payload)
			return nil
		})
	}

	go consume(events.TopicOrderAssigned, "notification-assigned")
	go consume(events.TopicOrderPickedUp, "notification-picked-up")
	go consume(events.TopicOrderDelivered, "notification-delivered")

	log.Printf("notification-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
