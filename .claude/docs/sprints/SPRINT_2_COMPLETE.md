# Sprint 2 Complete - Game Rules & Mechanics

**Status:** ✅ Complete
**Duration:** ~2 days
**Date Completed:** 2025-10-05

## Overview

Sprint 2 implemented core game mechanics (building, resources, combat) and refactored the network architecture to follow the Quake 3 model for better performance, reliability, and future scalability.

---

## Accomplishments

### Game Mechanics Implemented

#### 1. Building System
- **Building Placement**: Players can place generator buildings
- **Server-Side Validation**:
  - Money check ($50 cost)
  - Arena bounds checking
  - AABB collision detection (prevents overlapping buildings)
- **Visual Representation**: Buildings rendered as colored squares (40x40)
  - Yellow/orange for own buildings
  - Darker orange for enemy buildings
- **Client-Side Prediction**: Immediate validation feedback without waiting for server

#### 2. Resource Generation
- **Passive Income**: Generators produce $10/second
- **Starting Money**: $100 per player
- **Real-time Updates**: Money displayed in UI, updates every tick
- **Server Authority**: All resource changes validated server-side

#### 3. Combat System
- **Building Selection**:
  - Click to select enemy buildings
  - Visual highlight (yellow border) on selection
  - UI label shows selected target
- **Attack Mechanics**:
  - 25 damage per attack
  - 4 hits to destroy (100 HP buildings)
  - Can only attack enemy buildings
  - Health bars show damage in real-time
- **Hotkey**: Press Q to attack selected target

---

## Network Architecture Refactor

### Major Changes (Quake 3 Model)

#### 1. Input Queue System
**Before:** Inputs processed immediately on network thread
**After:** Inputs enqueued and processed in tick order

```go
type QueuedInput struct {
    ClientId uint32
    Sequence uint32
    Tick     uint64
    Commands []Command
}
```

**Benefits:**
- Fair processing (no early-mover advantage)
- Deterministic simulation
- Better support for lag compensation

#### 2. Tick-Ordered Processing
- Inputs sorted by `tick` field (earliest first)
- Ensures consistent game state across all clients
- Player 1's tick 100 processed before Player 2's tick 101, regardless of network arrival order

#### 3. Input Redundancy (Packet Loss Tolerance)
- Clients send last **N=3 command frames** per message
- Server deduplicates using `LastProcessedSeq`
- Handles UDP packet loss gracefully

**Protocol Change:**
```json
{
  "type": "input",
  "data": {
    "clientId": 1,
    "commands": [
      {"sequence": 98, "tick": 1950, "commands": [...]},
      {"sequence": 99, "tick": 1970, "commands": [...]},
      {"sequence": 100, "tick": 1990, "commands": [...]}
    ]
  }
}
```

#### 4. Event System Removed
**Before:** Separate event messages (build_success, build_failed, damage, destroyed)
**After:** All state changes via snapshots only

**Benefits:**
- Simpler protocol (one message type for state)
- No mid-tick messaging (eliminates deadlock issues)
- Client infers outcomes from snapshot changes
- Better for client-side prediction

#### 5. Delta Compression Framework
**Added (not yet implemented):**
```go
type Client struct {
    LastAckTick uint64  // For delta compression
}

type SnapshotMessage struct {
    BaselineTick uint64  // 0 = full snapshot
    // ...
}
```

Structure in place for future optimization - currently always sends full snapshots.

#### 6. Single-Threaded Game Logic
**Before:** Network thread and tick thread both modified game state (mutex hell)
**After:** Only tick thread modifies state (input queue acts as boundary)

**Benefits:**
- No deadlocks
- Simpler code
- Easier to reason about
- Better performance (less lock contention)

---

## Technical Details

### Server Changes (`server/main.go`)

**New Data Structures:**
```go
type GameServer struct {
    // ... existing fields ...
    inputQueue []QueuedInput
    queueMu    sync.Mutex
}

type Client struct {
    // ... existing fields ...
    Money            float32
    LastProcessedSeq uint32
    LastAckTick      uint64
}
```

**Key Functions:**
- `handleInput()` - now just enqueues (no processing)
- `gameTick()` - dequeues, sorts by tick, processes all inputs
- `processCommand()` - dispatches move/build/attack
- `handleBuildCommand()` - validates and creates buildings
- `handleAttackCommand()` - applies damage, handles destruction

### Client Changes

**New Features:**
- Input command history (last 3 frames)
- Client-side building validation
- Building selection with Area2D input handling
- Visual selection highlights
- JSON type handling (float→int conversion for IDs)

**Bug Fixes:**
- ColorRect `mouse_filter` set to `IGNORE` (was blocking Area2D clicks)
- All IDs converted to `int` on reception (JSON numbers are floats)

**UI Additions:**
- Money display
- Build button ($50 cost shown)
- Attack button with hotkey (Q)
- Selection label
- Event log

---

## Known Issues & Quirks

### 1. Client IDs Skip Numbers
**Example:** Client 1, Client 3, Client 6...

**Cause:** Single `nextId` counter used for both clients and entities
- Client 1 gets ID 1, their entity gets ID 2
- Client 2 gets ID 3, their entity gets ID 4
- etc.

