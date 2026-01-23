package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// username -> list of connections
	connections map[string][]*websocket.Conn
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[string][]*websocket.Conn),
	}
}

func (h *Hub) Subscribe(username string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.connections[username] = append(h.connections[username], conn)
}

func (h *Hub) Unsubscribe(username string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.connections[username]
	for i, c := range conns {
		if c == conn {
			h.connections[username] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
}

func (h *Hub) Notify(username string, messageID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	notification := map[string]string{
		"type": "new_message",
		"id":   messageID,
	}

	for _, conn := range h.connections[username] {
		conn.WriteJSON(notification)
	}
}
