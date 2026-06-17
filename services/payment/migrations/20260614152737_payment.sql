-- +goose Up
-- +goose StatementBegin
CREATE TABLE payment_methods (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  UUID        NOT NULL,
    stripe_payment_method_id TEXT        NOT NULL,
    is_default               BOOLEAN     NOT NULL DEFAULT true,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_pm_user ON payment_methods (user_id);

CREATE TABLE ledger_entries (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    trip_id          UUID         NOT NULL UNIQUE,
    rider_id         UUID         NOT NULL,
    driver_id        UUID         NOT NULL,
    amount_ngn       NUMERIC(12,2) NOT NULL,
    stripe_intent_id TEXT         NOT NULL,
    status           TEXT         NOT NULL,
    idempotency_key  TEXT         NOT NULL UNIQUE,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ledger_rider  ON ledger_entries (rider_id,  created_at DESC);
CREATE INDEX idx_ledger_driver ON ledger_entries (driver_id, created_at DESC);
CREATE INDEX idx_ledger_idem   ON ledger_entries (idempotency_key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS ledger_entries;

-- +goose StatementEnd
