package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/mossy-p/webrtc-signaling/internal/models"
	"github.com/mossy-p/webrtc-signaling/internal/redis"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin checking is handled by middleware
		return true
	},
}

// Room manages peers in a WebRTC room
type Room struct {
	ID    string
	Peers map[string]*Client
	mu    sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	ID     string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
}

var rooms = make(map[string]*Room)
var roomsMu sync.RWMutex

// HandleSignaling handles WebSocket connections for WebRTC signaling
func HandleSignaling(c *gin.Context) {
	roomIdentifier := c.Param("roomId")
	if roomIdentifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "roomId is required"})
		return
	}

	// Optional: Get display name from query param
	displayName := c.Query("displayName")

	// Validate room exists and get actual room ID
	roomID, roomMetadata, err := ValidateRoomExists(roomIdentifier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Generate unique peer ID
	peerID := uuid.New().String()
	if displayName != "" {
		// Optionally store display name with peer ID
		log.Printf("Peer %s joining as '%s'", peerID, displayName)
	}

	// Create client
	client := &Client{
		ID:     peerID,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	// Get or create room
	room := getOrCreateRoom(roomID)
	room.addClient(client)

	// Store peer in Redis
	redisClient := redis.GetClient()
	ctx := redis.GetContext()
	redisClient.SAdd(ctx, "room:"+roomID+":peers", peerID)
	redisClient.Expire(ctx, "room:"+roomID+":peers", 24*time.Hour)

	log.Printf("Peer %s joined room %s (code: %s) - %d/%d players",
		peerID, roomID, roomMetadata.Code, roomMetadata.PlayerCount+1, roomMetadata.MaxPlayers)

	// Send join confirmation
	joinMsg := models.SignalMessage{
		Type:   models.SignalTypeJoin,
		From:   peerID,
		RoomID: roomID,
	}
	client.sendMessage(joinMsg)

	// Notify other peers in room
	room.broadcastMessage(models.SignalMessage{
		Type:   models.SignalTypeJoin,
		From:   peerID,
		RoomID: roomID,
	}, peerID)

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump(room)
}

func getOrCreateRoom(roomID string) *Room {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		room = &Room{
			ID:    roomID,
			Peers: make(map[string]*Client),
		}
		rooms[roomID] = room
		log.Printf("Created new room: %s", roomID)
	}
	return room
}

func (r *Room) addClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Peers[client.ID] = client
}

func (r *Room) removeClient(client *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Peers, client.ID)

	// Clean up room if empty
	if len(r.Peers) == 0 {
		roomsMu.Lock()
		delete(rooms, r.ID)
		roomsMu.Unlock()
		log.Printf("Removed empty room: %s", r.ID)
	}
}

func (r *Room) broadcastMessage(msg models.SignalMessage, excludePeerID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	for peerID, client := range r.Peers {
		if peerID != excludePeerID {
			select {
			case client.Send <- data:
			default:
				log.Printf("Failed to send message to peer %s, buffer full", peerID)
			}
		}
	}
}

func (r *Room) sendToClient(msg models.SignalMessage, targetPeerID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	client, exists := r.Peers[targetPeerID]
	if !exists {
		log.Printf("Target peer %s not found in room %s", targetPeerID, r.ID)
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		log.Printf("Failed to send message to peer %s, buffer full", targetPeerID)
	}
}

func (c *Client) readPump(room *Room) {
	defer func() {
		room.removeClient(c)
		c.Conn.Close()

		// Remove from Redis
		redisClient := redis.GetClient()
		ctx := redis.GetContext()
		redisClient.SRem(ctx, "room:"+c.RoomID+":peers", c.ID)

		// Notify other peers
		room.broadcastMessage(models.SignalMessage{
			Type:   models.SignalTypeLeave,
			From:   c.ID,
			RoomID: c.RoomID,
		}, c.ID)

		log.Printf("Peer %s left room %s", c.ID, c.RoomID)
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse message
		var msg models.SignalMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		// Set the sender
		msg.From = c.ID
		msg.RoomID = c.RoomID

		// Route message based on type
		switch msg.Type {
		case models.SignalTypeOffer, models.SignalTypeAnswer, models.SignalTypeCandidate:
			// Forward to specific peer if "to" is specified
			if msg.To != "" {
				room.sendToClient(msg, msg.To)
			} else {
				// Broadcast to all other peers
				room.broadcastMessage(msg, c.ID)
			}
		default:
			log.Printf("Unknown message type: %s", msg.Type)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) sendMessage(msg models.SignalMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	select {
	case c.Send <- data:
	default:
		log.Printf("Failed to send message to peer %s, buffer full", c.ID)
	}
}
