package handler

import (
	"net/http"

	"github.com/BlackWire-Project/bwrelay/internal/db"
	"github.com/gin-gonic/gin"
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

// POST /users
func (h *UserHandler) Create(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GET /users/:username
func (h *UserHandler) Get(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// GET /users/:username/prekey
func (h *UserHandler) GetPrekey(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}

// POST /users/:username/prekeys
func (h *UserHandler) AddPrekeys(c *gin.Context) {
	// TODO: implement
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
