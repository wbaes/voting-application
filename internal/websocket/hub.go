package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

// Hub maintains the set of active WebSocket clients and broadcasts messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

// HandleConnection upgrades an HTTP connection to WebSocket and registers it.
func (h *Hub) HandleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	// Read loop — keeps connection alive, detects close
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			_ = conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}()
}

// VoteUpdate is the JSON payload broadcast to all clients.
type VoteUpdate struct {
	PhotoID string `json:"photo_id"`
	Counts  []PhotoCount `json:"counts"`
	Total   int64  `json:"total"`
}

// PhotoCount represents the vote count for a single photo.
type PhotoCount struct {
	PhotoID   string `json:"photo_id"`
	VoteCount int64  `json:"vote_count"`
}

// Broadcast sends a message to all connected WebSocket clients.
func (h *Hub) Broadcast(update VoteUpdate) {
	data, err := json.Marshal(update)
	if err != nil {
		log.Printf("websocket marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("websocket write error: %v", err)
			_ = conn.Close()
			go func(c *websocket.Conn) {
				h.mu.Lock()
				delete(h.clients, c)
				h.mu.Unlock()
			}(conn)
		}
	}
}

// ClientCount returns the number of connected WebSocket clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
