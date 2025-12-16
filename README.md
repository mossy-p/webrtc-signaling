# WebRTC Signaling Server

A lightweight WebRTC signaling server built in Go for party games. Enables peer-to-peer WebRTC connections with room-based management.

## Features

- **Room-based signaling**: Create rooms with unique codes for easy sharing
- **JWT authentication for room creation**: Only authenticated users can create rooms
- **Frictionless joining**: Guests join with just a room code
- **Origin filtering**: CORS protection for allowed frontend domains
- **Redis-backed**: Uses Redis for connection state and room management
- **WebSocket-based**: Real-time bidirectional communication
- **Party game optimized**: Max players per room, display names, room lifecycle management

## Architecture

### Authentication Flow

1. **Create Room** (authenticated):
   - User logs in via mossp.me-api and gets JWT
   - POST `/api/rooms` with JWT to create room
   - Server generates unique 6-character room code (e.g., "ABCD23")
   - Returns room ID and code

2. **Join Room** (unauthenticated):
   - Guest navigates to game URL with room code
   - Connects to WebSocket at `/ws/signal/:roomCode?displayName=PlayerName`
   - Server validates room exists and isn't full
   - Connection established, peer joins room

### API Endpoints

#### HTTP Endpoints

```http
POST /api/rooms
Authorization: Bearer <jwt-token>
Content-Type: application/json

{
  "maxPlayers": 8
}

Response:
{
  "roomId": "uuid",
  "code": "ABCD23"
}
```

```http
GET /api/rooms/:roomIdOrCode

Response:
{
  "id": "uuid",
  "code": "ABCD23",
  "creatorId": "user-id",
  "createdAt": "2025-12-16T...",
  "maxPlayers": 8,
  "playerCount": 3
}
```

```http
DELETE /api/rooms/:roomId
Authorization: Bearer <jwt-token>

Response:
{
  "message": "Room deleted"
}
```

#### WebSocket Endpoint

```
WS /ws/signal/:roomIdOrCode?displayName=YourName
```

**Message Types:**
- `join` - Sent when peer joins
- `leave` - Sent when peer leaves
- `offer` - WebRTC offer (SDP)
- `answer` - WebRTC answer (SDP)
- `candidate` - ICE candidate

**Message Format:**
```json
{
  "type": "offer|answer|candidate",
  "from": "peer-id",
  "to": "target-peer-id",  // Optional: for directed messages
  "roomId": "room-id",
  "payload": { /* WebRTC data */ }
}
```

## Configuration

Environment variables:

```bash
PORT=8080                    # Server port
ENVIRONMENT=production       # Environment (development|production)
ALLOWED_ORIGINS=https://...  # Comma-separated allowed origins
JWT_SECRET=your-secret       # JWT secret (must match mossp.me-api)

# Redis configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=optional
```

## Local Development

### Prerequisites

- Go 1.23+
- Redis 7+

### Setup

1. Start Redis:
```bash
docker run -d -p 6379:6379 redis:7-alpine
```

2. Run the server:
```bash
go run cmd/signaling/main.go
```

3. Test with WebSocket client:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/signal/TESTROOM?displayName=Player1');

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  console.log('Received:', msg);
};

// Send offer
ws.send(JSON.stringify({
  type: 'offer',
  payload: { /* SDP */ }
}));
```

## Deployment

Follows the same pattern as mossp.me-api:

1. **Build & Push**:
   - Push to `releases/**` branch
   - GitHub Actions builds Docker image
   - Pushes to ghcr.io
   - Triggers GitOps update

2. **GitOps Deployment**:
   - Kubernetes manifests in `moss-server-gitops/applications/webrtc-signaling/`
   - Kustomize for environment management
   - Sealed secrets for sensitive data
   - Tailscale integration for external access

### Deployment Files

- `deployment.yaml` - Kubernetes Deployment
- `service.yaml` - Kubernetes Service with Tailscale
- `redis-deployment.yaml` - Redis instance
- `kustomization.yaml` - Kustomize configuration
- `sealed-secret.yaml` - Sealed secrets template

## Security

- **Origin filtering**: Only allowed domains can connect
- **JWT validation**: Room creation requires authentication
- **Room lifecycle**: Automatic cleanup after 24 hours
- **Connection limits**: Max players per room enforced
- **WebSocket security**: Proper CORS and origin checks

## Testing

```bash
# Run tests
go test ./...

# Build
go build -o signaling ./cmd/signaling

# Docker build
docker build -t webrtc-signaling .
```

## Room Lifecycle

1. **Creation**: User with JWT creates room, gets 6-char code
2. **Active**: Peers join via code, WebRTC signaling occurs
3. **Cleanup**: Room expires after 24 hours or when empty
4. **Manual deletion**: Creator can delete room early

## Redis Schema

```
room:{roomId}          -> JSON metadata (creator, maxPlayers, etc.)
code:{roomCode}        -> roomId (for lookups)
room:{roomId}:peers    -> Set of connected peer IDs
```

## Future Enhancements

- [ ] Rate limiting per IP/user
- [ ] Room persistence beyond 24 hours
- [ ] Analytics/metrics
- [ ] Multiple signaling servers with Redis pub/sub
- [ ] Turn server integration
- [ ] Reconnection handling
- [ ] Room passwords/privacy settings

## License

MIT
