package ws

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const maxInboxIDLength = 255

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	hub *Hub
}

func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// GET /ws?inbox_id=...
func (h *WSHandler) Handle(c *gin.Context) {
	inboxID := c.Query("inbox_id")
	if inboxID == "" || len(inboxID) > maxInboxIDLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid inbox_id query parameter required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.hub.Subscribe(inboxID, conn)
	defer h.hub.Unsubscribe(inboxID, conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
