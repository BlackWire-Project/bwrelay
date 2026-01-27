package handler

import (
	"errors"
	"net/http"

	"github.com/BlackWire-Project/bwrelay/internal/db"
	"github.com/BlackWire-Project/bwrelay/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MessageHandler struct {
	queries *db.Queries
	hub     *ws.Hub
}

func NewMessageHandler(pool *pgxpool.Pool, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{
		queries: db.New(pool),
		hub:     hub,
	}
}

// Request/Response types

type CreateMessageRequest struct {
	Recipient           string  `json:"recipient" binding:"required"`
	Payload             string  `json:"payload" binding:"required"`
	DHPublic            string  `json:"dh_public" binding:"required"`
	MessageNumber       *int32  `json:"message_number" binding:"required"`
	PreviousChainLength int32   `json:"previous_chain_length"`
	PrekeyID            *string `json:"prekey_id"`
}

type CreateMessageResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
}

type GetMessageResponse struct {
	ID                  string `json:"id"`
	Recipient           string `json:"recipient"`
	Payload             string `json:"payload"`
	DHPublic            string `json:"dh_public"`
	MessageNumber       int32  `json:"message_number"`
	PreviousChainLength int32  `json:"previous_chain_length"`
	CreatedAt           string `json:"created_at"`
}

// POST /messages
func (h *MessageHandler) Create(c *gin.Context) {
	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate message_number is provided
	if req.MessageNumber == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_number is required"})
		return
	}

	// First message (message_number == 0) requires prekey_id
	if *req.MessageNumber == 0 && req.PrekeyID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prekey_id is required for first message (message_number == 0)"})
		return
	}

	// Check if recipient exists
	exists, err := h.queries.UserExists(c.Request.Context(), req.Recipient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "recipient not found"})
		return
	}

	// If prekey_id is provided, verify it exists and consume (delete) it
	if req.PrekeyID != nil {
		prekeyUUID, err := parseUUID(*req.PrekeyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid prekey_id format"})
			return
		}
		_, err = h.queries.DeletePrekeyByID(c.Request.Context(), prekeyUUID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusConflict, gin.H{"error": "prekey already consumed or invalid"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	// Create message
	msg, err := h.queries.CreateMessage(c.Request.Context(), db.CreateMessageParams{
		Recipient:           req.Recipient,
		Payload:             req.Payload,
		DhPublic:            req.DHPublic,
		MessageNumber:       *req.MessageNumber,
		PreviousChainLength: req.PreviousChainLength,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message"})
		return
	}

	messageID := uuidToString(msg.ID.Bytes)

	// Notify via WebSocket
	h.hub.Notify(req.Recipient, messageID)

	c.JSON(http.StatusCreated, CreateMessageResponse{
		ID:        messageID,
		CreatedAt: msg.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	})
}

// GET /messages/:id
func (h *MessageHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")

	id, err := parseUUID(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}

	msg, err := h.queries.GetMessageByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, GetMessageResponse{
		ID:                  uuidToString(msg.ID.Bytes),
		Recipient:           msg.Recipient,
		Payload:             msg.Payload,
		DHPublic:            msg.DhPublic,
		MessageNumber:       msg.MessageNumber,
		PreviousChainLength: msg.PreviousChainLength,
		CreatedAt:           msg.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	})
}

// GET /messages?recipient=username
func (h *MessageHandler) List(c *gin.Context) {
	recipient := c.Query("recipient")
	if recipient == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipient query parameter required"})
		return
	}

	messages, err := h.queries.GetMessagesByRecipient(c.Request.Context(), recipient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	response := make([]GetMessageResponse, len(messages))
	for i, msg := range messages {
		response[i] = GetMessageResponse{
			ID:                  uuidToString(msg.ID.Bytes),
			Recipient:           msg.Recipient,
			Payload:             msg.Payload,
			DHPublic:            msg.DhPublic,
			MessageNumber:       msg.MessageNumber,
			PreviousChainLength: msg.PreviousChainLength,
			CreatedAt:           msg.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, response)
}

