package driver

import "time"

const (
	StatusOffline    = "OFFLINE"
	StatusAvailable  = "AVAILABLE"
	StatusAssigned   = "ASSIGNED"
	StatusPickingUp  = "PICKING_UP"
	StatusDelivering = "DELIVERING"
)

type Driver struct {
	ID          string    `json:"driver_id"`
	Name        string    `json:"name"`
	VehicleType string    `json:"vehicle_type"`
	Capacity    int       `json:"capacity"`
	Status      string    `json:"status"`
	CurrentLat  *float64  `json:"current_lat,omitempty"`
	CurrentLng  *float64  `json:"current_lng,omitempty"`
	Version     int64     `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name        string `json:"name"`
	VehicleType string `json:"vehicle_type"`
	Capacity    int    `json:"capacity"`
}

type StatusRequest struct {
	Status string `json:"status"`
}

type AssignRequest struct {
	OrderID string `json:"order_id"`
}
