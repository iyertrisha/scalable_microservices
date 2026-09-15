package order

import "time"

// Status values for an order's lifecycle.
const (
	StatusCreated   = "CREATED"
	StatusAssigned  = "ASSIGNED"
	StatusPickedUp  = "PICKED_UP"
	StatusDelivered = "DELIVERED"
	StatusCancelled = "CANCELLED"
)

// Order is a package delivery request.
type Order struct {
	ID               string    `json:"order_id"`
	PickupLocation   Location  `json:"pickup_location"`
	DeliveryLocation Location  `json:"delivery_location"`
	PackageSize      string    `json:"package_size"`
	Priority         string    `json:"priority"`
	Status           string    `json:"status"`
	AssignedDriver   string    `json:"assigned_driver,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Location is a geographic point.
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// CreateRequest is the body for POST /orders.
type CreateRequest struct {
	PickupLocation   Location `json:"pickup_location"`
	DeliveryLocation Location `json:"delivery_location"`
	PackageSize      string   `json:"package_size"`
	Priority         string   `json:"priority"`
}
