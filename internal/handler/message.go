package handler

import (
	"net/http"

	"github.com/BlackWire-Project/bwrelay/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageHandler struct {
	queries *db.Queries
}

func NewMessageHandler(pool *pgxpool.Pool) *MessageHandler {
	return &MessageHandler{
		queries: db.New(pool),
	}
}

// POST /messages
func (h *MessageHandler) Create(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GET /messages/:id
func (h *MessageHandler) GetByID(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GET /messages
func (h *MessageHandler) List(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
