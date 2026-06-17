package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ridego/services/trip/internal/models"
)

var (
	ErrNotFound        = errors.New("trip not found")
	ErrVersionConflict = errors.New("version conflict, please retry")
)

type Repository interface {
	Create(ctx context.Context, t *models.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Trip, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, inp models.UpdateStatusInput, version int) (*models.Trip, error)
	AssignDriver(ctx context.Context, tripID, driverID uuid.UUID, version int) (*models.Trip, error)
	ListByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Trip, error)
	Rate(ctx context.Context, id uuid.UUID, inp models.RateTripInput) error
}

type PGRepo struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) Repository { return &PGRepo{pool: pool} }

func (r *PGRepo) Create(ctx context.Context, t *models.Trip) error {
	pickup, _ := json.Marshal(t.PickupLocation)
	dropoff, _ := json.Marshal(t.DropoffLocation)
	t.ID = uuid.New()
	t.Status = models.StatusRequested

	_, err := r.pool.Exec(ctx, `
        INSERT INTO trips
            (id, rider_id, status, pickup_location, dropoff_location,
             pickup_address, dropoff_address, estimated_fare_ngn, version)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,1)
    `, t.ID, t.RiderID, t.Status, pickup, dropoff,
		t.PickupAddress, t.DropoffAddress, t.EstimatedFareNGN)
	return err
}

// UpdateStatus uses optimistic locking via a version column.
// If the row was already updated concurrently, version won't match
// and we return ErrVersionConflict — callers should retry.
func (r *PGRepo) UpdateStatus(ctx context.Context, id uuid.UUID,
	inp models.UpdateStatusInput, version int) (*models.Trip, error) {

	row := r.pool.QueryRow(ctx, `
        UPDATE trips
        SET status       = $3,
            distance_km  = COALESCE($4, distance_km),
            final_fare_ngn = COALESCE($5, final_fare_ngn),
            started_at   = CASE WHEN $3='in_progress'  THEN NOW() ELSE started_at   END,
            completed_at = CASE WHEN $3='completed'    THEN NOW() ELSE completed_at END,
            updated_at   = NOW(),
            version      = version + 1
        WHERE id = $1 AND version = $2
        RETURNING id, rider_id, driver_id, status, pickup_location, dropoff_location,
                  pickup_address, dropoff_address, estimated_fare_ngn, final_fare_ngn,
                  distance_km, started_at, completed_at, created_at, updated_at, version
    `, id, version, inp.Status, inp.DistanceKm, inp.FinalFare)

	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionConflict
	}
	return t, err
}

func (r *PGRepo) AssignDriver(ctx context.Context, tripID, driverID uuid.UUID, version int) (*models.Trip, error) {
	row := r.pool.QueryRow(ctx, `
        UPDATE trips SET driver_id=$3, status='driver_assigned',
               updated_at=NOW(), version=version+1
        WHERE id=$1 AND version=$2
        RETURNING id, rider_id, driver_id, status, pickup_location, dropoff_location,
                  pickup_address, dropoff_address, estimated_fare_ngn, final_fare_ngn,
                  distance_km, started_at, completed_at, created_at, updated_at, version
    `, tripID, version, driverID)

	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrVersionConflict
	}
	return t, err
}

func (r *PGRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Trip, error) {
	row := r.pool.QueryRow(ctx, `
        SELECT id, rider_id, driver_id, status, pickup_location, dropoff_location,
               pickup_address, dropoff_address, estimated_fare_ngn, final_fare_ngn,
               distance_km, started_at, completed_at, created_at, updated_at, version
        FROM trips WHERE id=$1
    `, id)
	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

func (r *PGRepo) ListByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Trip, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, rider_id, driver_id, status, pickup_location, dropoff_location,
               pickup_address, dropoff_address, estimated_fare_ngn, final_fare_ngn,
               distance_km, started_at, completed_at, created_at, updated_at, version
        FROM trips WHERE rider_id=$1
        ORDER BY created_at DESC LIMIT $2 OFFSET $3
    `, riderID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trips []*models.Trip
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		trips = append(trips, t)
	}
	return trips, rows.Err()
}

func (r *PGRepo) Rate(ctx context.Context, id uuid.UUID, inp models.RateTripInput) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE trips
        SET driver_rating = COALESCE($2, driver_rating),
            rider_rating  = COALESCE($3, rider_rating),
            updated_at    = NOW()
        WHERE id=$1 AND status='completed'
    `, id, inp.DriverRating, inp.RiderRating)
	return err
}

type scanner interface{ Scan(...any) error }

func scanTrip(row scanner) (*models.Trip, error) {
	t := &models.Trip{}
	var pickupJSON, dropoffJSON []byte
	err := row.Scan(
		&t.ID, &t.RiderID, &t.DriverID, &t.Status,
		&pickupJSON, &dropoffJSON,
		&t.PickupAddress, &t.DropoffAddress,
		&t.EstimatedFareNGN, &t.FinalFareNGN, &t.DistanceKm,
		&t.StartedAt, &t.CompletedAt,
		&t.CreatedAt, &t.UpdatedAt, &t.Version,
	)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(pickupJSON, &t.PickupLocation)
	json.Unmarshal(dropoffJSON, &t.DropoffLocation)
	return t, nil
}
