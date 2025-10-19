# Server Development Handoff Guide

**Last Updated:** 2025-10-20
**For:** Multiplayer RTS Game Server (Go)
**Target Audience:** Developer taking over server development

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [System Architecture](#system-architecture)
3. [Code Organization](#code-organization)
4. [Core Systems](#core-systems)
5. [How to Add Features](#how-to-add-features)
6. [Testing Guide](#testing-guide)
7. [Common Pitfalls](#common-pitfalls)
8. [Performance Considerations](#performance-considerations)
9. [Debugging Tips](#debugging-tips)

---

## Quick Start

### Running the Server

```bash
cd server
go run main.go
# Output: "Game server listening on :8080"
```

### Running Tests

```bash
cd server
go test -v                     # All tests
go test -run TestPathfinding   # Specific test
go test ./...                  # All packages (after reorganization)
```

### Making Changes

1. Edit `server/main.go` (currently monolithic)
2. Ctrl+C to stop server
3. `go run main.go` to restart
4. Clients auto-reconnect

### Current State

- **Language:** Go 1.21+
- **Architecture:** Tick-based authoritative server (Quake 3 model)
- **Protocol:** JSON over UDP
- **Tick Rate:** 20 Hz (50ms per tick)
- **Port:** 8080
- **Max Clients:** 6
- **Map:** 40×30 tiles, loaded from JSON

---

## System Architecture

### High-Level Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   Network Thread (Goroutine)                 │
│  - Receives UDP packets                                      │
│  - Deserializes JSON messages                                │
│  - Enqueues inputs (does NOT process game logic)            │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼ (input queue)
┌─────────────────────────────────────────────────────────────┐
│                    Tick Thread (Goroutine)                   │
│  - Dequeues inputs                                           │
│  - Sorts by tick (earliest first)                            │
│  - Processes commands (move, build, attack)                  │
│  - Updates entities (movement, pathfinding, formations)      │
│  - Ticks production (resources from buildings)               │
│  - Broadcasts snapshots (world state to all clients)         │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼ (snapshot broadcast)
┌─────────────────────────────────────────────────────────────┐
│                        Godot Clients                         │
│  - Receive snapshots                                         │
│  - Render game state                                         │
│  - Send input commands                                       │
└─────────────────────────────────────────────────────────────┘
```

### The Quake 3 Network Model

**Key Principle:** Network thread NEVER modifies game state directly.

**Why?**
- Deterministic simulation (tick-ordered processing)
- No race conditions
- Fair gameplay (no early-mover advantage)
- Easy to debug (single-threaded game logic)

**Implementation:**
1. Network thread receives input → enqueues it
2. Tick thread dequeues → processes → updates game state
3. Snapshots broadcast without holding game lock

**Concurrency:**
- `queueMu`: Protects input queue (enqueue/dequeue)
- `mu`: Protects clients/entities (game state)
- **No deadlocks:** Network thread never holds `mu` during I/O

---

## Code Organization

### Current Structure (Monolithic)

```
server/
├── main.go              # Everything (1952 lines)
├── game_test.go         # Unit tests
├── scenario_test.go     # Scenario tests
├── testutil/            # Test framework
│   ├── scenario.go
│   ├── scenario_runner.go
│   ├── test_server.go
│   └── assertions.go
└── go.mod
```

**Why monolithic?**
- Rapid prototyping
- Easy debugging
- Fast iteration

**When to split?**
- Now! Code is stable enough to organize into packages.

---

### Planned Structure (Modular)

**See:** `.claude/docs/SANDBOX_ECONOMY_PLAN.md` Phase 0.2

```
server/
├── main.go              # Entry point (~50 lines)
├── go.mod
├── game/                # Game logic
│   ├── server.go       # GameServer struct + Start()
│   ├── tick.go         # gameTick() + game loop
│   ├── commands.go     # Command handlers
│   ├── buildings.go    # Building logic
│   ├── resources.go    # Resource management
│   └── workers.go      # Worker assignment
├── movement/            # Pathfinding & movement
│   ├── pathfinding.go  # A* algorithm
│   ├── formations.go   # Formation calculations
│   └── movement.go     # Entity movement
├── network/             # Networking layer
│   ├── protocol.go     # Message types
│   ├── handlers.go     # Message handling
│   └── serialization.go
└── types/               # Shared types
    ├── entity.go       # Entity, Client structs
    ├── message.go      # Message structs
    └── map.go          # Map types
```

---

## Core Systems

### 1. Game Loop (Tick System)

**Location:** `main.go:336-343` (`tickLoop()`)

```go
func (s *GameServer) tickLoop() {
    ticker := time.NewTicker(time.Duration(1000/TickRate) * time.Millisecond)
    defer ticker.Stop()

    for range ticker.C {
        s.gameTick()  // Called every 50ms
    }
}
```

**What happens each tick:**
1. Dequeue all pending inputs
2. Sort by tick (earliest first)
3. Process commands (move, build, attack)
4. Update movement (pathfinding, formations)
5. Tick production (resources from buildings)
6. Clean up disconnected clients
7. Create snapshot
8. Broadcast snapshot to all clients

**Key Code:** `main.go:345-445` (`gameTick()`)

**Critical:** All game state changes happen here. Never modify game state from network thread!

---

### 2. Networking & Protocol

**Message Flow:**

```
Client                          Server
   │                              │
   ├─── Hello ──────────────────→ │ (Connect)
   │                              │
   │←────── Welcome ─────────────┤ (Send client ID)
   │                              │
   ├─── Input (commands) ───────→ │ (Enqueue)
   │                              │
   │←────── Snapshot ────────────┤ (World state)
   │                              │
   ├─── Ping ───────────────────→ │ (Heartbeat)
   │←────── Pong ────────────────┤
```

**Message Types:**

| Type | Direction | Purpose |
|------|-----------|---------|
| Hello | Client→Server | Initial connection |
| Welcome | Server→Client | Connection confirmed + client ID |
| Input | Client→Server | Player commands (move, build, attack) |
| Snapshot | Server→Client | World state (entities, players) |
| Ping/Pong | Bidirectional | Heartbeat (every 2s) |

**Input Redundancy:**
- Clients send last 3 command frames per message
- Server deduplicates using `sequence` number
- Tolerates packet loss

**Code Locations:**
- Protocol types: `main.go:37-125`
- Message handling: `main.go:447-488`
- Input processing: `main.go:611-643`

---

### 3. Entity System

**Entity Types:**
- `worker` - Player units (5 per player)
- `generator` - Building that produces resources
- `hq` - Headquarters (planned, not implemented)

**Entity Struct:**

```go
type Entity struct {
    Id              uint32
    OwnerId         uint32
    Type            string
    TileX           int       // Current tile position
    TileY           int
    TargetTileX     int       // Next waypoint
    TargetTileY     int
    MoveProgress    float32   // 0.0 to 1.0
    Health          int32
    MaxHealth       int32
    FootprintWidth  int       // For buildings (0 for units)
    FootprintHeight int

    // Pathfinding (server-only)
    Path        []TilePosition  // Full path to goal
    PathIndex   int             // Current waypoint
    BlockedTime float32         // For rerouting
}
```

**Key Points:**
- **Tile-based:** All positions are tile coordinates (not pixels)
- **Server authoritative:** Client never modifies entity state
- **Path is server-only:** Client receives current + next tile, not full path
- **Interpolation:** Client lerps between tiles using `MoveProgress`

**Code:** `main.go:127-145`

---

### 4. Pathfinding System

**Algorithm:** A* with Manhattan distance heuristic

**Features:**
- 4-directional movement (N, E, S, W)
- Avoids terrain obstacles (rocks)
- Avoids buildings
- Avoids other unit destinations
- Dynamic collision avoidance (pause when next waypoint occupied)
- Automatic rerouting after 1 second of blocking

**Key Functions:**

| Function | Purpose | Line # |
|----------|---------|--------|
| `findPath()` | A* pathfinding | 917-1012 |
| `isTilePassable()` | Terrain + building check | 1712-1746 |
| `isTileOccupiedByUnit()` | Unit collision check | 1749-1776 |
| `updateEntityMovement()` | Move along path | 656-755 |

**Friendly Unit Pass-Through:**
- Units with same `OwnerId` can pass through each other
- Enemy units still block
- **Code:** `main.go:690-693`

**Rerouting:**
- If blocked for >1 second, recalculate path
- Finds alternate route around obstacle
- **Code:** `main.go:706-725`

**Tests:**
- `TestPathfindingStraightLine()`
- `TestPathfindingAroundObstacle()`
- `TestPathfindingCollisionAvoidance()`

**Docs:** `.claude/docs/PATHFINDING_IMPLEMENTATION.md`

---

### 5. Formation System

**Formation Types:**
- **Box:** Grid (√n × √n)
- **Line:** Horizontal line
- **Spread:** Spiral from center

**Direction-Aware Formations:**
- Formations orient based on movement direction
- Tip unit (closest to click) at click point
- Rest of formation extends backward toward origin
- 8 compass directions: N, NE, E, SE, S, SW, W, NW

**How It Works:**
1. Client sends move command with `unitIds[]` and `formation` type
2. Server sorts units by distance to click (closest = tip/leader)
3. Calculates formation positions (oriented by movement direction)
4. Each unit pathfinds to its final position **independently**
5. No leader-follower during travel (simpler, more reliable)

**Key Functions:**

| Function | Purpose | Line # |
|----------|---------|--------|
| `handleMoveCommand()` | Formation setup | 1431-1662 |
| `calculateFormation()` | Get formation positions | 1014-1027 |
| `getPrimaryDirection()` | 8-way direction | 1400-1429 |

**Formation Movement:**
- **Leader:** Closest unit to click point
- **Followers:** All other units
- **Offsets:** Relative position to leader's final position
- **Movement:** All units pathfind independently (no bouncing)
- **Disbanding:** Formation disbands when all units arrive

**Code:**
- Formation struct: `main.go:158-168`
- Formation tick: `main.go:758-806`
- Movement command: `main.go:1431-1662`

**Tests:**
- `TestBoxFormation()`
- `TestLineFormationBackwardExtension()`
- `TestFormationDirection()`

**Docs:** `.claude/docs/FORMATION_REFACTOR.md`

---

### 6. Building System

**Current Buildings:**
- **Generator:** 2×2 footprint, $50 cost, produces $10/sec

**Placement Validation:**
1. Check money
2. Check bounds (footprint must fit in map)
3. Check collision (all tiles in footprint must be free)

**Code:**
- Build command: `main.go:1793-1859`
- Collision check: `main.go:1664-1675`

**Planned Improvements:**
- Construction system (workers build over time)
- More building types (HQ, Shop, Housing, Storage)
- Resource costs (money + materials)
- Worker assignment to operate buildings

**See:** `.claude/docs/SANDBOX_ECONOMY_PLAN.md` Phase 2

---

### 7. Combat System

**Current:**
- Click enemy building to select
- Press Q (or Attack button) to attack
- 25 damage per attack
- Instant damage (no projectiles)

**Validation:**
- Can't attack own buildings
- Target must exist
- Only buildings can be attacked (not units)

**Code:** `main.go:1861-1901`

**Planned Improvements:**
- Worker combat (workers attack each other)
- Range checking
- Attack cooldowns
- Projectiles or melee animations

---

### 8. Map System

**Format:** JSON files in `maps/` directory

**Current Map:** `maps/default.json` (40×30 tiles)

**Map Features:**
- Dynamic size (sent to clients via Welcome message)
- Terrain types (grass, rock, dirt, water, tree)
- Passability flags (rocks block movement)
- Spawn points (team-based)
- Features (multi-tile obstacles)

**Loading:**
```go
mapData, err := LoadMap("../maps/default.json")
```

**Code:**
- Map types: `main.go:171-234`
- Map loader: `main.go:269-314`
- Passability check: `main.go:1712-1746`

**Test Maps:**
- `test_single_rock.json` - Pathfinding around 1 rock
- `test_rock_cluster.json` - Navigate through cluster
- `test_corridor.json` - Narrow passage

**Docs:** `.claude/docs/MAP_SYSTEM.md`

---

### 9. Testing Framework

**Test Types:**
1. **Unit Tests:** Test individual functions (pathfinding, formations)
2. **Scenario Tests:** Declarative JSON → automated execution

**Running Tests:**
```bash
go test -v                    # All tests
go test -run TestPathfinding  # Specific test
go test -run TestAllScenarios # Scenario tests
```

**Current Tests:**
- 15 unit tests (pathfinding, formations, collisions)
- 2 scenario tests (navigate around rock, formation movement)

**Scenario Format:**
```json
{
  "name": "Navigate Around Rock",
  "mapFile": "test_single_rock.json",
  "setup": {
    "units": [{"id": 100, "position": {"x": 5, "y": 5}}]
  },
  "actions": [
    {"tick": 10, "command": {"type": "move", "data": {...}}}
  ],
  "expectations": {
    "finalState": {
      "units": [{"id": 100, "position": {"x": 15, "y": 5}, "tolerance": 2}]
    }
  }
}
```

**Scenario Tests:**
- `maps/scenarios/navigate_around_rock.json`
- `maps/scenarios/formation_around_cluster.json`

**Docs:** `.claude/docs/TEST_FRAMEWORK.md`

---

## How to Add Features

### Adding a New Command

**Example:** Add a "hire worker" command

#### 1. Define Command Type

**In `main.go` (or `types/message.go` after reorganization):**

```go
type HireWorkerCommand struct {
    // No data needed - just hire 1 worker
}
```

#### 2. Add Command Handler

**In `main.go:645-654` (or `game/commands.go`):**

```go
func (s *GameServer) processCommand(cmd Command, client *Client) {
    switch cmd.Type {
    case "move":
        s.handleMoveCommand(cmd, client)
    case "build":
        s.handleBuildCommand(cmd, client)
    case "attack":
        s.handleAttackCommand(cmd, client)
    case "hireWorker":  // NEW
        s.handleHireWorkerCommand(cmd, client)
    }
}

func (s *GameServer) handleHireWorkerCommand(cmd Command, client *Client) {
    const HireCost = 30.0

    // Validate money
    if client.Money < HireCost {
        return
    }

    // Deduct money
    client.Money -= HireCost

    // Create worker entity
    entityId := s.nextId
    s.nextId++

    // Find spawn position near HQ or first building
    spawnX, spawnY := s.findPlayerSpawnPosition(client.Id)

    worker := &Entity{
        Id:          entityId,
        OwnerId:     client.Id,
        Type:        "worker",
        TileX:       spawnX,
        TileY:       spawnY,
        TargetTileX: spawnX,
        TargetTileY: spawnY,
        Health:      100,
        MaxHealth:   100,
    }

    s.entities[entityId] = worker
    client.OwnedUnits = append(client.OwnedUnits, entityId)

    log.Printf("Client %d hired worker %d", client.Id, entityId)
}
```

#### 3. Update Client

**In `client/GameController.gd`:**

```gdscript
func _on_hire_worker_button_pressed():
    if local_money < 30:
        log_event("Not enough money to hire worker!")
        return

    var commands = [{
        "type": "hireWorker",
        "data": {}
    }]
    network_manager.send_input(commands)
    log_event("Hiring worker...")
```

#### 4. Test

**Add unit test in `server/game_test.go`:**

```go
func TestHireWorker(t *testing.T) {
    server := NewTestServer()
    client := server.AddTestClient("Player1")

    // Set initial money
    client.Money = 100.0
    initialWorkers := len(client.OwnedUnits)

    // Send hire command
    cmd := Command{Type: "hireWorker", Data: map[string]interface{}{}}
    server.ProcessCommand(cmd, client)

    // Verify
    assert.Equal(t, 70.0, client.Money, "Money should be deducted")
    assert.Equal(t, initialWorkers+1, len(client.OwnedUnits), "Should have 1 more worker")
}
```

---

### Adding a New Building Type

**Example:** Add a "Barracks" building

#### 1. Add Building Definition

**In `main.go` constants (or `game/buildings.go`):**

```go
const (
    BuildingCostBarracks  = 150
    BarracksFootprintW    = 3
    BarracksFootprintH    = 3
)
```

#### 2. Update Build Command Handler

**In `main.go:1793-1859` (or `game/commands.go`):**

```go
func (s *GameServer) handleBuildCommand(cmd Command, client *Client) {
    // ... existing code

    var footprintWidth, footprintHeight int
    var cost float32

    switch buildingType {
    case "generator":
        footprintWidth = 2
        footprintHeight = 2
        cost = BuildingCost
    case "barracks":  // NEW
        footprintWidth = BarracksFootprintW
        footprintHeight = BarracksFootprintH
        cost = BuildingCostBarracks
    default:
        return
    }

    // Validate money
    if client.Money < cost {
        return
    }

    // ... rest of validation + creation
}
```

#### 3. Add Building Behavior

**If building produces units:**

**Add to `gameTick()` in `main.go:345-445`:**

```go
// Generate workers from barracks
for _, entity := range s.entities {
    if entity.Type == "barracks" {
        // TODO: Implement worker production logic
    }
}
```

#### 4. Update Client Rendering

**In `client/GameController.gd:661-757` (`create_building()`):**

```gdscript
func create_building(entity_id, owner_id, pos, footprint_w, footprint_h, building_type, health, max_health):
    var building = Node2D.new()
    # ... existing setup

    # Choose color based on type
    var base_color
    match building_type:
        "generator":
            base_color = Color(1, 0.8, 0, 1)  # Gold
        "barracks":  # NEW
            base_color = Color(0.6, 0.6, 0.8, 1)  # Blue-gray
        _:
            base_color = Color(0.8, 0.8, 0.8, 1)

    # ... rest of rendering
```

---

## Testing Guide

### Unit Testing Best Practices

**1. Use Test Utilities**

```go
func TestSomething(t *testing.T) {
    server := testutil.NewTestServer()
    client := server.AddTestClient("Player1")

    // ... test logic

    testutil.AssertPosition(t, entity, expectedX, expectedY, tolerance)
}
```

**2. Test One Thing Per Test**

```go
// Good
func TestPathfindingStraightLine(t *testing.T) { /* ... */ }
func TestPathfindingAroundObstacle(t *testing.T) { /* ... */ }

// Bad
func TestPathfinding(t *testing.T) {
    // Tests 10 different scenarios
}
```

**3. Use Descriptive Names**

```go
// Good
func TestWorkerCannotBuildWithoutMoney(t *testing.T) { /* ... */ }

// Bad
func TestBuild(t *testing.T) { /* ... */ }
```

---

### Scenario Testing

**When to Use:**
- Testing multi-step behaviors (movement → build → attack)
- Testing formation movement
- Testing complex interactions

**Creating a Scenario:**

1. Create JSON file in `maps/scenarios/`
2. Define setup (units, buildings)
3. Define actions (commands at specific ticks)
4. Define expectations (final state, constraints)

**Example:**
```json
{
  "name": "Build Generator",
  "mapFile": "default.json",
  "setup": {
    "units": [
      {"id": 100, "type": "worker", "position": {"x": 10, "y": 10}, "ownerId": 1}
    ]
  },
  "actions": [
    {
      "tick": 10,
      "command": {
        "type": "build",
        "clientId": 1,
        "data": {"buildingType": "generator", "tileX": 15, "tileY": 10}
      }
    }
  ],
  "expectations": {
    "finalState": {
      "buildings": [
        {"type": "generator", "position": {"x": 15, "y": 10}, "ownerId": 1}
      ]
    },
    "constraints": {
      "noCollisions": true
    }
  }
}
```

**Run:** `go test -run TestAllScenarios`

---

## Common Pitfalls

### 1. Modifying Game State from Network Thread

**DON'T:**
```go
func (s *GameServer) handleMessage(msg Message, addr *net.UDPAddr) {
    s.mu.Lock()
    client := s.clients[clientId]
    client.Money += 100  // ❌ Modifying state in network thread!
    s.mu.Unlock()
}
```

**DO:**
```go
func (s *GameServer) handleMessage(msg Message, addr *net.UDPAddr) {
    // Enqueue command
    s.queueMu.Lock()
    s.inputQueue = append(s.inputQueue, QueuedInput{ /* ... */ })
    s.queueMu.Unlock()
}

// In gameTick() (tick thread):
func (s *GameServer) gameTick() {
    s.mu.Lock()
    // Process enqueued commands here
    client.Money += 100  // ✅ Modifying state in tick thread
    s.mu.Unlock()
}
```

---

### 2. Forgetting to Validate Client Ownership

**DON'T:**
```go
func (s *GameServer) handleAttackCommand(cmd Command, client *Client) {
    targetId := /* ... */
    target := s.entities[targetId]
    target.Health -= 25  // ❌ No ownership check!
}
```

**DO:**
```go
func (s *GameServer) handleAttackCommand(cmd Command, client *Client) {
    targetId := /* ... */
    target := s.entities[targetId]

    // Can't attack own entities
    if target.OwnerId == client.Id {
        return  // ✅ Ownership check
    }

    target.Health -= 25
}
```

---

### 3. Off-by-One Errors in Tile Coordinates

**DON'T:**
```go
if tileX <= 0 || tileX >= s.mapData.Width {  // ❌ Allows Width, which is out of bounds!
    return false
}
```

**DO:**
```go
if tileX < 0 || tileX >= s.mapData.Width {  // ✅ Correct bounds check
    return false
}
```

**Remember:** Tiles are 0-indexed. Valid range: `[0, Width)` (0 to Width-1).

---

### 4. JSON Type Confusion

**Problem:** JSON numbers are always `float64`, but entity IDs are `uint32`.

**DON'T:**
```go
unitId := cmd.Data["unitId"].(uint32)  // ❌ Panic! JSON numbers are float64
```

**DO:**
```go
unitIdFloat := cmd.Data["unitId"].(float64)  // ✅ Get as float64
unitId := uint32(unitIdFloat)                 // ✅ Then convert
```

---

### 5. Not Testing for Packet Loss

**Remember:** UDP packets can be lost!

**Client handles this via redundancy:**
- Sends last 3 command frames per message
- Server deduplicates using `sequence` number

**Server must deduplicate:**
```go
if input.Sequence <= client.LastProcessedSeq {
    continue  // ✅ Skip already-processed commands
}
client.LastProcessedSeq = input.Sequence
```

---

## Performance Considerations

### Current Performance

**With 4 clients:**
- Server CPU: ~2-5%
- Tick rate: Stable 20 Hz
- Bandwidth: ~6 KB/s per client
- Snapshot size: ~500 bytes @ 20Hz

**Tested Up To:**
- 4 concurrent clients
- 40 entities (5 workers + buildings per player)
- 1200 terrain tiles (40×30 map)

---

### Scaling Bottlenecks

**1. Broadcast Snapshots - O(n²)**

**Problem:** Every snapshot sent to every client = n clients × m entities.

**Solutions:**
- Interest management (only send nearby entities)
- Delta compression (only send changed entities)
- Spatial partitioning (chunk-based updates)

**When Needed:** >10 clients or >100 entities

---

**2. Pathfinding - O(n log n) per unit**

**Problem:** A* search every move command.

**Solutions:**
- Path caching (reuse paths for similar destinations)
- Hierarchical pathfinding (navmesh + A*)
- Flow fields (for large groups)

**When Needed:** >20 units pathfinding simultaneously

---

**3. JSON Parsing - CPU Overhead**

**Problem:** JSON serialization/deserialization every message.

**Solutions:**
- Switch to MessagePack or Protobuf (binary)
- Pre-allocate buffers
- Use faster JSON library (e.g., `jsoniter`)

**When Needed:** >1000 messages/sec or bandwidth concerns

---

### Optimization Tips

**1. Reduce Snapshot Size**

**Current:** Full snapshot every tick (~500 bytes)

**Future:**
- Delta compression (send only changed entities)
- Quantize floats (0.123456 → 0.12)
- Bitfields for flags

**Estimated Savings:** 60-80%

---

**2. Tick Budget**

**Target:** 50ms per tick (20 Hz)

**Breakdown:**
- Input processing: <5ms
- Pathfinding: <15ms
- Movement update: <5ms
- Production tick: <5ms
- Snapshot creation: <5ms
- Broadcast: <15ms

**Monitor:** Add timing logs if tick starts lagging.

---

**3. Profiling**

```bash
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof
```

**Look for:**
- Hot paths (functions called many times)
- Allocations (GC pressure)
- Lock contention

---

## Debugging Tips

### Enable Detailed Logging

**In `main.go`:**

```go
// Uncomment these lines for debugging
// log.Printf("Processing command: %+v", cmd)
// log.Printf("Entity %d: pos=(%d,%d) target=(%d,%d) progress=%.2f",
//     entity.Id, entity.TileX, entity.TileY, entity.TargetTileX, entity.TargetTileY, entity.MoveProgress)
```

**Tip:** Use conditional logging:
```go
const DEBUG = false  // Change to true when debugging

if DEBUG {
    log.Printf("Debug info: %+v", data)
}
```

---

### Packet Capture

**Use Wireshark:**

1. Start capture on `lo0` (loopback)
2. Filter: `udp.port == 8080`
3. Right-click packet → Follow → UDP Stream
4. See JSON messages

**Useful for:**
- Verifying message format
- Detecting packet loss
- Timing analysis

---

### Test Server (Isolated Testing)

**Use `testutil` for debugging:**

```go
func TestDebugMovement(t *testing.T) {
    server := testutil.NewTestServer()
    client := server.AddTestClient("Player1")

    // Add entity
    entity := &Entity{
        Id: 100,
        OwnerId: client.Id,
        Type: "worker",
        TileX: 10,
        TileY: 10,
    }
    server.AddEntity(entity)

    // Send move command
    cmd := Command{
        Type: "move",
        Data: map[string]interface{}{
            "unitIds": []interface{}{float64(100)},
            "targetTileX": 20.0,
            "targetTileY": 10.0,
            "formation": "box",
        },
    }
    server.ProcessCommand(cmd, client)

    // Inspect state
    log.Printf("Entity path: %+v", entity.Path)
    log.Printf("Entity target: (%d, %d)", entity.TargetTileX, entity.TargetTileY)
}
```

---

### Common Issues

**1. "Unit not moving"**

**Check:**
- Entity has path: `log.Printf("Path: %+v", entity.Path)`
- Path is valid: Not empty, not nil
- Target is passable: `isTilePassable(targetX, targetY)`
- Not blocked by other units

**Fix:**
- Add debug logging in `updateEntityMovement()`
- Verify pathfinding returns valid path

---

**2. "Building not appearing"**

**Check:**
- Server created entity: Check server logs for "built generator"
- Entity in snapshot: Print snapshot entities
- Client receiving snapshot: Check NetworkManager logs
- Client rendering: Verify `create_building()` called

**Fix:**
- Add log in `handleBuildCommand()`
- Add log in client `_on_snapshot_received()`

---

**3. "Client disconnecting"**

**Check:**
- Heartbeat working: Client sends ping every 2s
- Server receiving pings: Check `LastSeen` time
- Timeout threshold: 10 seconds

**Fix:**
- Verify client `send_ping()` is called
- Check server `handlePing()` updates `LastSeen`

---

## Summary Checklist

**Before Making Changes:**
- [ ] Read relevant section of this doc
- [ ] Check existing tests for similar feature
- [ ] Understand tick-based architecture (don't modify state from network thread!)

**When Adding Feature:**
- [ ] Define new types (if needed)
- [ ] Update protocol (message structs)
- [ ] Add command handler (in tick thread)
- [ ] Validate ownership + resources
- [ ] Update client to send command
- [ ] Write unit test
- [ ] Test manually with 2+ clients

**Before Committing:**
- [ ] `go test ./...` passes
- [ ] Server compiles without warnings
- [ ] Client loads without errors
- [ ] Manual playtest works

---

## Next Steps

1. **Read:** `.claude/docs/SANDBOX_ECONOMY_PLAN.md` for next features
2. **Explore:** `server/main.go` to understand current code
3. **Reorganize:** Split monolithic code into packages (Phase 0.2)
4. **Implement:** New economy features (Phases 1-5)

**Questions?** Check:
- `.claude/docs/ARCHITECTURE.md` - System architecture
- `.claude/docs/CURRENT_STATE.md` - Current features
- `.claude/docs/PATHFINDING_IMPLEMENTATION.md` - Pathfinding details
- `.claude/docs/TEST_FRAMEWORK.md` - Testing guide

---

**Good luck with server development!**
