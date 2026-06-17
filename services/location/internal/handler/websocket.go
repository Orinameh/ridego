package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ridego/services/location/internal/hub"
)

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	ReadBufferSize:   256,
	WriteBufferSize:  4096,
	// In production, validate Origin against your allowed domains list
	CheckOrigin: func(r *http.Request) bool { return true },
}

// DriverStream upgrades the HTTP connection to WebSocket and subscribes
// the client to position updates for a specific driver.
// Used by the rider app to display the driver pin moving on the map.
func (h *Handler) DriverStream(w http.ResponseWriter, r *http.Request) {
	// Extracts the {id} parameter using standard net/http (Go 1.22+)
	driverID := r.PathValue("id")
	if driverID == "" {
		http.Error(w, "Missing driver ID", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("ws upgrade failed", "err", err)
		return
	}

	client := &hub.Client{
		DriverID: driverID,
		Conn:     conn,
		Send:     make(chan []byte, 64),
	}

	h.hub.Subscribe(client)
	defer h.hub.Unsubscribe(client)

	// Write pump — drains the client.Send channel to the WebSocket
	go client.WritePump()

	// Read pump — keeps the connection alive, handles pong/close frames
	// We don't expect the rider app to send anything, just pong frames.
	client.ReadPump()
}
