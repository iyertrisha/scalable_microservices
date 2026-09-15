-- Pricing Service schema (database: pricing)
CREATE TABLE IF NOT EXISTS quotes (
    id            UUID PRIMARY KEY,
    order_id      TEXT NOT NULL,
    amount_inr    INT NOT NULL,
    demand_level  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL
);
