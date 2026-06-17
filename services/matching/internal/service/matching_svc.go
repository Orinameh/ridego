package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	tripevents "github.com/ridego/pkg/events"
	pb "github.com/ridego/proto/location"
)

const (
	searchRadiusKm   = 5.0
	acceptTimeoutSec = 15
	maxRematches     = 3
)

type MatchingService struct {
	loc  pb.LocationServiceClient
	trip *TripClient
	nc   *nats.Conn
	// pending tracks driverID → tripID for in-flight accept/reject
	pending map[string]uuid.UUID
}

func New(loc pb.LocationServiceClient, trip *TripClient, nc *nats.Conn) *MatchingService {
	return &MatchingService{loc: loc, trip: trip, nc: nc, pending: make(map[string]uuid.UUID)}
}

// ListenForTripRequests subscribes to ridego.trip.requested and kicks off
// the matching flow for every new trip asynchronously.
func (s *MatchingService) ListenForTripRequests(ctx context.Context) {
	sub, err := s.nc.Subscribe(tripevents.SubjectTripRequested, func(msg *nats.Msg) {
		var env tripevents.Envelope[tripevents.TripRequestedPayload]
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Warn("bad trip.requested payload", "err", err)
			return
		}
		go s.matchTrip(context.Background(), env.Payload, 0)
	})
	if err != nil {
		slog.Error("nats subscribe", "err", err)
		return
	}
	<-ctx.Done()
	sub.Unsubscribe()
}

func (s *MatchingService) matchTrip(ctx context.Context,
	pl tripevents.TripRequestedPayload, attempt int) {

	if attempt >= maxRematches {
		slog.Warn("no driver found after max attempts", "trip", pl.TripID)
		// TODO: publish ridego.trip.no_driver_found event → notify rider
		return
	}

	// 1. Ask Location Service for nearby online drivers
	nearby, err := s.loc.NearbyDrivers(ctx, &pb.NearbyRequest{
		Lat: pl.Pickup.Lat, Lng: pl.Pickup.Lng,
		RadiusKm: searchRadiusKm, Limit: 10,
	})
	if err != nil || len(nearby.Drivers) == 0 {
		slog.Info("no drivers nearby, retrying", "attempt", attempt)
		time.Sleep(5 * time.Second)
		s.matchTrip(ctx, pl, attempt+1)
		return
	}

	// 2. Rank by weighted score: 60% distance, 40% rating
	candidates := s.rankDrivers(nearby.Drivers)

	// 3. Try each candidate in order until one accepts
	for _, driver := range candidates {
		accepted, err := s.offerToDriver(ctx, pl.TripID, driver.DriverId)
		if err != nil {
			continue
		}
		if !accepted {
			continue
		}

		driverID, _ := uuid.Parse(driver.DriverId)
		if err := s.trip.AssignDriver(ctx, pl.TripID, driverID); err != nil {
			slog.Error("assign driver", "err", err)
		}
		return
	}

	// All candidates rejected — expand radius and retry
	s.matchTrip(ctx, pl, attempt+1)
}

type scoredDriver struct {
	*pb.DriverPosition
	score float64
}

func (s *MatchingService) rankDrivers(drivers []*pb.DriverPosition) []scoredDriver {
	scored := make([]scoredDriver, len(drivers))
	for i, d := range drivers {
		// Normalise distance: closer = higher score component
		distScore := 1.0 / (1.0 + d.DistanceKm)
		ratingScore := float64(d.Rating) / 5.0
		scored[i] = scoredDriver{d, 0.6*distScore + 0.4*ratingScore}
	}
	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	return scored
}

// offerToDriver sends a push notification to the driver and waits up to
// acceptTimeoutSec for them to accept via the /v1/match/accept endpoint.
func (s *MatchingService) offerToDriver(ctx context.Context, tripID uuid.UUID, driverID string) (bool, error) {
	// Store pending offer — Accept/Reject handlers read this
	s.pending[driverID] = tripID

	// Publish offer event → push notification service picks it up
	payload, _ := json.Marshal(map[string]any{
		"trip_id": tripID, "driver_id": driverID,
	})
	s.nc.Publish("ridego.match.offered", payload)

	responseCh := make(chan bool, 1)
	sub, _ := s.nc.Subscribe(
		fmt.Sprintf("ridego.match.response.%s", driverID),
		func(msg *nats.Msg) {
			responseCh <- string(msg.Data) == "accept"
		},
	)
	defer sub.Unsubscribe()
	defer delete(s.pending, driverID)

	select {
	case accepted := <-responseCh:
		return accepted, nil
	case <-time.After(acceptTimeoutSec * time.Second):
		slog.Info("driver timeout", "driver_id", driverID, "trip_id", tripID)
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

// DriverResponse is called by the handler when a driver accepts/rejects.
func (s *MatchingService) DriverResponse(driverID string, accepted bool) {
	resp := "reject"
	if accepted {
		resp = "accept"
	}
	s.nc.Publish(fmt.Sprintf("ridego.match.response.%s", driverID), []byte(resp))
}
