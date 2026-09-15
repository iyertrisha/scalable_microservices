package main

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/iyertrisha/fleetflow/internal/pricing"
	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/db"
	"github.com/iyertrisha/fleetflow/pkg/events"
	"github.com/iyertrisha/fleetflow/pkg/httpx"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
)

func main() {
	addr := config.Env("PRICING_ADDR", ":8085")
	dsn := config.Env("PRICING_DATABASE_URL", "postgres://fleetflow:fleetflow@localhost:5432/pricing?sslmode=disable")
	brokers := strings.Split(config.Env("KAFKA_BROKERS", "localhost:9092"), ",")
	demand := config.Env("DEMAND_LEVEL", "normal")

	sqlDB, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer sqlDB.Close()
	store := pricing.NewStore(sqlDB)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /quotes", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			OrderID     string  `json:"order_id"`
			DistanceKm  float64 `json:"distance_km"`
			PackageSize string  `json:"package_size"`
			Priority    string  `json:"priority"`
		}
		if err := httpx.Decode(r, &body); err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		q := pricing.NewQuote(body.OrderID, body.DistanceKm, body.PackageSize, body.Priority, demand)
		if err := store.Save(q); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		httpx.JSON(w, http.StatusCreated, q)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	go func() {
		_ = kafkapkg.EnsureTopic(brokers, events.TopicOrderCreated, 3)
		c := kafkapkg.NewConsumer(brokers, events.TopicOrderCreated, "pricing-service")
		defer c.Close()
		log.Printf("pricing consuming %s", events.TopicOrderCreated)
		_ = c.Run(context.Background(), func(ctx context.Context, _, value []byte) error {
			var ev events.OrderCreated
			if err := json.Unmarshal(value, &ev); err != nil {
				return err
			}
			dist := haversine(ev.PickupLat, ev.PickupLng, ev.DeliveryLat, ev.DeliveryLng)
			q := pricing.NewQuote(ev.OrderID, dist, ev.PackageSize, ev.Priority, demand)
			if err := store.Save(q); err != nil {
				return err
			}
			log.Printf("priced order %s => ₹%d (%s)", q.OrderID, q.AmountINR, q.DemandLevel)
			return nil
		})
	}()

	log.Printf("pricing-service listening on %s demand=%s", addr, demand)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
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
