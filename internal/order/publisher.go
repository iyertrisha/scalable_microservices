package order

import (
	"context"
	"encoding/json"
	"log"

	"github.com/iyertrisha/fleetflow/pkg/events"
)

// Publisher publishes domain events after successful persistence.
type Publisher interface {
	PublishOrderCreated(ctx context.Context, o *Order) error
}

// KafkaPublisher publishes order.created events.
type KafkaPublisher struct {
	publish func(ctx context.Context, key string, value []byte) error
}

func NewKafkaPublisher(publish func(ctx context.Context, key string, value []byte) error) *KafkaPublisher {
	return &KafkaPublisher{publish: publish}
}

func (p *KafkaPublisher) PublishOrderCreated(ctx context.Context, o *Order) error {
	ev := events.OrderCreated{
		OrderID:     o.ID,
		PickupLat:   o.PickupLocation.Lat,
		PickupLng:   o.PickupLocation.Lng,
		DeliveryLat: o.DeliveryLocation.Lat,
		DeliveryLng: o.DeliveryLocation.Lng,
		PackageSize: o.PackageSize,
		Priority:    o.Priority,
		CreatedAt:   o.CreatedAt,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.publish(ctx, o.ID, b)
}

// LoggingPublisher is used before Kafka is wired (or for tests).
type LoggingPublisher struct{}

func (LoggingPublisher) PublishOrderCreated(_ context.Context, o *Order) error {
	log.Printf("event order.created order_id=%s", o.ID)
	return nil
}

// NopPublisher discards events.
type NopPublisher struct{}

func (NopPublisher) PublishOrderCreated(context.Context, *Order) error { return nil }
