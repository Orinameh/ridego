package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridego/services/payment/internal/models"
)

var ErrNotFound = errors.New("not found")

// Repository defines every database operation the Payment Service needs.
type Repository interface {
	// Payment methods
	SavePaymentMethod(ctx context.Context, userID uuid.UUID, stripeMethodID string) error
	GetPaymentMethod(ctx context.Context, userID uuid.UUID) (*models.PaymentMethod, error)

	// Ledger
	CreateLedgerEntry(ctx context.Context, entry *models.LedgerEntry) error
	GetLedgerEntry(ctx context.Context, id uuid.UUID) (*models.LedgerEntry, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.LedgerEntry, error)

	// Idempotency — check if a charge was already processed
	ExistsByIdempotencyKey(ctx context.Context, key string) (bool, error)
}

type PostgresRepo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) Repository {
	return &PostgresRepo{pool: pool}
}

// ---- payment methods -----------------------------------------------
func (r *PostgresRepo) SavePaymentMethod(ctx context.Context, userID uuid.UUID, stripeMethodID string) error {
	// Mark all existing methods as non-default first
	_, err := r.pool.Exec(ctx,
		`UPDATE payment_methods SET is_default = false WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return err
	}

	// Insert the new method as default
	_, err = r.pool.Exec(ctx, `
		INSERT INTO payment_methods (user_id, stripe_payment_method_id, is_default)
		VALUES ($1, $2, true)
		ON CONFLICT (stripe_payment_method_id) DO NOTHING
	`, userID, stripeMethodID)
	return err
}

func (r *PostgresRepo) GetPaymentMethod(ctx context.Context, userID uuid.UUID) (*models.PaymentMethod, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, stripe_payment_method_id, is_default, created_at
		FROM payment_methods
		WHERE user_id = $1 AND is_default = true
		LIMIT 1
	`, userID)

	pm := &models.PaymentMethod{}
	err := row.Scan(&pm.ID, &pm.UserID, &pm.StripePaymentMethodID, &pm.IsDefault, &pm.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return pm, err
}

// ---- ledger --------------------------------------------------------
func (r *PostgresRepo) CreateLedgerEntry(ctx context.Context, e *models.LedgerEntry) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO ledger_entries
			(id, trip_id, rider_id, driver_id, amount_ngn, stripe_intent_id, status, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (idempotency_key) DO NOTHING
	`,
		e.ID, e.TripID, e.RiderID, e.DriverID,
		e.AmountNGN, e.StripeIntentID, e.Status, e.IdempotencyKey,
	)
	return err
}

func (r *PostgresRepo) GetLedgerEntry(ctx context.Context, id uuid.UUID) (*models.LedgerEntry, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, trip_id, rider_id, driver_id, amount_ngn, stripe_intent_id, status, idempotency_key, created_at
		FROM ledger_entries WHERE id = $1
	`, id)
	return scanEntry(row)
}

func (r *PostgresRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.LedgerEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, trip_id, rider_id, driver_id, amount_ngn, stripe_intent_id, status, idempotency_key, created_at
		FROM ledger_entries
		WHERE rider_id = $1 OR driver_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*models.LedgerEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *PostgresRepo) ExistsByIdempotencyKey(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM ledger_entries WHERE idempotency_key = $1)`,
		key,
	).Scan(&exists)
	return exists, err
}

// ---- helpers -------------------------------------------------------
type scanner interface{ Scan(...any) error }

func scanEntry(row scanner) (*models.LedgerEntry, error) {
	e := &models.LedgerEntry{}
	err := row.Scan(
		&e.ID, &e.TripID, &e.RiderID, &e.DriverID,
		&e.AmountNGN, &e.StripeIntentID, &e.Status,
		&e.IdempotencyKey, &e.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return e, err
}