**Status:** Cosmetic only, not a bug. IDs are still unique.

### 2. Building Selection Required Click Debugging
**Issue:** Initial implementation had Area2D clicks not working

**Root Causes Found:**
- `input_pickable` not set on Area2D
- ColorRect blocking input (needed `MOUSE_FILTER_IGNORE`)
- Type mismatch: entity IDs stored as float, searched as int

**Resolution:** Fixed via guided debugging session (excellent learning moment!)

### 3. JSON Type Handling
**Issue:** All JSON numbers are floats, GDScript dictionaries are type-strict

**Solution:** Convert all IDs to int on client:
```gdscript
var entity_id = int(entity_data.get("id", -1))  # JSON→int
```

Applied to: entity_id, owner_id, player_id, client_id, tick_rate, etc.

---

## Dev Tools Added

### Multi-Client Launch Script (`launch_all.sh`)
```bash
./launch_all.sh 2    # Start server + 2 clients
./launch_all.sh 4    # Start server + 4 clients
```

**Features:**
- Starts Go server automatically
- Launches N Godot client windows
- Color-coded terminal output:
  - `[SERVER]` - Yellow
  - `[CLIENT1]` - Green
  - `[CLIENT2]` - Blue
  - `[CLIENT3]` - Cyan
  - etc.
- Ctrl+C stops everything cleanly

---

## Testing Results

### Multiplayer Functionality
- ✅ Multiple clients connect successfully
- ✅ Movement synced across clients
- ✅ Building placement works (with validation)
- ✅ Resources generate properly ($10/sec)
- ✅ Combat works (damage, destruction)
- ✅ Client timeout/disconnect handled cleanly
- ✅ Input redundancy handles simulated packet loss

### Performance Metrics
- **Tick Rate:** 20 Hz (stable)
- **Snapshot Rate:** 20 Hz (same as tick)
- **Input Send Rate:** 20 Hz (client-side)
- **Heartbeat Interval:** 2 seconds
- **Client Timeout:** 10 seconds

### Network Bandwidth (estimated)
- **Input Message:** ~200 bytes (3 redundant frames)
- **Snapshot Message:** ~500 bytes (10 entities, 4 players)
- **Total:** ~6 KB/sec per client (well under 10 KB/sec target)

---

## Code Quality

### Documentation
- ✅ Network protocol formally documented (`.claude/docs/NETWORK_PROTOCOL.md`)
- ✅ Code comments explain non-obvious patterns (e.g., JSON type conversion)
- ✅ Sprint completion documented (this file)

### Testing Approach
- Manual testing with 2-4 clients
- Server logs validated for correct behavior
- Client-side debugging via print statements
- Guided debugging to teach Godot tools (Remote tab, print debugging, type checking)

---

## Lessons Learned

### 1. JSON Limitations
JSON has no integer type - all numbers are doubles/floats. This caused type mismatches in GDScript dictionaries. **Solution:** Always convert IDs to int on client side.

### 2. Godot Input Handling
Control nodes (ColorRect) block input by default. Area2D needs:
- `input_pickable = true`
- No blocking UI elements in front (or set `mouse_filter = IGNORE`)

### 3. Network Architecture
Events as separate messages caused deadlocks when trying to send mid-processing. **Snapshot-only architecture is simpler and more robust.**

### 4. Debugging Approach
Guided debugging was effective:
- Remote tab to inspect runtime nodes
- Print statements to trace execution
- Type checking with `typeof()`
- Incremental problem isolation

---

## Next Steps (Sprint 3)

### Immediate
- Tune gameplay balance (damage, costs, generation rates)
- Add win condition
- Improve UX (better visual feedback, sounds)

### Network Optimizations
- Implement delta compression (framework is ready)
- Add lag compensation for hit detection
- Optimize snapshot size (send only changed entities)

### Game Features
- More building types (defense, special abilities)
- Player movement improvements
- Map/terrain system
- Matchmaking/lobby system

---

## Files Changed

### Server
- `server/main.go` - Major refactor (input queue, tick processing, game mechanics)

### Client
- `client/GameController.gd` - Building system, combat, selection, type conversion
- `client/NetworkManager.gd` - Input redundancy, type conversion
- `client/Main.tscn` - UI additions (money, buttons, selection label, event log)
- `client/Player.gd` - Minor updates

### Documentation
- `.claude/docs/NETWORK_PROTOCOL.md` - NEW: Formal protocol spec
- `.claude/docs/sprints/SPRINT_2_COMPLETE.md` - NEW: This file
- `CLAUDE.md` - Updated sprint status
- `.claude/docs/README.md` - Updated index

### Dev Tools
- `launch_all.sh` - NEW: Multi-client launcher with color-coded logs

---

## Summary

Sprint 2 successfully delivered a **playable prototype** with building, resource generation, and combat. The network architecture refactor to the Quake 3 model provides a solid foundation for future features and optimizations. The game is now in a state where gameplay iteration and balancing can begin.

**Key Achievement:** Debugged and fixed building selection collaboratively, teaching Godot debugging techniques in the process!
