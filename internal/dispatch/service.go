package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/iyertrisha/fleetflow/pkg/events"
	"github.com/iyertrisha/fleetflow/pkg/lock"
)

type Driver struct {
	ID         string   `json:"driver_id"`
	Status     string   `json:"status"`
	Capacity   int      `json:"capacity"`
	CurrentLat *float64 `json:"current_lat,omitempty"`
	CurrentLng *float64 `json:"current_lng,omitempty"`
}

type Service struct {
	DriverBaseURL string
	OrderBaseURL  string
	HTTP          *http.Client
	Lock          *lock.RedisLock
	UseLock       bool
	PublishAssigned func(ctx context.Context, key string, value []byte) error
}

func New(driverURL, orderURL string) *Service {
	return &Service{
		DriverBaseURL: driverURL,
		OrderBaseURL:  orderURL,
		HTTP:          &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *Service) HandleOrderCreated(ctx context.Context, _, value []byte) error {
	var ev events.OrderCreated
	if err := json.Unmarshal(value, &ev); err != nil {
		return err
	}
	log.Printf("dispatch received order.created order_id=%s", ev.OrderID)

	drivers, err := s.listAvailable(ctx)
	if err != nil {
		return err
	}
	if len(drivers) == 0 {
		log.Printf("no available drivers for order %s", ev.OrderID)
		return nil
	}

	best := pickBest(drivers, ev.PickupLat, ev.PickupLng, ev.Priority)
	if best == nil {
		log.Printf("no suitable driver for order %s", ev.OrderID)
		return nil
	}

	token := uuid.NewString()
	lockKey := lock.DriverKey(best.ID)
	if s.UseLock && s.Lock != nil {
		ok, err := s.Lock.TryAcquire(ctx, lockKey, token)
		if err != nil {
			return err
		}
		if !ok {
			log.Printf("could not lock driver %s for order %s", best.ID, ev.OrderID)
			return fmt.Errorf("driver lock contended")
		}
		defer func() { _ = s.Lock.Release(ctx, lockKey, token) }()
	}

	if err := s.assignDriver(ctx, best.ID); err != nil {
		return err
	}
	if err := s.assignOrder(ctx, ev.OrderID, best.ID); err != nil {
		return err
	}

	if s.PublishAssigned != nil {
		payload, _ := json.Marshal(events.OrderAssigned{OrderID: ev.OrderID, DriverID: best.ID})
		if err := s.PublishAssigned(ctx, ev.OrderID, payload); err != nil {
			log.Printf("publish order.assigned failed: %v", err)
		}
	}

	log.Printf("assigned driver %s to order %s", best.ID, ev.OrderID)
	return nil
}

func (s *Service) listAvailable(ctx context.Context) ([]Driver, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.DriverBaseURL+"/drivers?status=AVAILABLE", nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list drivers: %s %s", resp.Status, b)
	}
	var drivers []Driver
	if err := json.NewDecoder(resp.Body).Decode(&drivers); err != nil {
		return nil, err
	}
	return drivers, nil
}

func (s *Service) assignDriver(ctx context.Context, driverID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.DriverBaseURL+"/drivers/"+driverID+"/assign", bytes.NewBufferString(`{}`))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("driver %s already taken", driverID)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("assign driver: %s %s", resp.Status, b)
	}
	return nil
}

func (s *Service) assignOrder(ctx context.Context, orderID, driverID string) error {
	body, _ := json.Marshal(map[string]string{"driver_id": driverID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.OrderBaseURL+"/orders/"+orderID+"/assign", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("assign order: %s %s", resp.Status, b)
	}
	return nil
}

func pickBest(drivers []Driver, pickupLat, pickupLng float64, priority string) *Driver {
	var best *Driver
	bestScore := math.Inf(1)
	for i := range drivers {
		d := &drivers[i]
		dist := 0.0
		if d.CurrentLat != nil && d.CurrentLng != nil {
			dist = haversine(pickupLat, pickupLng, *d.CurrentLat, *d.CurrentLng)
		}
		score := dist
		if priority == "high" {
			score *= 0.8
		}
		if score < bestScore {
			bestScore = score
			best = d
		}
	}
	return best
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
