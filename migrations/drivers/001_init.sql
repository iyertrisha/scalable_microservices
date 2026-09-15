-- Driver Service schema (database: drivers)
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
