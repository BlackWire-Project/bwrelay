package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/BlackWire-Project/bwrelay/internal/db"
	"github.com/BlackWire-Project/bwrelay/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxKindLength            = 32
	maxHeaderLength          = 65536
	maxCiphertextLength      = 262144
	maxClientMessageIDLength = 255
	defaultListLimit         = 100
	maxListLimit             = 500
	prekeyMessageKind        = "prekey_message"
	ratchetMessageKind       = "ratchet_message"
	timeFormat               = "2006-01-02T15:04:05Z"
)

type MessageHandler struct {
	pool    *pgxpool.Pool
	queries *db.Queries
	hub     *ws.Hub
}

func NewMessageHandler(pool *pgxpool.Pool, hub *ws.Hub) *MessageHandler {
	return &MessageHandler{
		pool:    pool,
		queries: db.New(pool),
		hub:     hub,
	}
}

type CreateMessageRequest struct {
	InboxID         string  `json:"inbox_id" binding:"required"`
	Kind            string  `json:"kind" binding:"required"`
	Header          string  `json:"header" binding:"required"`
	Ciphertext      string  `json:"ciphertext" binding:"required"`
	UsedPrekeyID    *string `json:"used_prekey_id"`
	ClientMessageID string  `json:"client_message_id" binding:"required"`
}

