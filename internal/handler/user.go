package handler

import (
	"errors"
	"net/http"

	"github.com/BlackWire-Project/bwrelay/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	queries *db.Queries
}

func NewUserHandler(pool *pgxpool.Pool) *UserHandler {
	return &UserHandler{
		queries: db.New(pool),
	}
}

// Request/Response types

type CreateUserRequest struct {
	Username              string   `json:"username" binding:"required"`
	IdentityKey           string   `json:"identity_key" binding:"required"`
	SignedPrekey          string   `json:"signed_prekey" binding:"required"`
	SignedPrekeySignature string   `json:"signed_prekey_signature" binding:"required"`
	OneTimePrekeys        []string `json:"one_time_prekeys"`
}

type CreateUserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	CreatedAt string `json:"created_at"`
}

type GetUserResponse struct {
	Username              string `json:"username"`
	IdentityKey           string `json:"identity_key"`
	SignedPrekey          string `json:"signed_prekey"`
	SignedPrekeySignature string `json:"signed_prekey_signature"`
}

type GetPrekeyResponse struct {
	PrekeyID      *string `json:"prekey_id"`
	OneTimePrekey *string `json:"one_time_prekey"`
}

type AddPrekeysRequest struct {
	OneTimePrekeys []string `json:"one_time_prekeys" binding:"required"`
}

type AddPrekeysResponse struct {
	Count int `json:"count"`
}

// POST /users
func (h *UserHandler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	exists, err := h.queries.UserExists(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	// Create user
	user, err := h.queries.CreateUser(c.Request.Context(), db.CreateUserParams{
		Username:              req.Username,
		IdentityKey:           req.IdentityKey,
		SignedPrekey:          req.SignedPrekey,
		SignedPrekeySignature: req.SignedPrekeySignature,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	// Add one-time prekeys if provided
	for _, prekey := range req.OneTimePrekeys {
		h.queries.CreatePrekey(c.Request.Context(), db.CreatePrekeyParams{
			Username: req.Username,
			Prekey:   prekey,
		})
	}

	c.JSON(http.StatusCreated, CreateUserResponse{
		ID:        uuidToString(user.ID.Bytes),
		Username:  user.Username,
		CreatedAt: user.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	})
}

// GET /users/:username
func (h *UserHandler) Get(c *gin.Context) {
	username := c.Param("username")

	user, err := h.queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	c.JSON(http.StatusOK, GetUserResponse{
		Username:              user.Username,
		IdentityKey:           user.IdentityKey,
		SignedPrekey:          user.SignedPrekey,
		SignedPrekeySignature: user.SignedPrekeySignature,
	})
}

// GET /users/:username/prekey
func (h *UserHandler) GetPrekey(c *gin.Context) {
	username := c.Param("username")

	// Check if user exists
	exists, err := h.queries.UserExists(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get oldest prekey (FIFO) - does NOT delete it
	prekey, err := h.queries.GetPrekey(c.Request.Context(), username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// No prekeys available
			c.JSON(http.StatusOK, GetPrekeyResponse{PrekeyID: nil, OneTimePrekey: nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	prekeyID := uuidToString(prekey.ID.Bytes)
	c.JSON(http.StatusOK, GetPrekeyResponse{PrekeyID: &prekeyID, OneTimePrekey: &prekey.Prekey})
}

// POST /users/:username/prekeys
func (h *UserHandler) AddPrekeys(c *gin.Context) {
	username := c.Param("username")

	var req AddPrekeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user exists
	exists, err := h.queries.UserExists(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Add prekeys
	for _, prekey := range req.OneTimePrekeys {
		h.queries.CreatePrekey(c.Request.Context(), db.CreatePrekeyParams{
			Username: username,
			Prekey:   prekey,
		})
	}

	c.JSON(http.StatusOK, AddPrekeysResponse{Count: len(req.OneTimePrekeys)})
}

