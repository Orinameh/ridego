package events

import (
	"time"

	"github.com/google/uuid"
)

// Subject constants — every service imports these
// so subject names never get out of sync.
const (
	SubjectTripRequested = "ridego.trip.requested"
	SubjectTripAssigned  = "ridego.trip.driver_assigned"
	SubjectTripStarted   = "ridego.trip.started"
	SubjectTripCompleted = "ridego.trip.completed"
	SubjectTripCancelled = "ridego.trip.cancelled"
	SubjectMatchOffered  = "ridego.match.offered"
)

// Envelope wraps every event with routing metadata.
// Every service uses this same wrapper when publishing or consuming.
type Envelope[T any] struct {
	EventID     uuid.UUID `json:"event_id"`
	OccurredAt  time.Time `json:"occurred_at"`
	ServiceName string    `json:"service"`
	Payload     T         `json:"payload"`
}

// Coordinates mirrors the Trip Service's location type.
// Duplicated here intentionally — shared packages should not depend
// on a service's internal models. Keep this in sync with
// services/trip/internal/models if the shape changes.
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ---- payloads -------------------------------------------------------

type TripRequestedPayload struct {
	TripID  uuid.UUID   `json:"trip_id"`
	RiderID uuid.UUID   `json:"rider_id"`
	Pickup  Coordinates `json:"pickup"`
	Dropoff Coordinates `json:"dropoff"`
}

type TripAssignedPayload struct {
	TripID   uuid.UUID `json:"trip_id"`
	RiderID  uuid.UUID `json:"rider_id"`
	DriverID uuid.UUID `json:"driver_id"`
}

type TripCompletedPayload struct {
	TripID     uuid.UUID `json:"trip_id"`
	RiderID    uuid.UUID `json:"rider_id"`
	DriverID   uuid.UUID `json:"driver_id"`
	FinalFare  float64   `json:"final_fare_ngn"`
	DistanceKm float64   `json:"distance_km"`
}

type TripCancelledPayload struct {
	TripID      uuid.UUID `json:"trip_id"`
	RiderID     uuid.UUID `json:"rider_id"`
	CancelledBy string    `json:"cancelled_by"` // "rider" | "driver" | "system"
}
