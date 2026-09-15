package events

import "time"

const (
	TopicOrderCreated          = "order.created"
	TopicOrderAssigned         = "order.assigned"
	TopicOrderPickedUp         = "order.picked_up"
	TopicOrderDelivered        = "order.delivered"
	TopicDriverLocationUpdated = "driver.location.updated"
)

type OrderCreated struct {
	OrderID          string    `json:"order_id"`
	PickupLat        float64   `json:"pickup_lat"`
	PickupLng        float64   `json:"pickup_lng"`
	DeliveryLat      float64   `json:"delivery_lat"`
	DeliveryLng      float64   `json:"delivery_lng"`
	PackageSize      string    `json:"package_size"`
	Priority         string    `json:"priority"`
	CreatedAt        time.Time `json:"created_at"`
}

type OrderAssigned struct {
	OrderID  string `json:"order_id"`
	DriverID string `json:"driver_id"`
}

type DriverLocationUpdated struct {
	DriverID  string    `json:"driver_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	UpdatedAt time.Time `json:"updated_at"`
}
