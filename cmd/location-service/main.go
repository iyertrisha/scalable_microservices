package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/events"
	"github.com/iyertrisha/fleetflow/pkg/httpx"
	kafkapkg "github.com/iyertrisha/fleetflow/pkg/kafka"
	"github.com/iyertrisha/fleetflow/pkg/lock"
)

func main() {
	addr := config.Env("LOCATION_ADDR", ":8084")
	brokers := strings.Split(config.Env("KAFKA_BROKERS", "localhost:9092"), ",")
	redisAddr := config.Env("REDIS_ADDR", "localhost:6379")

	_ = kafkapkg.EnsureTopic(brokers, events.TopicDriverLocationUpdated, 6)
	prod := kafkapkg.NewProducer(brokers, events.TopicDriverLocationUpdated)
	defer prod.Close()

	rdb := lock.NewRedis(redisAddr, "", 0)

	publishLocation := func(w http.ResponseWriter, r *http.Request, id string) {
		var body struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}
		if err := httpx.Decode(r, &body); err != nil {
			httpx.Error(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		ev := events.DriverLocationUpdated{
			DriverID:  id,
			Lat:       body.Lat,
			Lng:       body.Lng,
			UpdatedAt: time.Now().UTC(),
		}
		b, _ := json.Marshal(ev)
		if err := prod.Publish(r.Context(), id, b); err != nil {
			httpx.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = rdb.HSet(r.Context(), lock.LocationKey(id), map[string]any{
			"lat": body.Lat,
			"lng": body.Lng,
			"ts":  ev.UpdatedAt.Format(time.RFC3339),
		}).Err()
		httpx.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /drivers/{id}/location", func(w http.ResponseWriter, r *http.Request) {
		publishLocation(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("POST /location/drivers/{id}", func(w http.ResponseWriter, r *http.Request) {
		publishLocation(w, r, r.PathValue("id"))
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis ping: %v", err)
	}

	log.Printf("location-service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
