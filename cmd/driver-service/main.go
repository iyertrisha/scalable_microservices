package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/iyertrisha/fleetflow/internal/driver"
	"github.com/iyertrisha/fleetflow/pkg/config"
	"github.com/iyertrisha/fleetflow/pkg/db"
)

func main() {
	addr := config.Env("DRIVER_ADDR", ":8081")
	dsn := config.Env("DRIVER_DATABASE_URL", "postgres://fleetflow:fleetflow@localhost:5432/drivers?sslmode=disable")

	sqlDB, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer sqlDB.Close()

	store := driver.NewStore(sqlDB)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	mux := http.NewServeMux()
	driver.NewHandler(store).Register(mux)

	log.Printf("driver-service listening on %s (SAFE_ASSIGN=%v)", addr, config.Env("SAFE_ASSIGN", "true"))
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
