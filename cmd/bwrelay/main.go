package main

import (
	"context"
	"log"
	"net/http"

	"github.com/BlackWire-Project/bwrelay/internal/config"
	"github.com/BlackWire-Project/bwrelay/internal/handler"
	"github.com/BlackWire-Project/bwrelay/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	godotenv.Load()

	// Load config
	cfg := config.Load()

	// Connect to database
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Unable to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Initialize WebSocket hub
	hub := ws.NewHub()

	// Initialize handlers
	userHandler := handler.NewUserHandler(pool)
	messageHandler := handler.NewMessageHandler(pool, hub)
	wsHandler := ws.NewWSHandler(hub)

	// Setup router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// User routes
	r.POST("/users", userHandler.Create)
	r.GET("/users/:username/bundle", userHandler.GetBundle)
	r.POST("/users/:username/prekeys", userHandler.AddPrekeys)

	// Message routes
	r.POST("/messages", messageHandler.Create)
	r.GET("/messages", messageHandler.List)

	// WebSocket
	r.GET("/ws", wsHandler.Handle)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	r.Run(":" + cfg.Port)
}
