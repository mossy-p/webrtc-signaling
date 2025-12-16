package main

import (
	"log"

	"github.com/mossy-p/webrtc-signaling/config"
	"github.com/mossy-p/webrtc-signaling/internal/handlers"
	"github.com/mossy-p/webrtc-signaling/internal/middleware"
	"github.com/mossy-p/webrtc-signaling/internal/redis"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to Redis
	if err := redis.Connect(cfg.Redis); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

	log.Println("Redis connection established")

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Global CORS middleware (runs before routing)
	router.Use(handlers.OriginFilter(cfg.AllowedOrigins))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Room management API (authenticated)
	apiGroup := router.Group("/api")
	{
		// Login endpoint (public)
		apiGroup.POST("/auth/login", handlers.Login(cfg.JWTSecret))

		// Create room (requires JWT)
		apiGroup.POST("/rooms", middleware.JWTAuth(cfg.JWTSecret), handlers.CreateRoom)

		// Get room info (public)
		apiGroup.GET("/rooms/:roomId", handlers.GetRoom)

		// Delete room (requires JWT, creator only)
		apiGroup.DELETE("/rooms/:roomId", middleware.JWTAuth(cfg.JWTSecret), handlers.DeleteRoom)
	}

	// WebSocket signaling endpoint
	wsGroup := router.Group("/ws")
	{
		// WebSocket signaling - accepts room code or ID
		wsGroup.GET("/signal/:roomId", handlers.HandleSignaling)
	}

	// Start server
	log.Printf("Starting WebRTC signaling server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
