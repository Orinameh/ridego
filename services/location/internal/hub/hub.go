package hub

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10 // slightly less than pongWait
	maxMessageSize = 512
)

// Client represents one WebSocket connection watching a driver.
type Client struct {
	DriverID string
	Conn     *websocket.Conn
	Send     chan []byte
}

// Hub maintains the set of active WebSocket clients keyed by driverID.
// All mutations go through the channel-based event loop to avoid lock contention.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{} // driverID → set of clients
	subscribe   chan *Client
	unsubscribe chan *Client
	broadcast   chan broadcastMsg
}

type broadcastMsg struct {
	driverID string
	data     []byte
}

func New() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*Client]struct{}),
		subscribe:   make(chan *Client, 64),
		unsubscribe: make(chan *Client, 64),
		broadcast:   make(chan broadcastMsg, 256),
	}
}

// Run is the hub's event loop. Must be called in its own goroutine.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.subscribe:
			h.mu.Lock()
			if _, ok := h.subscribers[client.DriverID]; !ok {
				h.subscribers[client.DriverID] = make(map[*Client]struct{})
			}
			h.subscribers[client.DriverID][client] = struct{}{}
			h.mu.Unlock()
			slog.Debug("ws subscribe", "driver", client.DriverID)

		case client := <-h.unsubscribe:
			h.mu.Lock()
			if subs, ok := h.subscribers[client.DriverID]; ok {
				delete(subs, client)
				if len(subs) == 0 {
					delete(h.subscribers, client.DriverID)
				}
			}
			h.mu.Unlock()
			close(client.Send)

		case msg := <-h.broadcast:
			h.mu.RLock()
			subs := h.subscribers[msg.driverID]
			h.mu.RUnlock()
			for client := range subs {
				select {
				case client.Send <- msg.data:
				default:
					// Client's send buffer full — drop the message (non-blocking)
					slog.Warn("ws send buffer full, dropping", "driver", msg.driverID)
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) Subscribe(c *Client)   { h.subscribe <- c }
func (h *Hub) Unsubscribe(c *Client) { h.unsubscribe <- c }
func (h *Hub) Broadcast(driverID string, data []byte) {
	h.broadcast <- broadcastMsg{driverID: driverID, data: data}
}

// WritePump pumps messages from the client's Send channel to the WebSocket.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() { ticker.Stop(); c.Conn.Close() }()

	for {
		select {
		case msg, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, msg)

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump keeps the connection alive by reading pong frames.
// Blocking — call from the handler goroutine, not a new one.
func (c *Client) ReadPump() {
	defer c.Conn.Close()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("ws unexpected close", "err", err)
			}
			return
		}
	}
}
