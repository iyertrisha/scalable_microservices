package main

import (
	"context"
	"log"
	"strings"

	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/events"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
)

// event-logger is the tiny Step 4 consumer that only logs messages.
func main() {
	brokers := strings.Split(config.Env("KAFKA_BROKERS", "localhost:9092"), ",")
	topic := config.Env("KAFKA_TOPIC", events.TopicOrderCreated)
	group := config.Env("LOGGER_GROUP", "event-logger")

	_ = kafkapkg.EnsureTopic(brokers, topic, 3)
	c := kafkapkg.NewConsumer(brokers, topic, group)
	defer c.Close()

	log.Printf("event-logger consuming topic=%s group=%s", topic, group)
	err := c.Run(context.Background(), func(_ context.Context, key, value []byte) error {
		log.Printf("received topic=%s key=%s value=%s", topic, string(key), string(value))
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
