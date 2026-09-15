package tracking

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/iyertrisha/fleetflow/pkg/events"
	"github.com/iyertrisha/fleetflow/pkg/httpx"
	"github.com/iyertrisha/fleetflow/pkg/lock"
	"github.com/redis/go-redis/v9"
)

type Snapshot struct {
	Status     string  `json:"status"`
	DriverID   string  `json:"driver,omitempty"`
	Lat        float64 `json:"lat,omitempty"`
	Lng        float64 `json:"lng,omitempty"`
	ETAMinutes int     `json:"eta_minutes"`
	UpdatedAt  string  `json:"updated_at"`
}

type Service struct {
	mu      sync.RWMutex
	orders  map[string]*Snapshot // order_id -> tracking
	drivers map[string]string    // driver_id -> order_id
	redis   *redis.Client
}

func New(rdb *redis.Client) *Service {
	return &Service{
		orders:  make(map[string]*Snapshot),
		drivers: make(map[string]string),
		redis:   rdb,
	}
}

func (s *Service) HandleAssigned(ctx context.Context, _, value []byte) error {
	var ev events.OrderAssigned
	if err := json.Unmarshal(value, &ev); err != nil {
		return err
	}
	s.mu.Lock()
	s.orders[ev.OrderID] = &Snapshot{
		Status:     "OUT_FOR_DELIVERY",
		DriverID:   ev.DriverID,
		ETAMinutes: 20,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	s.drivers[ev.DriverID] = ev.OrderID
	s.mu.Unlock()
	return nil
}

func (s *Service) HandleLocation(ctx context.Context, _, value []byte) error {
	var ev events.DriverLocationUpdated
	if err := json.Unmarshal(value, &ev); err != nil {
		return err
	}
	if s.redis != nil {
		_ = s.redis.HSet(ctx, lock.LocationKey(ev.DriverID), map[string]any{
			"lat": ev.Lat,
			"lng": ev.Lng,
			"ts":  ev.UpdatedAt.Format(time.RFC3339),
		}).Err()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	orderID, ok := s.drivers[ev.DriverID]
	if !ok {
		return nil
	}
	snap := s.orders[orderID]
	if snap == nil {
		return nil
	}
	snap.Lat = ev.Lat
	snap.Lng = ev.Lng
	snap.ETAMinutes = estimateETA(ev.Lat, ev.Lng)
	snap.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}

func estimateETA(lat, lng float64) int {
	// Stub: pretend destination is Bangalore CBD; ETA ~ distance * 3 minutes/km, min 5.
	destLat, destLng := 12.9716, 77.5946
	km := haversine(lat, lng, destLat, destLng)
	eta := int(math.Max(5, math.Round(km*3)))
	return eta
}

func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}

func (s *Service) Get(orderID string) (*Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snap, ok := s.orders[orderID]
	if !ok {
		return nil, false
	}
	cp := *snap
	return &cp, true
}

func (s *Service) Register(mux *http.ServeMux) {
	getTracking := func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		snap, ok := s.Get(id)
		if !ok {
			httpx.Error(w, http.StatusNotFound, "tracking not found")
			return
		}
		httpx.JSON(w, http.StatusOK, snap)
	}
	mux.HandleFunc("GET /orders/{id}/tracking", getTracking)
	mux.HandleFunc("GET /tracking/orders/{id}", getTracking)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}
