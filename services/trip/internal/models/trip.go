package models

import (
	"time"

	"github.com/google/uuid"
)

type TripStatus string

const (
	StatusRequested      TripStatus = "requested"
	StatusDriverAssigned TripStatus = "driver_assigned"
	StatusDriverArrived  TripStatus = "driver_arrived"
	StatusInProgress     TripStatus = "in_progress"
	StatusCompleted      TripStatus = "completed"
	StatusCancelled      TripStatus = "cancelled"
)

type Coordinates struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Trip struct {
	ID               uuid.UUID   `json:"id"`
	RiderID          uuid.UUID   `json:"rider_id"`
	DriverID         *uuid.UUID  `json:"driver_id,omitempty"`
	Status           TripStatus  `json:"status"`
	PickupLocation   Coordinates `json:"pickup_location"`
	DropoffLocation  Coordinates `json:"dropoff_location"`
	PickupAddress    string      `json:"pickup_address"`
	DropoffAddress   string      `json:"dropoff_address"`
	EstimatedFareNGN float64     `json:"estimated_fare_ngn"`
	FinalFareNGN     *float64    `json:"final_fare_ngn,omitempty"`
	DistanceKm       *float64    `json:"distance_km,omitempty"`
	RiderRating      *float64    `json:"rider_rating,omitempty"`
	DriverRating     *float64    `json:"driver_rating,omitempty"`
	StartedAt        *time.Time  `json:"started_at,omitempty"`
	CompletedAt      *time.Time  `json:"completed_at,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
	Version          int         `json:"-"` // optimistic locking
}

type RequestTripInput struct {
	PickupLocation  Coordinates `json:"pickup_location"`
	DropoffLocation Coordinates `json:"dropoff_location"`
	PickupAddress   string      `json:"pickup_address"`
	DropoffAddress  string      `json:"dropoff_address"`
}

type UpdateStatusInput struct {
	Status     TripStatus `json:"status"`
	DistanceKm *float64   `json:"distance_km,omitempty"`
	FinalFare  *float64   `json:"final_fare_ngn,omitempty"`
}

type RateTripInput struct {
	DriverRating *float64 `json:"driver_rating,omitempty"`
	RiderRating  *float64 `json:"rider_rating,omitempty"`
}
