package models

// SignalType represents the type of WebRTC signaling message
type SignalType string

const (
	SignalTypeJoin      SignalType = "join"
	SignalTypeLeave     SignalType = "leave"
	SignalTypeOffer     SignalType = "offer"
	SignalTypeAnswer    SignalType = "answer"
	SignalTypeCandidate SignalType = "candidate"
	SignalTypeError     SignalType = "error"
)

// SignalMessage represents a WebRTC signaling message
type SignalMessage struct {
	Type     SignalType  `json:"type"`
	From     string      `json:"from,omitempty"`
	To       string      `json:"to,omitempty"`
	RoomID   string      `json:"roomId"`
	Payload  interface{} `json:"payload,omitempty"`
	Error    string      `json:"error,omitempty"`
}

// Peer represents a connected peer in a room
type Peer struct {
	ID     string
	RoomID string
}
