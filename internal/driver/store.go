package driver

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("driver not found")
	ErrNotAvailable  = errors.New("driver not available")
	ErrInvalidStatus = errors.New("invalid status transition")
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS drivers (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL,
    vehicle_type  TEXT NOT NULL DEFAULT 'bike',
    capacity      INT NOT NULL DEFAULT 1,
    status        TEXT NOT NULL,
    current_lat   DOUBLE PRECISION,
    current_lng   DOUBLE PRECISION,
    version       BIGINT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_drivers_status ON drivers(status);
`)
	return err
}

func (s *Store) Create(req CreateRequest) (*Driver, error) {
	if req.Name == "" {
		return nil, errors.New("name required")
	}
	if req.VehicleType == "" {
		req.VehicleType = "bike"
	}
	if req.Capacity <= 0 {
		req.Capacity = 1
	}
	now := time.Now().UTC()
	d := &Driver{
		ID:          uuid.NewString(),
		Name:        req.Name,
		VehicleType: req.VehicleType,
		Capacity:    req.Capacity,
		Status:      StatusOffline,
		Version:     0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.db.Exec(`
INSERT INTO drivers (id, name, vehicle_type, capacity, status, version, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.ID, d.Name, d.VehicleType, d.Capacity, d.Status, d.Version, d.CreatedAt, d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (s *Store) Get(id string) (*Driver, error) {
	row := s.db.QueryRow(`
SELECT id, name, vehicle_type, capacity, status, current_lat, current_lng, version, created_at, updated_at
FROM drivers WHERE id=$1`, id)
	return scanDriver(row)
}

func (s *Store) UpdateStatus(id, status string) (*Driver, error) {
	switch status {
	case StatusOffline, StatusAvailable, StatusAssigned, StatusPickingUp, StatusDelivering:
	default:
		return nil, ErrInvalidStatus
	}
	res, err := s.db.Exec(`
UPDATE drivers SET status=$1, version=version+1, updated_at=$2 WHERE id=$3`,
		status, time.Now().UTC(), id,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return s.Get(id)
}

// ListAvailable returns drivers currently AVAILABLE.
func (s *Store) ListAvailable() ([]Driver, error) {
	rows, err := s.db.Query(`
SELECT id, name, vehicle_type, capacity, status, current_lat, current_lng, version, created_at, updated_at
FROM drivers WHERE status=$1 ORDER BY updated_at ASC`, StatusAvailable)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Driver
	for rows.Next() {
		d, err := scanDriver(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// TryAssign atomically moves AVAILABLE -> ASSIGNED.
// Returns ErrNotAvailable if another worker won the race.
func (s *Store) TryAssign(id string) (*Driver, error) {
	res, err := s.db.Exec(`
UPDATE drivers
SET status=$1, version=version+1, updated_at=$2
WHERE id=$3 AND status=$4`,
		StatusAssigned, time.Now().UTC(), id, StatusAvailable,
	)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		if _, err := s.Get(id); err != nil {
			return nil, err
		}
		return nil, ErrNotAvailable
	}
	return s.Get(id)
}

// NaiveAssign is the broken Step 5 path: check then update (race-prone).
func (s *Store) NaiveAssign(id string) (*Driver, error) {
	d, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if d.Status != StatusAvailable {
		return nil, ErrNotAvailable
	}
	// Artificial delay makes the race easy to reproduce in tests.
	time.Sleep(20 * time.Millisecond)
	_, err = s.db.Exec(`
UPDATE drivers SET status=$1, version=version+1, updated_at=$2 WHERE id=$3`,
		StatusAssigned, time.Now().UTC(), id,
	)
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *Store) UpdateLocation(id string, lat, lng float64) error {
	res, err := s.db.Exec(`
UPDATE drivers SET current_lat=$1, current_lng=$2, updated_at=$3 WHERE id=$4`,
		lat, lng, time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDriver(row scannable) (*Driver, error) {
	var d Driver
	var lat, lng sql.NullFloat64
	err := row.Scan(&d.ID, &d.Name, &d.VehicleType, &d.Capacity, &d.Status, &lat, &lng, &d.Version, &d.CreatedAt, &d.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lat.Valid {
		d.CurrentLat = &lat.Float64
	}
	if lng.Valid {
		d.CurrentLng = &lng.Float64
	}
	return &d, nil
}
