package statemachine

import (
	"fmt"

	"github.com/ridego/services/trip/internal/models"
)

// allowed maps each TripStatus to the set of statuses it may
// transition into. Only listed transitions are legal.

var allowed = map[models.TripStatus]map[models.TripStatus]bool{
	models.StatusRequested: {
		models.StatusDriverAssigned: true,
		models.StatusCancelled:      true,
	},
	models.StatusDriverAssigned: {
		models.StatusDriverArrived: true,
		models.StatusCancelled:     true,
	},
	models.StatusDriverArrived: {
		models.StatusInProgress: true,
	},
	models.StatusInProgress: {
		models.StatusCompleted: true,
	},
	// terminal states — no outgoing transitions
	models.StatusCompleted: {},
	models.StatusCancelled: {},
}

// Transition validates that moving from -> to is allowed and
// returns an error describing the illegal transition if not.
func Transition(from, to models.TripStatus) error {
	targets, ok := allowed[from]
	if !ok {
		return fmt.Errorf("unknown state %q", from)
	}
	if !targets[to] {
		return fmt.Errorf("illegal transition %q → %q", from, to)
	}
	return nil
}

// IsTerminal reports whether s is a terminal (non-modifiable) state.
func IsTerminal(s models.TripStatus) bool {
	return s == models.StatusCompleted || s == models.StatusCancelled
}

// AllowedNext returns the list of valid next states from s.
func AllowedNext(s models.TripStatus) []models.TripStatus {
	targets := allowed[s]
	out := make([]models.TripStatus, 0, len(targets))
	for t := range targets {
		out = append(out, t)
	}
	return out
}
