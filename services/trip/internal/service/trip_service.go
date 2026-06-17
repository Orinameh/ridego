package service

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	sharedevents "github.com/ridego/pkg/events"
	"github.com/ridego/services/trip/internal/events"
	"github.com/ridego/services/trip/internal/models"
	"github.com/ridego/services/trip/internal/repository"
	"github.com/ridego/services/trip/internal/statemachine"
)

type TripService interface {
	RequestTrip(ctx context.Context, riderID uuid.UUID, inp models.RequestTripInput) (*models.Trip, error)
	GetTrip(ctx context.Context, id uuid.UUID) (*models.Trip, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, callerID uuid.UUID, inp models.UpdateStatusInput) (*models.Trip, error)
	AssignDriver(ctx context.Context, tripID, driverID uuid.UUID) (*models.Trip, error)
	ListByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Trip, error)
	RateTrip(ctx context.Context, id uuid.UUID, inp models.RateTripInput) error
}

type tripService struct {
	repo repository.Repository
	pub  *events.Publisher
}

func New(repo repository.Repository, pub *events.Publisher) TripService {
	return &tripService{repo: repo, pub: pub}
}

func (s *tripService) RequestTrip(ctx context.Context, riderID uuid.UUID,
	inp models.RequestTripInput) (*models.Trip, error) {

	trip := &models.Trip{
		RiderID:          riderID,
		PickupLocation:   inp.PickupLocation,
		DropoffLocation:  inp.DropoffLocation,
		PickupAddress:    inp.PickupAddress,
		DropoffAddress:   inp.DropoffAddress,
		EstimatedFareNGN: estimateFare(inp.PickupLocation, inp.DropoffLocation),
	}
	if err := s.repo.Create(ctx, trip); err != nil {
		return nil, fmt.Errorf("create trip: %w", err)
	}

	// Publish event — Matching Service subscribes to this.
	_ = s.pub.TripRequested(ctx, sharedevents.TripRequestedPayload{
		TripID:  trip.ID,
		RiderID: riderID,
		Pickup: sharedevents.Coordinates{
			Lat: inp.PickupLocation.Lat,
			Lng: inp.PickupLocation.Lng,
		},
		Dropoff: sharedevents.Coordinates{
			Lat: inp.DropoffLocation.Lat,
			Lng: inp.DropoffLocation.Lng,
		},
	})
	return trip, nil
}

func (s *tripService) UpdateStatus(ctx context.Context, id, callerID uuid.UUID,
	inp models.UpdateStatusInput) (*models.Trip, error) {

	current, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	// Validate caller ownership
	if inp.Status == models.StatusCancelled {
		if current.RiderID != callerID && (current.DriverID == nil || *current.DriverID != callerID) {
			return nil, fmt.Errorf("only trip participants may cancel")
		}
	} else if current.DriverID == nil || *current.DriverID != callerID {
		return nil, fmt.Errorf("only the assigned driver may update status")
	}

	// Guard with state machine before hitting the DB
	if err := statemachine.Transition(current.Status, inp.Status); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateStatus(ctx, id, inp, current.Version)
	if err != nil {
		return nil, err
	}

	// Publish the right event for each terminal/key transition.
	switch inp.Status {
	case models.StatusInProgress:
		_ = s.pub.TripStarted(ctx, id, *updated.DriverID)
	case models.StatusCompleted:
		fare := 0.0
		dist := 0.0
		if updated.FinalFareNGN != nil {
			fare = *updated.FinalFareNGN
		}
		if updated.DistanceKm != nil {
			dist = *updated.DistanceKm
		}
		_ = s.pub.TripCompleted(ctx, sharedevents.TripCompletedPayload{
			TripID: id, RiderID: updated.RiderID,
			DriverID:  *updated.DriverID,
			FinalFare: fare, DistanceKm: dist,
		})
	case models.StatusCancelled:
		cancelledBy := "rider"
		if current.DriverID != nil && *current.DriverID == callerID {
			cancelledBy = "driver"
		}
		_ = s.pub.TripCancelled(ctx, sharedevents.TripCancelledPayload{
			TripID: id, RiderID: updated.RiderID, CancelledBy: cancelledBy,
		})
	}
	return updated, nil
}

func (s *tripService) AssignDriver(ctx context.Context, tripID, driverID uuid.UUID) (*models.Trip, error) {
	current, err := s.repo.GetByID(ctx, tripID)
	if err != nil {
		return nil, err
	}

	if err := statemachine.Transition(current.Status, models.StatusDriverAssigned); err != nil {
		return nil, err
	}

	updated, err := s.repo.AssignDriver(ctx, tripID, driverID, current.Version)
	if err != nil {
		return nil, err
	}

	_ = s.pub.TripAssigned(ctx, sharedevents.TripAssignedPayload{
		TripID: tripID, RiderID: updated.RiderID, DriverID: driverID,
	})
	return updated, nil
}

func (s *tripService) GetTrip(ctx context.Context, id uuid.UUID) (*models.Trip, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *tripService) ListByRider(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*models.Trip, error) {
	return s.repo.ListByRider(ctx, riderID, limit, offset)
}

func (s *tripService) RateTrip(ctx context.Context, id uuid.UUID, inp models.RateTripInput) error {
	return s.repo.Rate(ctx, id, inp)
}

// estimateFare is a naive Haversine-based flat-rate calculator.
// Replace with a real pricing engine (surge, zone, category).
func estimateFare(pickup, dropoff models.Coordinates) float64 {
	const R = 6371
	lat1 := pickup.Lat * math.Pi / 180
	lat2 := dropoff.Lat * math.Pi / 180
	dLat := (dropoff.Lat - pickup.Lat) * math.Pi / 180
	dLng := (dropoff.Lng - pickup.Lng) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	distKm := R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	const (
		baseNGN  = 500
		perKmNGN = 150
	)
	return math.Round((baseNGN+distKm*perKmNGN)/50) * 50 // round to nearest ₦50
}
