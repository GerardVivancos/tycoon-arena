# System Architecture & Implementation Guide

## Table of Contents
1. [System Overview](#system-overview)
2. [Server Architecture](#server-architecture)
3. [Client Architecture](#client-architecture)
4. [Network Protocol](#network-protocol)
5. [Key Implementation Decisions](#key-implementation-decisions)
6. [Code Organization](#code-organization)
7. [Development Workflow](#development-workflow)
8. [Known Issues & Future Work](#known-issues--future-work)

## System Overview

### Architecture Pattern
- **Client-Server Model**: Authoritative dedicated server with predictive clients
- **Tick-Based Simulation**: Fixed 20Hz tick rate (50ms per tick)
- **Transport**: UDP with application-level reliability for critical messages
- **Serialization**: JSON (chosen for rapid prototyping, easy debugging)

### Component Diagram
```
┌──────────────┐         UDP:8080          ┌──────────────┐
│ Godot Client ├──────────────────────────►│   Go Server  │
│              │◄──────────────────────────┤              │
│ - Prediction │      JSON Messages        │ - Game State │
│ - Interpol.  │                           │ - Validation │
│ - Rendering  │                           │ - Tick Loop  │
└──────────────┘                           └──────────────┘
```

## Server Architecture

### Core Components (`server/main.go`)

#### 1. **GameServer Struct**
```go
type GameServer struct {
    conn     *net.UDPConn       // UDP socket
    clients  map[uint32]*Client  // Active connections
    entities map[uint32]*Entity  // All game entities
    tick     uint64              // Current simulation tick
    nextId   uint32              // ID generator
    mu       sync.RWMutex        // Thread safety
}
```

#### 2. **Concurrency Model**
- **Main Thread**: Handles incoming UDP messages
- **Tick Goroutine**: Runs game simulation at 20Hz
- **Mutex Strategy**: RWMutex for client/entity access
  - Write lock during state mutations
  - Read lock during snapshot broadcasting

#### 3. **Client Management**
- **Connection**: Hello/Welcome handshake establishes client ID
- **Timeout**: 30-second keepalive (clients removed if no input)
- **Entity Spawning**: Each client gets one player entity on connect
- **Position**: Spawn at `(100 + index*150, 300)` to avoid overlap

#### 4. **Movement System**
```go
const (
    ArenaWidth = 800
    ArenaHeight = 600
    PlayerSpeed = 200.0  // units per second
)
```
- **Delta Movement**: Clients send deltaX/deltaY, not absolute positions
- **Validation**: Server clamps movement to max speed per tick
- **Bounds Checking**: Positions clamped to arena boundaries

### Message Flow
1. **Incoming**: `ReadFromUDP` → `handleMessage` → type-specific handler
2. **Game Loop**: `tickLoop` → `gameTick` → `broadcastSnapshot`
3. **Outgoing**: `sendMessage` / `broadcastMessage` → `WriteToUDP`

## Client Architecture

### Core Components (Godot/GDScript)

#### 1. **NetworkManager** (`NetworkManager.gd`)
**Responsibilities**:
- UDP socket management
- Message serialization/deserialization
- Connection state tracking
- Signal emission for game events

**Key Signals**:
- `connected_to_server(client_id, tick_rate)`
- `snapshot_received(snapshot)`
- `disconnected_from_server()`

#### 2. **GameController** (`GameController.gd`)
**Responsibilities**:
- Input handling (WASD/arrows)
- Entity lifecycle management
- Snapshot processing
- UI updates

**Input System**:
- Polls input every 50ms (20Hz to match server)
- Batches commands before sending
- Applies movement locally (prediction)

#### 3. **Player Entity** (`Player.gd`)
**Local Player (Prediction)**:
```gdscript
# Apply predicted movement immediately
predicted_position += movement * speed * delta_time
target_position = predicted_position

# Store for reconciliation
input_buffer.append({
    "movement": movement,
    "position": predicted_position
})
```

**Remote Players (Interpolation)**:
```gdscript
# Smooth interpolation to server position
position = position.lerp(target_position, interpolation_speed * delta)
```

#### 4. **Reconciliation System**
- Server position arrives via snapshot
- Calculate error: `error = server_pos - predicted_pos`
- If error > 2 units: snap to server position
- Otherwise: keep prediction (avoids micro-corrections)

### Scene Structure
```
Main (Node2D)
├── NetworkManager (Node)
├── Camera2D
├── Entities (Node2D)
│   └── [Dynamic Player instances]
└── UI (CanvasLayer)
    ├── ConnectionStatus
    ├── FPS
    └── PlayerList
```

## Network Protocol

### Message Types

#### Client → Server
1. **Hello**: Initial connection
```json
{
  "type": "hello",
  "data": {
    "clientVersion": "1.0",
    "playerName": "Player123"
  }
}
```

2. **Input**: Player commands
```json
{
  "type": "input",
  "data": {
    "tick": 42,
    "clientId": 1,
    "sequence": 10,
    "commands": [{
      "type": "move",
      "data": {"deltaX": 5.0, "deltaY": 0.0}
    }]
  }
}
```

#### Server → Client
1. **Welcome**: Connection confirmation
```json
{
  "type": "welcome",
  "data": {
    "clientId": 1,
    "tickRate": 20
  }
}
```

2. **Snapshot**: World state update
```json
{
  "type": "snapshot",
  "data": {
    "tick": 42,
    "entities": [{
      "id": 2,
      "ownerId": 1,
      "type": "player",
      "x": 150.0,
      "y": 300.0,
      "health": 100,
      "maxHealth": 100
    }]
  }
}
```

### Protocol Characteristics
- **Unreliable**: Position updates via UDP (lost packets acceptable)
- **Eventually Consistent**: Snapshots bring all clients to same state
- **Tick-Aligned**: All updates reference server tick for synchronization

## Key Implementation Decisions

### 1. Why Go for Server?
- **Rapid Prototyping**: Fast compilation, simple syntax
- **Built-in Concurrency**: Goroutines perfect for game loops
- **Standard Library**: `net` package handles UDP excellently
- **Performance**: Sufficient for prototype (can handle 100+ clients)

### 2. Why Godot for Client?
- **Cross-Platform**: Single codebase → Windows/Mac/Linux/iOS
- **Visual Editor**: Rapid UI iteration
- **GDScript**: Simple, Python-like syntax
- **Built-in Networking**: PacketPeerUDP matches our needs

### 3. Why JSON over Binary?
- **Debugging**: Human-readable in packet captures
- **Flexibility**: Easy to add fields during development
- **Compatibility**: Works everywhere without code generation
- **Trade-off**: Higher bandwidth (acceptable for prototype)

### 4. Why Client-Side Prediction?
- **Responsiveness**: Immediate feedback on input
- **Latency Hiding**: 50-100ms latency becomes invisible
- **Industry Standard**: Proven technique from Quake/Source/Unreal

### 5. Why 20Hz Tick Rate?
- **Balance**: Good responsiveness vs. bandwidth
- **Sufficient**: Smooth for strategy/building game
- **Headroom**: Can increase to 30-60Hz if needed

## Code Organization

### Directory Structure
```
realtime-game-engine/
├── server/
│   ├── main.go          # Entire server (intentionally monolithic)
│   └── go.mod
├── client/
│   ├── project.godot    # Godot project config
│   ├── Main.tscn        # Main scene
│   ├── Player.tscn      # Player prefab
│   ├── GameController.gd
│   ├── NetworkManager.gd
│   └── Player.gd
├── test_client.go       # Go test client
├── launch_client.sh     # Client launcher script
├── Claude.md           # Project instructions
└── .claude/
    └── docs/
        ├── README.md
        ├── ARCHITECTURE.md (this file)
        ├── planning/
        └── sprints/
```

### Why Monolithic?
- **Prototype Speed**: Everything in one file = fast iteration
- **Easy Debugging**: No jumping between files
- **Refactor Later**: Split when patterns emerge

## Development Workflow

### Running the System

1. **Start Server**:
```bash
cd server
go run main.go
# Output: "Game server listening on :8080"
```

2. **Launch Clients**:
```bash
./launch_client.sh  # First client
./launch_client.sh  # Second client (new terminal)
```

3. **Monitor Server**:
- Watch server terminal for connection logs
- Each client shows as "Client X (PlayerY) connected"

### Testing Changes

#### Server Changes:
1. Ctrl+C to stop server
2. Make edits to `server/main.go`
3. `go run main.go` to restart
4. Clients auto-reconnect

#### Client Changes:
1. Edit `.gd` files in Godot editor or text editor
2. Godot hot-reloads scripts automatically
3. Press F6 in Godot to restart scene

### Debugging Tips

1. **Network Issues**:
   - Use Wireshark filter: `udp.port == 8080`
   - Add logging: `log.Printf("Received: %+v", message)`

2. **Movement Issues**:
   - Server: Log position after each update
   - Client: Draw debug lines for predicted vs actual

3. **Entity Spawning**:
   - Check server `entities` map
   - Verify client `entities` dictionary matches

## Known Issues & Future Work

### Current Limitations

1. **No Reliable Messaging**
   - Building placement needs guaranteed delivery
   - Solution: Add sequence numbers + ACK system

2. **No Lag Compensation**
   - High latency players at disadvantage
   - Solution: Server-side rewind for hit validation

3. **Simple Collision**
   - Players can overlap
   - Solution: Add physics system or grid-based placement

4. **No Persistence**
   - Server restart = lose all state
   - Solution: Add save/load system

### Sprint 2 Preparations

To implement building mechanics, you'll need:

1. **New Message Types**:
   - `BuildRequest` (client → server)
   - `BuildResponse` (server → client, reliable)
   - `BuildingUpdate` in snapshots

2. **Server Validation**:
   - Check resources
   - Verify placement location
   - Prevent overlaps

3. **Client UI**:
   - Build menu
   - Resource display
   - Placement preview

4. **Entity Types**:
   - Extend entity system for buildings
   - Add `Building` type with production logic

### Performance Considerations

**Current Performance**:
- 2-3 KB/s per client
- 50ms tick = smooth movement
- 2-6 players tested successfully

**Scaling Bottlenecks**:
1. **Broadcast Snapshots**: O(n²) with n clients
   - Solution: Interest management (only send nearby entities)

2. **JSON Parsing**: CPU overhead
   - Solution: Switch to binary (MessagePack/Protobuf)

3. **Single-threaded Physics**: All updates in tick loop
   - Solution: Parallel processing per chunk/region

### Architecture Evolution

**Phase 1 (Current)**: Monolithic prototype
**Phase 2**: Extract modules (network, game logic, physics)
**Phase 3**: Microservices (game servers, lobby, persistence)

## Handoff Notes

### Critical Files to Understand

1. **Server Core**: `server/main.go`
   - Start with `handleInput()` - validates all player actions
   - `gameTick()` - authoritative state updates
   - `broadcastSnapshot()` - state synchronization

2. **Client Prediction**: `client/Player.gd`
   - `apply_input()` - prediction logic
   - `update_from_snapshot()` - reconciliation

3. **Networking**: `client/NetworkManager.gd`
   - `send_input()` - command batching
   - `handle_snapshot()` - state updates

### Quick Wins for Improvement

1. **Add Building Placement** (Sprint 2):
   - New entity type in server
   - Place command in protocol
   - Preview system in client

2. **Improve UI**:
   - Better health bars
   - Player names above entities
   - Mini-map

3. **Add Resources**:
   - Currency field on players
   - Generation from buildings
   - UI display

### Testing Checklist

Before each major change:
- [ ] Server compiles: `go build server/main.go`
- [ ] Client loads: Open in Godot
- [ ] Single player connects and moves
- [ ] Two players see each other
- [ ] Movement is smooth (no jitter)
- [ ] Disconnection handled gracefully

## Questions & Decisions Log

**Q: Why not use Godot's high-level multiplayer API?**
A: Need full control over prediction/reconciliation logic. High-level API assumes authoritative nodes, not tick-based simulation.

**Q: Why separate NetworkManager from GameController?**
A: Separation of concerns. Network can be reused; game logic is specific.

**Q: Why 800x600 arena?**
A: Reasonable size for 2-6 players, fits on most screens without scrolling.

**Q: Why spawn players at different X positions?**
A: Prevents initial overlap, gives visual separation for testing.

---

This architecture is designed for rapid iteration while maintaining clean separation between server authority and client prediction. The code is intentionally simple to modify quickly as gameplay emerges.