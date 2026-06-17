package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	sharedevents "github.com/ridego/pkg/events"
	"github.com/ridego/services/payment/internal/models"
	"github.com/ridego/services/payment/internal/repository"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/paymentintent"
)

type PaymentService struct {
	repo repository.Repository
	nc   *nats.Conn
}

func New(repo repository.Repository, nc *nats.Conn, stripeKey string) *PaymentService {
	stripe.Key = stripeKey
	return &PaymentService{repo: repo, nc: nc}
}

// ListenForCompletedTrips subscribes to ridego.trip.completed.
// When a trip finishes, we capture the pre-authorised payment and
// record a ledger entry — all idempotently via idempotency keys.
func (s *PaymentService) ListenForCompletedTrips(ctx context.Context) {
	sub, err := s.nc.Subscribe(sharedevents.SubjectTripCompleted, func(msg *nats.Msg) {
		var env sharedevents.Envelope[sharedevents.TripCompletedPayload]
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			slog.Warn("bad trip.completed payload", "err", err)
			return
		}
		if err := s.capturePayment(context.Background(), env.Payload); err != nil {
			slog.Error("capture payment", "trip", env.Payload.TripID, "err", err)
		}
	})
	if err != nil {
		slog.Error("nats subscribe", "err", err)
		return
	}
	<-ctx.Done()
	sub.Unsubscribe()
}

func (s *PaymentService) capturePayment(ctx context.Context, pl sharedevents.TripCompletedPayload) error {
	// Idempotency key — deterministic hash of trip ID so replays are safe
	ikey := fmt.Sprintf("%x", sha256.Sum256([]byte("capture:"+pl.TripID.String())))

	// Check upfront — avoid an unnecessary Stripe call on replay
	exists, err := s.repo.ExistsByIdempotencyKey(ctx, ikey)
	if err != nil {
		return fmt.Errorf("check idempotency: %w", err)
	}
	if exists {
		slog.Info("payment already captured, skipping", "trip", pl.TripID)
		return nil
	}

	// Retrieve the payment method for the rider
	pm, err := s.repo.GetPaymentMethod(ctx, pl.RiderID)
	if err != nil {
		return fmt.Errorf("get payment method: %w", err)
	}

	amountNGN := int64(pl.FinalFare * 100) // Stripe uses minor units (kobo)

	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(amountNGN),
		Currency:           stripe.String("ngn"),
		PaymentMethod:      stripe.String(pm.StripePaymentMethodID),
		ConfirmationMethod: stripe.String("automatic"),
		Confirm:            stripe.Bool(true),
	}
	params.SetIdempotencyKey(ikey)

	intent, err := paymentintent.New(params)
	if err != nil {
		return fmt.Errorf("stripe charge: %w", err)
	}

	// Record ledger entry regardless of Stripe status so we can reconcile later
	return s.repo.CreateLedgerEntry(ctx, &models.LedgerEntry{
		ID:             uuid.New(),
		TripID:         pl.TripID,
		RiderID:        pl.RiderID,
		DriverID:       pl.DriverID,
		AmountNGN:      pl.FinalFare,
		StripeIntentID: intent.ID,
		Status:         string(intent.Status),
		IdempotencyKey: ikey,
	})
}

func (s *PaymentService) GetHistory(ctx context.Context, userID uuid.UUID) ([]*models.LedgerEntry, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *PaymentService) AddPaymentMethod(ctx context.Context, userID uuid.UUID, stripeMethodID string) error {
	return s.repo.SavePaymentMethod(ctx, userID, stripeMethodID)
}
