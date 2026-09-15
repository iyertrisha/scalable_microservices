package order

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("order not found")
	ErrInvalidStatus = errors.New("order cannot be cancelled in its current status")
)

// Store persists orders. Step 1 uses an in-memory map.
type Store interface {
	Create(req CreateRequest) (*Order, error)
	Get(id string) (*Order, error)
	Cancel(id string) (*Order, error)
	AssignDriver(id, driverID string) (*Order, error)
}

// MemoryStore keeps orders in a concurrent map. Data is lost on restart.
type MemoryStore struct {
	mu     sync.RWMutex
	orders map[string]*Order
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{orders: make(map[string]*Order)}
}

func (s *MemoryStore) Create(req CreateRequest) (*Order, error) {
	now := time.Now().UTC()
	o := &Order{
		ID:               uuid.NewString(),
		PickupLocation:   req.PickupLocation,
		DeliveryLocation: req.DeliveryLocation,
		PackageSize:      req.PackageSize,
		Priority:         req.Priority,
		Status:           StatusCreated,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	s.mu.Lock()
	s.orders[o.ID] = o
	s.mu.Unlock()
	return clone(o), nil
}

func (s *MemoryStore) Get(id string) (*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clone(o), nil
}

func (s *MemoryStore) Cancel(id string) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrNotFound
	}
	if o.Status == StatusDelivered || o.Status == StatusCancelled {
		return nil, ErrInvalidStatus
	}
	o.Status = StatusCancelled
	o.UpdatedAt = time.Now().UTC()
	return clone(o), nil
}

func (s *MemoryStore) AssignDriver(id, driverID string) (*Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return nil, ErrNotFound
	}
	if o.Status != StatusCreated {
		return nil, ErrInvalidStatus
	}
	o.AssignedDriver = driverID
	o.Status = StatusAssigned
	o.UpdatedAt = time.Now().UTC()
	return clone(o), nil
}

func clone(o *Order) *Order {
	c := *o
	return &c
}
