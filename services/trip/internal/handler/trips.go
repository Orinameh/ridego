package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/ridego/pkg/response"
	"github.com/ridego/services/trip/internal/models"
	"github.com/ridego/services/trip/internal/repository"
)

func (h *Handler) RequestTrip(w http.ResponseWriter, r *http.Request) {
	riderID, _ := uuid.Parse(r.Header.Get("X-User-ID"))

	var inp models.RequestTripInput
	if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	trip, err := h.svc.RequestTrip(r.Context(), riderID, inp)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "could not create trip")
		return
	}
	response.Success(w, http.StatusCreated, trip)
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	trip, err := h.svc.GetTrip(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "trip not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, http.StatusOK, trip)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid ID")
		return
	}
	callerID, _ := uuid.Parse(r.Header.Get("X-User-ID"))

	var inp models.UpdateStatusInput
	if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	trip, err := h.svc.UpdateStatus(r.Context(), id, callerID, inp)
	switch {
	case errors.Is(err, repository.ErrNotFound):
		response.Error(w, http.StatusNotFound, "trip not found")
	case errors.Is(err, repository.ErrVersionConflict):
		response.Error(w, http.StatusConflict, "concurrent update, please retry")
	case err != nil && isStateMachineError(err):
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
	case err != nil:
		response.Error(w, http.StatusInternalServerError, "internal error")
	default:
		response.Success(w, http.StatusOK, trip)
	}
}

func (h *Handler) AssignDriver(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	tripID, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid trip ID")
		return
	}

	var body struct {
		DriverID uuid.UUID `json:"driver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	trip, err := h.svc.AssignDriver(r.Context(), tripID, body.DriverID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, http.StatusOK, trip)
}

func (h *Handler) TripHistory(w http.ResponseWriter, r *http.Request) {
	riderID, _ := uuid.Parse(r.Header.Get("X-User-ID"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit == 0 || limit > 50 {
		limit = 20
	}

	trips, err := h.svc.ListByRider(r.Context(), riderID, limit, offset)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, http.StatusOK, map[string]any{"trips": trips, "count": len(trips)})
}

func (h *Handler) RateTrip(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid  ID")
		return
	}

	var inp models.RateTripInput
	if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if err := h.svc.RateTrip(r.Context(), id, inp); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, http.StatusOK, map[string]string{"status": "rated"})
}

func isStateMachineError(err error) bool {
	return strings.HasPrefix(err.Error(), "illegal transition") ||
		strings.HasPrefix(err.Error(), "only the assigned driver") ||
		strings.HasPrefix(err.Error(), "only trip participants")
}
