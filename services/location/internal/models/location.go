package models

import "time"

type DriverLocation struct {
	DriverID  string    `json:"driver_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	Heading   float32   `json:"heading"` // degrees 0-360
	SpeedKmh  float32   `json:"speed_kmh"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NearbyDriver struct {
	DriverID   string  `json:"driver_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	DistanceKm float64 `json:"distance_km"`
	Rating     float32 `json:"rating"`
	IsOnline   bool    `json:"is_online"`
}

type UpdateLocationInput struct {
	DriverID string  `json:"driver_id"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Heading  float32 `json:"heading"`
	SpeedKmh float32 `json:"speed_kmh"`
}

type NearbyRequest struct {
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	RadiusKm float64 `json:"radius_km"`
	Limit    int     `json:"limit"`
}

// WSMessage is the JSON envelope sent over WebSocket to rider apps.
type WSMessage struct {
	Type    string `json:"type"` // "location_update" | "driver_offline"
	Payload any    `json:"payload"`
}
