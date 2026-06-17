-- +goose Up
-- +goose StatementBegin

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE trips (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    rider_id           UUID         NOT NULL,
    driver_id          UUID,
    status             TEXT         NOT NULL
                                    CHECK (status IN (
                                        'requested','driver_assigned','driver_arrived',
                                        'in_progress','completed','cancelled'
                                    )),
    pickup_location    JSONB        NOT NULL,
    dropoff_location   JSONB        NOT NULL,
    pickup_address     TEXT         NOT NULL DEFAULT '',
    dropoff_address    TEXT         NOT NULL DEFAULT '',
    estimated_fare_ngn NUMERIC(10,2) NOT NULL DEFAULT 0,
    final_fare_ngn     NUMERIC(10,2),
    distance_km        NUMERIC(6,2),
    rider_rating       NUMERIC(2,1) CHECK (rider_rating  BETWEEN 1 AND 5),
    driver_rating      NUMERIC(2,1) CHECK (driver_rating BETWEEN 1 AND 5),
    started_at         TIMESTAMPTZ,
    completed_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    -- Optimistic locking: UPDATE fails if version doesn't match
    version            INT          NOT NULL DEFAULT 1
);

-- Query patterns
CREATE INDEX idx_trips_rider_id   ON trips (rider_id, created_at DESC);
CREATE INDEX idx_trips_driver_id  ON trips (driver_id) WHERE driver_id IS NOT NULL;
CREATE INDEX idx_trips_status     ON trips (status)    WHERE status NOT IN ('completed','cancelled');
CREATE INDEX idx_trips_pickup_loc ON trips USING GIN (pickup_location);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS trips;

-- +goose StatementEnd
