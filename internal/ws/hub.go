// Package ws provides a WebSocket fan-out hub that broadcasts real-time
// events to all connected clients subscribed to a specific company.
package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Event is the envelope sent to all WebSocket clients.
type Event struct {
	Type      string      `json:"type"`
	CompanyID uuid.UUID   `json:"company_id"`
	Payload   interface{} `json:"payload"`
}

// EventType constants used throughout the system.
const (
	EventAgentStatus    = "agent_status"
	EventAgentLog       = "agent_log"
	EventIssueUpdate    = "issue_update"
	EventHeartbeat      = "heartbeat"
	EventNotification   = "notification"
	EventHirePending    = "hire_pending"
	EventChatMessage    = "chat_message"
	EventEscalation     = "escalation"
	EventRuntimeStatus  = "runtime_status"
)

// client is a single connected WebSocket subscriber.
type client struct {
	conn      *websocket.Conn
	companyID uuid.UUID
	send      chan []byte
}

// Hub manages all active WebSocket connections, bucketed by company.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*client]struct{}),
	}
}

// Register adds a client to the hub for the given company.
// Returns a done channel that is closed when the connection drops.
func (h *Hub) Register(conn *websocket.Conn, companyID uuid.UUID) <-chan struct{} {
	c := &client{
		conn:      conn,
		companyID: companyID,
		send:      make(chan []byte, 256),
	}

	h.mu.Lock()
	if h.clients[companyID] == nil {
		h.clients[companyID] = make(map[*client]struct{})
	}
	h.clients[companyID][c] = struct{}{}
	h.mu.Unlock()

	done := make(chan struct{})
	go c.writePump(done, func() {
		h.mu.Lock()
		delete(h.clients[companyID], c)
		h.mu.Unlock()
	})
	go c.readPump() // consume pings from client (keeps connection alive)

	return done
}

// Broadcast sends an event to all clients subscribed to event.CompanyID.
func (h *Hub) Broadcast(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("ws: marshal event: %v", err)
		return
	}

	h.mu.RLock()
	targets := h.clients[event.CompanyID]
	h.mu.RUnlock()

	for c := range targets {
		select {
		case c.send <- data:
		default:
			// Slow client — drop message rather than block
		}
	}
}

// BroadcastAll sends to all connected clients regardless of company.
func (h *Hub) BroadcastAll(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, clients := range h.clients {
		for c := range clients {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func (c *client) writePump(done chan<- struct{}, onClose func()) {
	defer func() {
		c.conn.Close()
		onClose()
		close(done)
	}()

	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *client) readPump() {
	defer c.conn.Close()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}
