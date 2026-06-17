package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ridego/pkg/response"
	"github.com/ridego/services/location/internal/models"
)

func (h *Handler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	var inp models.UpdateLocationInput
	if err := json.NewDecoder(r.Body).Decode(&inp); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if inp.DriverID == "" {
		// Fallback: read driver ID from auth header injected by Gateway
		inp.DriverID = r.Header.Get("X-User-ID")
	}
	if err := h.svc.UpdateLocation(r.Context(), inp); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) NearbyDrivers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	lat, errLat := strconv.ParseFloat(q.Get("lat"), 64)
	lng, errLng := strconv.ParseFloat(q.Get("lng"), 64)
	if errLat != nil || errLng != nil {
		response.Error(w, http.StatusBadRequest, "lat and lng required")
		return
	}
	radius, _ := strconv.ParseFloat(q.Get("radius_km"), 64)
	if radius <= 0 {
		radius = 5
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	drivers, err := h.svc.NearbyDrivers(r.Context(), models.NearbyRequest{
		Lat: lat, Lng: lng, RadiusKm: radius, Limit: limit,
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"drivers": drivers, "count": len(drivers)})
}

func (h *Handler) GetDriverLocation(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	loc, err := h.svc.GetDriverLocation(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "driver not found")
		return
	}
	response.JSON(w, http.StatusOK, loc)
}

func (h *Handler) SetOnline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DriverOnline(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"online": true})
}

func (h *Handler) SetOffline(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.svc.DriverOffline(r.Context(), id); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]bool{"online": false})
}
