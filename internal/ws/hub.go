package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// inbox_id -> list of connections
	connections map[string][]*websocket.Conn
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[string][]*websocket.Conn),
	}
}

func (h *Hub) Subscribe(inboxID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[inboxID] = append(h.connections[inboxID], conn)
}

func (h *Hub) Unsubscribe(inboxID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.connections[inboxID]
	for i, c := range conns {
		if c == conn {
			h.connections[inboxID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.connections[inboxID]) == 0 {
		delete(h.connections, inboxID)
	}
}

func (h *Hub) Notify(inboxID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	notification := map[string]string{
		"type": "messages_available",
	}

	for _, conn := range h.connections[inboxID] {
		_ = conn.WriteJSON(notification)
	}
}
