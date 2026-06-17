package models

import (
	"time"

	"github.com/google/uuid"
)

// PaymentMethod stores a Stripe payment method ID for a user.
// We never store raw card numbers — Stripe holds those.
type PaymentMethod struct {
	ID                    uuid.UUID `json:"id"`
	UserID                uuid.UUID `json:"user_id"`
	StripePaymentMethodID string    `json:"stripe_payment_method_id"`
	IsDefault             bool      `json:"is_default"`
	CreatedAt             time.Time `json:"created_at"`
}

// LedgerEntry is an immutable record of every charge attempt.
// status mirrors Stripe PaymentIntent status:
//   - "succeeded"       — money collected
//   - "requires_action" — 3DS challenge needed
//   - "canceled"        — payment failed or refunded
type LedgerEntry struct {
	ID             uuid.UUID `json:"id"`
	TripID         uuid.UUID `json:"trip_id"`
	RiderID        uuid.UUID `json:"rider_id"`
	DriverID       uuid.UUID `json:"driver_id"`
	AmountNGN      float64   `json:"amount_ngn"`
	StripeIntentID string    `json:"stripe_intent_id"`
	Status         string    `json:"status"`
	IdempotencyKey string    `json:"-"` // never expose in API responses
	CreatedAt      time.Time `json:"created_at"`
}
