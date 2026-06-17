package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ridego/services/location/internal/hub"
	"github.com/ridego/services/location/internal/models"
	"github.com/ridego/services/location/internal/store"
)

type LocationService interface {
	UpdateLocation(ctx context.Context, inp models.UpdateLocationInput) error
	NearbyDrivers(ctx context.Context, req models.NearbyRequest) ([]models.NearbyDriver, error)
	GetDriverLocation(ctx context.Context, driverID string) (*models.DriverLocation, error)
	DriverOnline(ctx context.Context, driverID string) error
	DriverOffline(ctx context.Context, driverID string) error
}

type locationService struct {
	geo store.GeoStore
	hub *hub.Hub
}

func New(geo store.GeoStore, h *hub.Hub) LocationService {
	return &locationService{geo: geo, hub: h}
}

// UpdateLocation persists the position to Redis GEO and then broadcasts
// the update to all WebSocket clients subscribed to this driver.
func (s *locationService) UpdateLocation(ctx context.Context, inp models.UpdateLocationInput) error {
	if err := s.geo.UpdateLocation(ctx, inp); err != nil {
		return fmt.Errorf("geo update: %w", err)
	}

	// Broadcast to WebSocket subscribers (non-blocking — hub queues internally)
	msg := models.WSMessage{
		Type: "location_update",
		Payload: models.DriverLocation{
			DriverID: inp.DriverID, Lat: inp.Lat, Lng: inp.Lng,
			Heading: inp.Heading, SpeedKmh: inp.SpeedKmh,
		},
	}
	data, _ := json.Marshal(msg)
	s.hub.Broadcast(inp.DriverID, data)
	return nil
}

func (s *locationService) NearbyDrivers(ctx context.Context, req models.NearbyRequest) ([]models.NearbyDriver, error) {
	return s.geo.NearbyDrivers(ctx, req)
}

func (s *locationService) GetDriverLocation(ctx context.Context, driverID string) (*models.DriverLocation, error) {
	return s.geo.GetDriverLocation(ctx, driverID)
}

func (s *locationService) DriverOnline(ctx context.Context, driverID string) error {
	slog.Info("driver online", "driver_id", driverID)
	return s.geo.SetOnline(ctx, driverID)
}

func (s *locationService) DriverOffline(ctx context.Context, driverID string) error {
	slog.Info("driver offline", "driver_id", driverID)
	// Broadcast offline event to any watching rider apps
	msg := models.WSMessage{Type: "driver_offline", Payload: map[string]string{"driver_id": driverID}}
	data, _ := json.Marshal(msg)
	s.hub.Broadcast(driverID, data)
	return s.geo.SetOffline(ctx, driverID)
}
