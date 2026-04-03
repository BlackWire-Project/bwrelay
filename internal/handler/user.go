package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/BlackWire-Project/bwrelay/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxUsernameLength = 255
	maxInboxIDLength  = 512
	maxKeyLength      = 16384
	maxPrekeysPerUser = 1000
)

type UserHandler struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewUserHandler(pool *pgxpool.Pool) *UserHandler {
	return &UserHandler{
		pool:    pool,
		queries: db.New(pool),
	}
}

type CreateUserRequest struct {
	Username              string   `json:"username" binding:"required"`
	IdentityKey           string   `json:"identity_key" binding:"required"`
	SignedPrekey          string   `json:"signed_prekey" binding:"required"`
	SignedPrekeySignature string   `json:"signed_prekey_signature" binding:"required"`
	InboxID               string   `json:"inbox_id" binding:"required"`
	OneTimePrekeys        []string `json:"one_time_prekeys"`
}

type CreateUserResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	InboxID   string `json:"inbox_id"`
	CreatedAt string `json:"created_at"`
}

type GetBundleResponse struct {
	Username              string  `json:"username"`
	IdentityKey           string  `json:"identity_key"`
	SignedPrekey          string  `json:"signed_prekey"`
	SignedPrekeySignature string  `json:"signed_prekey_signature"`
	InboxID               string  `json:"inbox_id"`
	PrekeyID              *string `json:"prekey_id"`
	OneTimePrekey         *string `json:"one_time_prekey"`
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
	if err := validateUserRequest(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exists, err := h.queries.UserExists(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "username already exists"})
		return
	}

	if err := validatePrekeys(req.OneTimePrekeys); err != nil {
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

	user, err := qtx.CreateUser(c.Request.Context(), db.CreateUserParams{
		Username:              req.Username,
		IdentityKey:           req.IdentityKey,
		SignedPrekey:          req.SignedPrekey,
		SignedPrekeySignature: req.SignedPrekeySignature,
		InboxID:               req.InboxID,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			field := "key"
			switch {
			case strings.Contains(pgErr.ConstraintName, "users_username"):
				field = "username"
			case strings.Contains(pgErr.ConstraintName, "identity_key"):
				field = "identity_key"
			case strings.Contains(pgErr.ConstraintName, "signed_prekey"):
				field = "signed_prekey"
			case strings.Contains(pgErr.ConstraintName, "inbox_id"):
				field = "inbox_id"
			}
			c.JSON(http.StatusConflict, gin.H{"error": field + " already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	added := 0
	for _, prekey := range req.OneTimePrekeys {
		_, err := qtx.CreatePrekey(c.Request.Context(), db.CreatePrekeyParams{
			UserID: user.ID,
			Prekey: prekey,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add prekeys"})
			return
		}
		added++
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusCreated, CreateUserResponse{
		ID:        uuidToString(user.ID.Bytes),
		Username:  user.Username,
		InboxID:   user.InboxID,
		CreatedAt: user.CreatedAt.Time.Format(timeFormat),
	})
}

// GET /users/:username/bundle
func (h *UserHandler) GetBundle(c *gin.Context) {
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

	var prekeyID *string
	var oneTimePrekey *string

	prekey, err := h.queries.GetPrekeyByUsername(c.Request.Context(), username)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	if err == nil {
		id := uuidToString(prekey.ID.Bytes)
		prekeyID = &id
		oneTimePrekey = &prekey.Prekey
	}

	c.JSON(http.StatusOK, GetBundleResponse{
		Username:              user.Username,
		IdentityKey:           user.IdentityKey,
		SignedPrekey:          user.SignedPrekey,
		SignedPrekeySignature: user.SignedPrekeySignature,
		InboxID:               user.InboxID,
		PrekeyID:              prekeyID,
		OneTimePrekey:         oneTimePrekey,
	})
}

// POST /users/:username/prekeys
func (h *UserHandler) AddPrekeys(c *gin.Context) {
	username := c.Param("username")

	var req AddPrekeysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.OneTimePrekeys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "one_time_prekeys cannot be empty"})
		return
	}
	if err := validatePrekeys(req.OneTimePrekeys); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.queries.GetUserByUsername(c.Request.Context(), username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	qtx := h.queries.WithTx(tx)

	added := 0
	for _, prekey := range req.OneTimePrekeys {
		_, err := qtx.CreatePrekey(c.Request.Context(), db.CreatePrekeyParams{
			UserID: user.ID,
			Prekey: prekey,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				continue
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add prekeys"})
			return
		}
		added++
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, AddPrekeysResponse{Count: added})
}

func validateUserRequest(req CreateUserRequest) error {
	if err := validateLength("username", req.Username, maxUsernameLength); err != nil {
		return err
	}
	if err := validateLength("inbox_id", req.InboxID, maxInboxIDLength); err != nil {
		return err
	}
	if err := validateLength("identity_key", req.IdentityKey, maxKeyLength); err != nil {
		return err
	}
	if err := validateLength("signed_prekey", req.SignedPrekey, maxKeyLength); err != nil {
		return err
	}
	if err := validateLength("signed_prekey_signature", req.SignedPrekeySignature, maxKeyLength); err != nil {
		return err
	}
	if len(req.OneTimePrekeys) > maxPrekeysPerUser {
		return errors.New("too many one_time_prekeys")
	}
	return nil
}

func validatePrekeys(prekeys []string) error {
	if len(prekeys) > maxPrekeysPerUser {
		return errors.New("too many one_time_prekeys")
	}
	for _, prekey := range prekeys {
		if err := validateLength("one_time_prekey", prekey, maxKeyLength); err != nil {
			return err
		}
	}
	return nil
}
