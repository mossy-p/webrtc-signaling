package models

import "time"

// RoomMetadata stores information about a room
type RoomMetadata struct {
	ID         string    `json:"id"`
	Code       string    `json:"code"`       // Short, shareable room code (e.g., "ABCD123")
	CreatorID  string    `json:"creatorId"`  // User ID from JWT who created the room
	CreatedAt  time.Time `json:"createdAt"`
	MaxPlayers int       `json:"maxPlayers"`
	PlayerCount int      `json:"playerCount"`
}

// CreateRoomRequest is the request body for creating a room
type CreateRoomRequest struct {
	MaxPlayers int `json:"maxPlayers" binding:"min=2,max=16"` // Default validation
}

// CreateRoomResponse is the response for creating a room
type CreateRoomResponse struct {
	RoomID string `json:"roomId"`
	Code   string `json:"code"`
}

// JoinRoomRequest contains optional data when joining a room
type JoinRoomRequest struct {
	DisplayName string `json:"displayName,omitempty"`
}
