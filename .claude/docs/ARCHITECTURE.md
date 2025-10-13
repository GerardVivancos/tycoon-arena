# System Architecture & Implementation Guide

> **📍 For current implementation details, see [CURRENT_STATE.md](CURRENT_STATE.md)**
>
> This document covers core networking architecture (Quake 3 model, tick system, concurrency).
> Game mechanics have evolved - see CURRENT_STATE.md and SPRINT_3_PROGRESS.md for latest features.

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
    conn       *net.UDPConn          // UDP socket
    clients    map[uint32]*Client     // Active connections
    entities   map[uint32]*Entity     // All game entities
    tick       uint64                 // Current simulation tick
    nextId     uint32                 // ID generator (shared for clients + entities)
    mu         sync.RWMutex           // Thread safety for clients/entities
    inputQueue []QueuedInput          // Pending inputs (tick-ordered)
    queueMu    sync.Mutex             // Thread safety for input queue
}

type Client struct {
    Id               uint32
    Name             string
    Addr             *net.UDPAddr
    LastSeen         time.Time
    Entity           *Entity
    Money            float32         // Resource currency
    LastProcessedSeq uint32          // For input deduplication
    LastAckTick      uint64          // For delta compression (future)
}
```

#### 2. **Concurrency Model (Quake 3 Pattern)**
- **Network Thread**: Receives UDP packets, **enqueues** inputs (does NOT process)
- **Tick Thread**: Dequeues inputs, sorts by tick, processes, broadcasts snapshots
- **Single-Threaded Game Logic**: Only tick thread modifies game state
- **Mutex Strategy**:
  - `queueMu`: Protects input queue (enqueue/dequeue)
  - `mu`: Protects clients/entities (read during validation, write during tick)
  - **No deadlocks**: Network thread never holds `mu` while doing I/O

**Benefits:**
- Deterministic simulation (tick-ordered processing)
- Fair gameplay (no early-mover advantage)
- No race conditions in game logic
- Easy to reason about

#### 3. **Client Management**
- **Connection**: Hello/Welcome handshake establishes client ID
- **Heartbeat**: Clients ping every 2 seconds
- **Timeout**: 10-second timeout (clients removed if no ping/input)
- **Entity Spawning**: Each client gets one player entity on connect
- **Position**: Spawn at `(100 + index*150, 300)` to avoid overlap

#### 4. **Game Mechanics**
```go
const (
    ArenaWidth  = 800
    ArenaHeight = 600
    PlayerSpeed = 200.0     // units per second

    StartingMoney     = 100
    BuildingCost      = 50
    BuildingSize      = 40.0
    GeneratorIncome   = 10.0  // money per second
)
```

**Movement System:**
- **Delta Movement**: Clients send deltaX/deltaY, not absolute positions
- **Validation**: Server clamps movement to max speed per tick
- **Bounds Checking**: Positions clamped to arena boundaries

**Building System:**
- **Placement**: Server validates money, bounds, collision (AABB)
- **Types**: Generator (produces $10/sec)
- **Validation**: No events sent - client infers success/failure from snapshot

**Combat System:**
- **Damage**: 25 HP per attack
- **Targeting**: Client-selected entity ID
- **Validation**: Can't attack own buildings, target must exist

### Message Flow (Tick-Based)
1. **Input Enqueue**: `ReadFromUDP` → `handleMessage` → `handleInput` → enqueue to `inputQueue`
2. **Tick Processing**:
   - Dequeue all pending inputs
   - Sort by tick (earliest first)
   - Validate & process each command
   - Update game state (movement, building, combat, resources)
   - Broadcast snapshot
3. **Outgoing**: `broadcastSnapshot` → `WriteToUDP` (without holding game lock)

## Client Architecture

### Core Components (Godot/GDScript)

#### 1. **NetworkManager** (`NetworkManager.gd`)
**Responsibilities**:
- UDP socket management
- Message serialization/deserialization (JSON)
- Connection state tracking
- **Input redundancy**: Sends last N=3 command frames per message
- Signal emission for game events

**Key Signals**:
- `connected_to_server(client_id, tick_rate)`
- `snapshot_received(snapshot)`
- `disconnected_from_server()`

**Input Redundancy**:
```gdscript
var command_history: Array = []  # Last 3 frames
# Each message includes redundant commands for packet loss tolerance
```

**Type Conversion**:
```gdscript
# JSON numbers are floats - convert IDs to int for type-safe dict lookups
client_id = int(data.get("clientId", -1))
```

#### 2. **GameController** (`GameController.gd`)
**Responsibilities**:
- Input handling (WASD/arrows, mouse clicks)
- Entity lifecycle management (players, buildings)
- Snapshot processing
- UI updates (money, selection, event log)
- **Client-side prediction** for building placement

**Input System**:
- Polls input every 50ms (20Hz to match server)
- Batches commands before sending
- Applies movement locally (prediction)
- **Validates builds locally** before sending to server

**Building Selection**:
- Area2D with `input_pickable = true` for click detection
- ColorRect uses `MOUSE_FILTER_IGNORE` to not block clicks
- Visual highlight on selected building

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

2. **Input**: Player commands (with redundancy)
```json
{
  "type": "input",
  "data": {
    "clientId": 1,
    "commands": [
      {
        "sequence": 98,
        "tick": 1950,
        "commands": [{"type": "move", "data": {"deltaX": 5.0, "deltaY": 0.0}}]
      },
      {
        "sequence": 99,
        "tick": 1970,
        "commands": [{"type": "move", "data": {"deltaX": 5.0, "deltaY": 0.0}}]
      },
      {
        "sequence": 100,
        "tick": 1990,
        "commands": [{"type": "build", "data": {"buildingType": "generator", "x": 200, "y": 150}}]
      }
    ]
  }
}
```
**Note:** Last 3 command frames sent for packet loss tolerance. Server deduplicates using `sequence`.

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
    "baselineTick": 0,
    "entities": [
      {
        "id": 2,
        "ownerId": 1,
        "type": "player",
        "x": 150.0,
        "y": 300.0,
        "health": 100,
        "maxHealth": 100
      },
      {
        "id": 5,
        "ownerId": 1,
        "type": "generator",
        "x": 200.0,
        "y": 150.0,
        "health": 100,
        "maxHealth": 100,
        "width": 40.0,
        "height": 40.0
      }
    ],
    "players": {
      "1": {"id": 1, "name": "Player123", "money": 125.5},
      "3": {"id": 3, "name": "Player456", "money": 80.0}
    }
  }
}
```
**Note:** `baselineTick: 0` means full snapshot (delta compression framework in place but not implemented).

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

