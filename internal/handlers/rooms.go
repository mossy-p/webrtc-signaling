package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mossy-p/webrtc-signaling/internal/models"
	"github.com/mossy-p/webrtc-signaling/internal/redis"
)

const (
	roomCodeLength = 6
	roomTTL        = 24 * time.Hour
	codeChars      = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Removed ambiguous chars
)

// CreateRoom creates a new room (requires authentication)
func CreateRoom(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default max players if not specified
	if req.MaxPlayers == 0 {
		req.MaxPlayers = 8
	}

	// Generate unique room ID and code
	roomID := uuid.New().String()
	roomCode := generateRoomCode()

	// Create room metadata
	room := models.RoomMetadata{
		ID:         roomID,
		Code:       roomCode,
		CreatorID:  userID.(string),
		CreatedAt:  time.Now(),
		MaxPlayers: req.MaxPlayers,
		PlayerCount: 0,
	}

	// Store in Redis
	redisClient := redis.GetClient()
	ctx := redis.GetContext()

	// Store room metadata by ID
	roomData, err := json.Marshal(room)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	if err := redisClient.Set(ctx, "room:"+roomID, roomData, roomTTL).Err(); err != nil {
		log.Printf("Failed to store room in Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	// Store code-to-ID mapping for easy lookup
	if err := redisClient.Set(ctx, "code:"+roomCode, roomID, roomTTL).Err(); err != nil {
		log.Printf("Failed to store room code in Redis: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create room"})
		return
	}

	log.Printf("Room created: %s (code: %s) by user %s", roomID, roomCode, userID)

	c.JSON(http.StatusCreated, models.CreateRoomResponse{
		RoomID: roomID,
		Code:   roomCode,
	})
}

// GetRoom gets room information by code or ID (public)
func GetRoom(c *gin.Context) {
	roomIdentifier := c.Param("roomId")

	redisClient := redis.GetClient()
	ctx := redis.GetContext()

	// Try to find room by code first, then by ID
	roomID := roomIdentifier

	// Check if it's a code (6 chars) vs UUID
	if len(roomIdentifier) == roomCodeLength {
		id, err := redisClient.Get(ctx, "code:"+roomIdentifier).Result()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
			return
		}
		roomID = id
	}

	// Get room metadata
	roomData, err := redisClient.Get(ctx, "room:"+roomID).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	var room models.RoomMetadata
	if err := json.Unmarshal([]byte(roomData), &room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse room data"})
		return
	}

	// Get current player count
	playerCount, _ := redisClient.SCard(ctx, "room:"+roomID+":peers").Result()
	room.PlayerCount = int(playerCount)

	c.JSON(http.StatusOK, room)
}

// DeleteRoom deletes a room (requires authentication and creator)
func DeleteRoom(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	roomID := c.Param("roomId")

	redisClient := redis.GetClient()
	ctx := redis.GetContext()

	// Get room metadata to verify creator
	roomData, err := redisClient.Get(ctx, "room:"+roomID).Result()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
		return
	}

	var room models.RoomMetadata
	if err := json.Unmarshal([]byte(roomData), &room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse room data"})
		return
	}

	// Verify user is the creator
	if room.CreatorID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the room creator can delete the room"})
		return
	}

	// Delete room data
	redisClient.Del(ctx, "room:"+roomID)
	redisClient.Del(ctx, "code:"+room.Code)
	redisClient.Del(ctx, "room:"+roomID+":peers")

	log.Printf("Room deleted: %s by user %s", roomID, userID)

	c.JSON(http.StatusOK, gin.H{"message": "Room deleted"})
}

// generateRoomCode generates a random room code
func generateRoomCode() string {
	code := make([]byte, roomCodeLength)
	for i := range code {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(codeChars))))
		code[i] = codeChars[n.Int64()]
	}
	return string(code)
}

// ValidateRoomExists checks if a room exists and is not full
func ValidateRoomExists(roomIdentifier string) (string, *models.RoomMetadata, error) {
	redisClient := redis.GetClient()
	ctx := redis.GetContext()

	// Try to find room by code first, then by ID
	roomID := roomIdentifier

	// Check if it's a code (6 chars) vs UUID
	if len(roomIdentifier) == roomCodeLength {
		id, err := redisClient.Get(ctx, "code:"+roomIdentifier).Result()
		if err != nil {
			return "", nil, fmt.Errorf("room not found")
		}
		roomID = id
	}

	// Get room metadata
	roomData, err := redisClient.Get(ctx, "room:"+roomID).Result()
	if err != nil {
		return "", nil, fmt.Errorf("room not found")
	}

	var room models.RoomMetadata
	if err := json.Unmarshal([]byte(roomData), &room); err != nil {
		return "", nil, fmt.Errorf("failed to parse room data")
	}

	// Check if room is full
	playerCount, _ := redisClient.SCard(ctx, "room:"+roomID+":peers").Result()
	if int(playerCount) >= room.MaxPlayers {
		return "", nil, fmt.Errorf("room is full")
	}

	return roomID, &room, nil
}
