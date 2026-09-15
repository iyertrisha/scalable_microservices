package pricing

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/google/uuid"
)

type Quote struct {
	ID          string    `json:"quote_id"`
	OrderID     string    `json:"order_id"`
	AmountINR   int       `json:"amount_inr"`
	DemandLevel string    `json:"demand_level"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS quotes (
    id            UUID PRIMARY KEY,
    order_id      TEXT NOT NULL,
    amount_inr    INT NOT NULL,
    demand_level  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL
);`)
	return err
}

func (s *Store) Save(q *Quote) error {
	_, err := s.db.Exec(`INSERT INTO quotes (id, order_id, amount_inr, demand_level, created_at) VALUES ($1,$2,$3,$4,$5)`,
		q.ID, q.OrderID, q.AmountINR, q.DemandLevel, q.CreatedAt)
	return err
}

// Calculate returns a dynamic price from distance, size, priority, and demand.
func Calculate(distanceKm float64, packageSize, priority, demand string) (amount int, level string) {
	base := 80.0
	amountF := base + distanceKm*12
	switch packageSize {
	case "L", "XL":
		amountF += 40
	case "S":
		amountF -= 10
	}
	if priority == "high" {
		amountF *= 1.25
	}
	level = demand
	switch demand {
	case "high":
		amountF *= 1.5
	case "very_high":
		amountF *= 2.0
	default:
		level = "normal"
	}
	return int(math.Round(amountF)), level
}

func NewQuote(orderID string, distanceKm float64, packageSize, priority, demand string) *Quote {
	amt, level := Calculate(distanceKm, packageSize, priority, demand)
	return &Quote{
		ID:          uuid.NewString(),
		OrderID:     orderID,
		AmountINR:   amt,
		DemandLevel: level,
		CreatedAt:   time.Now().UTC(),
	}
}