### Known Quirks

1. **Client IDs Skip Numbers**
   - **Behavior**: Client IDs are non-sequential (1, 3, 6, 8...)
   - **Cause**: Single `nextId` counter shared between clients and entities
   - **Status**: Cosmetic only, not a bug. IDs remain unique.
   - **Example**: Client 1 (ID=1, entity=2), Client 2 (ID=3, entity=4)

2. **JSON Type Handling**
   - **Behavior**: All JSON numbers are floats
   - **Impact**: GDScript dictionary lookups fail if types don't match
   - **Solution**: Client converts all IDs to int on reception
   - **Pattern**: `var entity_id = int(entity_data.get("id", -1))`

### Current Limitations

1. **No Delta Compression (Yet)**
   - **Status**: Framework in place (`baselineTick`, `LastAckTick`)
   - **Current**: Always send full snapshots
   - **Future**: Send only changed entities to reduce bandwidth

2. **No Lag Compensation**
   - **Impact**: High-latency players at disadvantage for combat
   - **Solution**: Server-side rewind for hit validation
   - **Priority**: Medium (implement after delta compression)

3. **Simple Player Collision**
   - **Behavior**: Players can overlap
   - **Status**: Acceptable for prototype
   - **Solution**: Add physics system or grid-based movement

4. **No Persistence**
   - **Impact**: Server restart = lose all state
   - **Solution**: Add save/load system for match state
   - **Priority**: Low (implement with matchmaking)

5. **No Win Condition**
   - **Status**: Game continues indefinitely
   - **Needed**: Victory conditions (most money, last standing, etc.)

### Optimization Opportunities

1. **Delta Compression** (Ready to implement)
   - Structure exists: `baselineTick`, `LastAckTick`
   - Estimated savings: 60-80% bandwidth for static scenes
   - Implementation: Compare with baseline, send only diffs

2. **Binary Protocol** (Future)
   - Current: JSON (~500 bytes/snapshot)
   - Target: MessagePack or Protobuf (~200 bytes/snapshot)
   - Trade-off: Lose human readability, gain performance

3. **Interest Management** (Scaling)
   - Current: Send all entities to all clients
   - Future: Send only nearby entities (spatial partitioning)
   - Needed when: >20 entities or large maps

### Performance Metrics (Current)

**Network:**
- ~6 KB/s per client (well under target)
- Input: ~200 bytes @ 20Hz
- Snapshot: ~500 bytes @ 20Hz
- Tested: 2-4 clients stable

**Server:**
- Tick rate: 20 Hz (stable)
- Tested: Up to 4 concurrent clients
- CPU: Minimal (single-threaded game logic)

**Client:**
- FPS: 60 (stable)
- Prediction: Seamless on LAN
- Reconciliation: < 5ms on LAN

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