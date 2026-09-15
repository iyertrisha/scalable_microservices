-- Order Service schema (database: orders)
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
