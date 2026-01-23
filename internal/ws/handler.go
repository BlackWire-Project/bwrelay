package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (adjust for production)
	},
}

type WSHandler struct {
	hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

type WSMessage struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
}

// GET /ws
func (h *WSHandler) Handle(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var subscribedAs string

	for {
		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		switch msg.Type {
		case "subscribe":
			if subscribedAs != "" {
				h.hub.Unsubscribe(subscribedAs, conn)
			}
			subscribedAs = msg.Username
			h.hub.Subscribe(msg.Username, conn)

		case "unsubscribe":
			if subscribedAs != "" {
				h.hub.Unsubscribe(subscribedAs, conn)
				subscribedAs = ""
			}
		}
	}

	// Cleanup on disconnect
	if subscribedAs != "" {
		h.hub.Unsubscribe(subscribedAs, conn)
	}
}
