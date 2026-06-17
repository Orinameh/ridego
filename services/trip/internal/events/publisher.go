package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	sharedevents "github.com/ridego/pkg/events"
)

type Publisher struct{ nc *nats.Conn }

func NewPublisher(nc *nats.Conn) *Publisher {
	return &Publisher{nc: nc}
}

// Publish wraps a payload in an Envelope and publishes to subject.
func Publish[T any](ctx context.Context, p *Publisher, subject string, payload T) error {
	env := sharedevents.Envelope[T]{
		EventID:     uuid.New(),
		OccurredAt:  time.Now().UTC(),
		ServiceName: "trip-service",
		Payload:     payload,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	if err := p.nc.Publish(subject, data); err != nil {
		return fmt.Errorf("nats publish %s: %w", subject, err)
	}
	// Flush ensures the message is written to the TCP buffer before returning.
	return p.nc.FlushWithContext(ctx)
}

// Typed helpers — one per event type for call-site clarity.

func (p *Publisher) TripRequested(ctx context.Context, pl sharedevents.TripRequestedPayload) error {
	return Publish(ctx, p, sharedevents.SubjectTripRequested, pl)
}

func (p *Publisher) TripAssigned(ctx context.Context, pl sharedevents.TripAssignedPayload) error {
	return Publish(ctx, p, sharedevents.SubjectTripAssigned, pl)
}

func (p *Publisher) TripStarted(ctx context.Context, tripID, driverID uuid.UUID) error {
	return Publish(ctx, p, sharedevents.SubjectTripStarted, map[string]string{
		"trip_id": tripID.String(), "driver_id": driverID.String(),
	})
}

func (p *Publisher) TripCompleted(ctx context.Context, pl sharedevents.TripCompletedPayload) error {
	return Publish(ctx, p, sharedevents.SubjectTripCompleted, pl)
}

func (p *Publisher) TripCancelled(ctx context.Context, pl sharedevents.TripCancelledPayload) error {
	return Publish(ctx, p, sharedevents.SubjectTripCancelled, pl)
}