type CreateMessageResponse struct {
	ID        string `json:"id"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
}

type GetMessageResponse struct {
	ID              string  `json:"id"`
	InboxID         string  `json:"inbox_id"`
	Kind            string  `json:"kind"`
	Header          string  `json:"header"`
	Ciphertext      string  `json:"ciphertext"`
	UsedPrekeyID    *string `json:"used_prekey_id"`
	ClientMessageID string  `json:"client_message_id"`
	CreatedAt       string  `json:"created_at"`
	ExpiresAt       string  `json:"expires_at"`
}

type ListMessagesResponse struct {
	Items       []GetMessageResponse `json:"items"`
	NextAfterID *string              `json:"next_after_id"`
	HasMore     bool                 `json:"has_more"`
}

// POST /messages
func (h *MessageHandler) Create(c *gin.Context) {
	var req CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateCreateMessageRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	qtx := h.queries.WithTx(tx)

	if _, err := qtx.GetUserByInboxID(c.Request.Context(), req.InboxID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "inbox not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	usedPrekeyID := pgtype.UUID{}
	if req.UsedPrekeyID != nil {
		parsed, err := parseUUID(*req.UsedPrekeyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid used_prekey_id format"})
			return
		}
		row := tx.QueryRow(c.Request.Context(), `
DELETE FROM one_time_prekeys p
USING users u
WHERE p.id = $1
  AND p.user_id = u.id
  AND u.inbox_id = $2
RETURNING p.id
`, parsed, req.InboxID)
		if err := row.Scan(&usedPrekeyID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.JSON(http.StatusConflict, gin.H{"error": "prekey already consumed, invalid, or not owned by inbox"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
			return
		}
	}

	msg, err := qtx.CreateMessage(c.Request.Context(), db.CreateMessageParams{
		InboxID:         req.InboxID,
		Kind:            req.Kind,
		Header:          req.Header,
		Ciphertext:      req.Ciphertext,
		UsedPrekeyID:    usedPrekeyID,
		ClientMessageID: req.ClientMessageID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "duplicate client_message_id for inbox"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create message"})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	h.hub.Notify(req.InboxID, uuidToString(msg.ID.Bytes))

	c.JSON(http.StatusCreated, CreateMessageResponse{
		ID:        uuidToString(msg.ID.Bytes),
		CreatedAt: msg.CreatedAt.Time.Format(timeFormat),
		ExpiresAt: msg.ExpiresAt.Time.Format(timeFormat),
	})
}

// GET /messages?inbox_id=...&limit=...&after_id=...&created_after=...&created_before=...
func (h *MessageHandler) List(c *gin.Context) {
	inboxID := c.Query("inbox_id")
	if inboxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "inbox_id query parameter required"})
		return
	}
	if err := validateLength("inbox_id", inboxID, maxInboxIDLength); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit := int32(defaultListLimit)
	if rawLimit := c.Query("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return
		}
		if parsed > maxListLimit {
			parsed = maxListLimit
		}
		limit = int32(parsed)
	}

	createdAfter, err := parseOptionalRFC3339(c.Query("created_after"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "created_after must be a valid RFC3339 timestamp"})
		return
	}

	createdBefore, err := parseOptionalRFC3339(c.Query("created_before"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "created_before must be a valid RFC3339 timestamp"})
		return
	}

	if createdAfter != nil && createdBefore != nil && createdAfter.After(*createdBefore) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "created_after must be before or equal to created_before"})
		return
	}

	afterID := c.Query("after_id")

	filter := db.ListMessagesFilter{
		InboxID:       inboxID,
		Limit:         int(limit) + 1,
		AfterID:       afterID,
		CreatedAfter:  createdAfter,
		CreatedBefore: createdBefore,
	}

	messages, err := h.queries.ListMessages(c.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, db.ErrInvalidMessageCursor) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "after_id is invalid for inbox"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	hasMore := len(messages) > int(limit)
	if hasMore {
		messages = messages[:int(limit)]
	}

	response := make([]GetMessageResponse, len(messages))
	for i, msg := range messages {
		response[i] = toMessageResponse(msg)
	}

	var nextAfterID *string
	if hasMore && len(response) > 0 {
		id := response[len(response)-1].ID
		nextAfterID = &id
	}

	c.JSON(http.StatusOK, ListMessagesResponse{
		Items:       response,
		NextAfterID: nextAfterID,
		HasMore:     hasMore,
	})
}

// GET /messages/:id?inbox_id=...
func (h *MessageHandler) GetByID(c *gin.Context) {
	inboxID := c.Query("inbox_id")
	if inboxID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "inbox_id query parameter required"})
		return
	}
	if err := validateLength("inbox_id", inboxID, maxInboxIDLength); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messageID := c.Param("id")
	if messageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message id path parameter required"})
		return
	}

	message, err := h.queries.GetMessageByID(c.Request.Context(), inboxID, messageID)
	if err != nil {
		if errors.Is(err, db.ErrInvalidMessageCursor) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, toMessageResponse(message))
}

func validateCreateMessageRequest(req CreateMessageRequest) error {
	if err := validateLength("inbox_id", req.InboxID, maxInboxIDLength); err != nil {
		return err
	}
	if err := validateLength("kind", req.Kind, maxKindLength); err != nil {
		return err
	}
	if err := validateLength("header", req.Header, maxHeaderLength); err != nil {
		return err
	}
	if err := validateLength("ciphertext", req.Ciphertext, maxCiphertextLength); err != nil {
		return err
	}
	if err := validateLength("client_message_id", req.ClientMessageID, maxClientMessageIDLength); err != nil {
		return err
	}

	switch req.Kind {
	case prekeyMessageKind:
		if req.UsedPrekeyID == nil {
			return errors.New("used_prekey_id is required for prekey_message")
		}
	case ratchetMessageKind:
	default:
		return errors.New("kind must be prekey_message or ratchet_message")
	}

	return nil
}

func toMessageResponse(msg db.Message) GetMessageResponse {
	var usedPrekeyID *string
	if msg.UsedPrekeyID.Valid {
		id := uuidToString(msg.UsedPrekeyID.Bytes)
		usedPrekeyID = &id
	}

	return GetMessageResponse{
		ID:              uuidToString(msg.ID.Bytes),
		InboxID:         msg.InboxID,
		Kind:            msg.Kind,
		Header:          msg.Header,
		Ciphertext:      msg.Ciphertext,
		UsedPrekeyID:    usedPrekeyID,
		ClientMessageID: msg.ClientMessageID,
		CreatedAt:       msg.CreatedAt.Time.Format(timeFormat),
		ExpiresAt:       msg.ExpiresAt.Time.Format(timeFormat),
	}
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}

	utc := parsed.UTC().Truncate(time.Second)
	return &utc, nil
}
