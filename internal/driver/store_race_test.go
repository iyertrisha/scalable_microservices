package driver_test

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iyertrisha/fleetflow/internal/driver"
	"github.com/iyertrisha/fleetflow/pkg/db"
)

// Integration test: two concurrent assigns must produce exactly one winner when SAFE.
// Run with: ORDER_DATABASE_URL unused; needs DRIVER_DATABASE_URL or defaults to local.
func TestConcurrentAssign_AtomicWinsOnce(t *testing.T) {
	dsn := os.Getenv("DRIVER_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fleetflow:fleetflow@localhost:5433/drivers?sslmode=disable"
	}
	sqlDB, err := db.Open(dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer sqlDB.Close()

	store := driver.NewStore(sqlDB)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	d, err := store.Create(driver.CreateRequest{Name: "Race Driver", VehicleType: "bike"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateStatus(d.ID, driver.StatusAvailable); err != nil {
		t.Fatal(err)
	}

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.TryAssign(d.ID); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()

	if wins.Load() != 1 {
		t.Fatalf("expected exactly 1 winner with atomic assign, got %d", wins.Load())
	}
}

func TestConcurrentAssign_NaiveCanDoubleBook(t *testing.T) {
	if os.Getenv("RUN_RACE_DEMO") != "1" {
		t.Skip("set RUN_RACE_DEMO=1 to demonstrate the naive race")
	}
	dsn := os.Getenv("DRIVER_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fleetflow:fleetflow@localhost:5433/drivers?sslmode=disable"
	}
	sqlDB, err := db.Open(dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer sqlDB.Close()

	store := driver.NewStore(sqlDB)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = store.Migrate(ctx)

	d, err := store.Create(driver.CreateRequest{Name: "Naive Race", VehicleType: "bike"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateStatus(d.ID, driver.StatusAvailable); err != nil {
		t.Fatal(err)
	}

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := store.NaiveAssign(d.ID); err == nil {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()

	if wins.Load() < 2 {
		t.Logf("naive assign wins=%d (race depends on timing; often 2)", wins.Load())
	}
	if wins.Load() != 2 {
		t.Fatalf("expected naive path to double-book (2 wins), got %d — re-run; sleep makes it likely", wins.Load())
	}
}
