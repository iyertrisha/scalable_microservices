package order

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// PostgresStore persists orders in PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS orders (
    id                UUID PRIMARY KEY,
    pickup_lat        DOUBLE PRECISION NOT NULL,
    pickup_lng        DOUBLE PRECISION NOT NULL,
    delivery_lat      DOUBLE PRECISION NOT NULL,
    delivery_lng      DOUBLE PRECISION NOT NULL,
    package_size      TEXT NOT NULL DEFAULT 'M',
    priority          TEXT NOT NULL DEFAULT 'normal',
    status            TEXT NOT NULL,
    assigned_driver   TEXT,
    created_at        TIMESTAMPTZ NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
`)
	return err
}

func (s *PostgresStore) Create(req CreateRequest) (*Order, error) {
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
	_, err := s.db.Exec(`
INSERT INTO orders (id, pickup_lat, pickup_lng, delivery_lat, delivery_lng, package_size, priority, status, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		o.ID, o.PickupLocation.Lat, o.PickupLocation.Lng,
		o.DeliveryLocation.Lat, o.DeliveryLocation.Lng,
		o.PackageSize, o.Priority, o.Status, o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return o, nil
}

func (s *PostgresStore) Get(id string) (*Order, error) {
	row := s.db.QueryRow(`
SELECT id, pickup_lat, pickup_lng, delivery_lat, delivery_lng, package_size, priority, status, COALESCE(assigned_driver,''), created_at, updated_at
FROM orders WHERE id=$1`, id)
	return scanOrder(row)
}

func (s *PostgresStore) Cancel(id string) (*Order, error) {
	res, err := s.db.Exec(`
UPDATE orders SET status=$1, updated_at=$2
WHERE id=$3 AND status NOT IN ($4,$5)`,
		StatusCancelled, time.Now().UTC(), id, StatusDelivered, StatusCancelled,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := s.Get(id); err != nil {
			return nil, err
		}
		return nil, ErrInvalidStatus
	}
	return s.Get(id)
}

func (s *PostgresStore) AssignDriver(id, driverID string) (*Order, error) {
	res, err := s.db.Exec(`
UPDATE orders SET status=$1, assigned_driver=$2, updated_at=$3
WHERE id=$4 AND status=$5`,
		StatusAssigned, driverID, time.Now().UTC(), id, StatusCreated,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := s.Get(id); err != nil {
			return nil, err
		}
		return nil, ErrInvalidStatus
	}
	return s.Get(id)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanOrder(row scannable) (*Order, error) {
	var o Order
	var driver string
	err := row.Scan(
		&o.ID, &o.PickupLocation.Lat, &o.PickupLocation.Lng,
		&o.DeliveryLocation.Lat, &o.DeliveryLocation.Lng,
		&o.PackageSize, &o.Priority, &o.Status, &driver, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	o.AssignedDriver = driver
	return &o, nil
}
