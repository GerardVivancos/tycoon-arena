# Network Protocol Specification

**Version:** 1.0
**Date:** 2025-10-05
**Based on:** Quake 3 Network Model

## Overview

This document specifies the authoritative server network protocol for the realtime multiplayer game. The design prioritizes:
- **Deterministic simulation** via tick-ordered input processing
- **Low latency** via client-side prediction
- **Packet loss tolerance** via input redundancy
- **Scalability** via delta compression (framework in place, implementation deferred)

## Core Principles

1. **Server is authoritative** - all game state changes are validated and applied server-side
2. **Tick-based simulation** - server runs at fixed tick rate (20 Hz default)
3. **Snapshot-only state updates** - no separate event messages; all state flows through snapshots
4. **Client-side prediction** - clients apply inputs immediately and reconcile with server state
5. **Input redundancy** - clients send last N commands to handle packet loss

---

## Network Architecture

### Threading Model

**Server:**
- **Network thread** - receives UDP packets, enqueues inputs, sends snapshots
- **Tick thread** - processes inputs in tick order, updates game state, generates snapshots
- **No shared state modification** - only tick thread modifies game state (eliminates race conditions)

**Client:**
- **Main thread** - Godot game loop, applies predictions, renders
- **Network thread** - handled by Godot's PacketPeerUDP

### Message Flow

```
Client                          Server
------                          ------
  |                               |
  |-- Input (redundant N=3) ----> |  (enqueue)
  |                               |
  |                            [Tick Loop]
  |                         - Dequeue inputs
  |                         - Process in tick order
  |                         - Update game state
  |                         - Generate snapshot
  |                               |
  |<----- Snapshot (full) --------|  (broadcast)
  |                               |
  | (reconcile prediction)        |
  |                               |
```

---

## Message Types

### 1. Hello (Client → Server)

**Purpose:** Initial connection handshake

```json
{
  "type": "hello",
  "data": {
    "clientVersion": "1.0",
    "playerName": "PlayerName"
  }
}
```

### 2. Welcome (Server → Client)

**Purpose:** Connection acknowledgment with server parameters

```json
{
  "type": "welcome",
  "data": {
    "clientId": 1,
    "tickRate": 20,
    "heartbeatInterval": 2000,
    "inputRedundancy": 3
  }
}
```

**Fields:**
- `clientId` - unique identifier for this client
- `tickRate` - server simulation rate (Hz)
- `heartbeatInterval` - milliseconds between ping messages
- `inputRedundancy` - how many commands to send per input message (N)

### 3. Input (Client → Server)

**Purpose:** Send player commands with redundancy for packet loss tolerance

```json
{
  "type": "input",
  "data": {
    "clientId": 1,
    "commands": [
      {
        "sequence": 98,
        "tick": 1950,
        "commands": [
          {"type": "move", "data": {"deltaX": 5.0, "deltaY": 0.0}}
        ]
      },
      {
        "sequence": 99,
        "tick": 1970,
        "commands": [
          {"type": "move", "data": {"deltaX": 5.0, "deltaY": 0.0}}
        ]
      },
      {
        "sequence": 100,
        "tick": 1990,
        "commands": [
          {"type": "move", "data": {"deltaX": 5.0, "deltaY": 0.0}},
          {"type": "build", "data": {"buildingType": "generator", "x": 200, "y": 150}}
        ]
      }
    ]
  }
}
```

**Input Redundancy:**
- Client sends **last N command frames** (default N=3)
- Server tracks `lastProcessedSequence` per client
- Server only processes commands with sequence > lastProcessedSequence
- Handles packet loss: if packet drops, next packet contains missing commands

**Command Frame Fields:**
- `sequence` - monotonically increasing frame number (client-side)
- `tick` - client's estimate of server tick when command was generated
- `commands[]` - array of actions for this frame

**Command Types:**
- `move` - `{deltaX: float, deltaY: float}` - movement delta
- `build` - `{buildingType: string, x: float, y: float}` - place building
- `attack` - `{targetId: uint32}` - damage target entity

### 4. Snapshot (Server → Client)

**Purpose:** Broadcast authoritative game state

```json
{
  "type": "snapshot",
  "data": {
    "tick": 2000,
    "baselineTick": 0,
    "entities": [
      {
        "id": 1,
        "ownerId": 1,
        "type": "player",
        "x": 250.5,
        "y": 300.0,
        "health": 100,
        "maxHealth": 100
      },
      {
        "id": 3,
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
      "1": {"id": 1, "name": "Player1", "money": 125.5},
      "2": {"id": 2, "name": "Player2", "money": 80.0}
    }
  }
}
```

**Fields:**
- `tick` - server tick number for this snapshot
- `baselineTick` - reference tick for delta compression (0 = full snapshot)
- `entities[]` - all entities in the world
- `players{}` - player metadata (money, name, etc.)

**Delta Compression (Framework Only - Not Implemented):**
- `baselineTick` always 0 (full snapshot)
- Future: client sends ACK with last received tick
- Server uses ACK as baseline, sends only changed entities
- Falls back to full snapshot if baseline too old

### 5. Ping/Pong (Heartbeat)

**Client → Server:**
```json
{"type": "ping", "data": {}}
```

**Server → Client:**
```json
{"type": "pong", "data": {}}
```

**Purpose:** Keep connection alive, detect disconnections

---

## Tick Synchronization

### Server Tick
- Authoritative server time
- Increments at fixed rate (20 Hz default)
- All game state changes happen on tick boundaries

### Client Tick Estimation
- Client maintains estimate of current server tick
- Updated from snapshot tick + elapsed time
- Used for prediction and input timestamping

**Formula:**
```
estimatedServerTick = lastSnapshotTick + (timeSinceSnapshot * tickRate)
```

---

## Input Processing

### Server-Side Processing Order

1. **Enqueue** - Network thread receives input, adds to global queue
2. **Sort** - Queue sorted by tick (process in time order, not arrival order)
3. **Dequeue** - Tick loop dequeues inputs for current tick
4. **Deduplicate** - Skip commands with sequence ≤ client's lastProcessedSequence
5. **Validate** - Check bounds, collision, resources
6. **Apply** - Modify game state
7. **Snapshot** - Include changes in next snapshot

**Why Tick Order Matters:**
- Fair processing across all clients
- Deterministic simulation
- No early-mover advantage
- Player 1's tick 100 processed before Player 2's tick 101, regardless of arrival order

### Input Queue Structure

```go
type InputCommand struct {
    ClientId uint32
    Sequence uint32
    Tick     uint64
    Commands []Command
}

// Global queue, sorted by Tick ascending
var inputQueue []InputCommand
```

---

## Client-Side Prediction & Reconciliation

### Prediction

Client applies inputs immediately without waiting for server:

1. **Capture Input** - arrow keys pressed
2. **Generate Command** - create move/build/attack command
3. **Apply Locally** - update predicted position/state
4. **Store Command** - add to history buffer
5. **Send to Server** - include in next input message (with redundancy)

### Reconciliation

When snapshot arrives:

1. **Find Misprediction** - compare predicted state with snapshot
2. **Rollback** - revert to authoritative server state
3. **Replay Inputs** - re-apply unacknowledged commands
4. **Update Display** - smooth correction via interpolation

**Example: Movement**
```
Client predicts position: (100, 100)
Snapshot arrives: position = (98, 99)
Error = 2 pixels
→ Smooth lerp to correct position over next few frames
```

**Example: Build**
```
Client predicts: building placed, money -= 50
Snapshot arrives: no new building, money unchanged
→ Infer failure (not enough money / collision / out of bounds)
→ Client re-checks validation rules to show error message
→ Remove predicted building
```

---

## State Validation & Prediction

### Server Validation Rules

**Building Placement:**
- Money >= BuildingCost
- Position within arena bounds
- No collision with existing buildings (AABB check)

**Attack:**
- Target entity exists
- Target not owned by attacker
- Target is attackable type (generator, not player)

### Client Prediction Rules

**Clients duplicate server validation for prediction:**
- Check money before predicting build
- Check collision before showing ghost building
- Check target ownership before predicting attack

**Prediction Outcomes:**
- ✅ Success: predicted state matches snapshot → seamless
- ❌ Failure: mismatch → remove predicted change, show error

---

## Packet Loss Handling

### Input Loss
- **Mitigation:** Client sends last N=3 commands
- **Recovery:** Server processes redundant commands, skips duplicates
- **Example:** Packet 1 (seq 10) lost, Packet 2 (seq 11, 10, 9) arrives → server processes seq 10 from Packet 2

### Snapshot Loss
- **Mitigation:** Snapshots sent at 20 Hz, missing one is okay
- **Interpolation:** Client interpolates between last good snapshot and next
- **Recovery:** Next snapshot arrives, client reconciles

---

## Future Optimizations

### Delta Compression (Framework In Place)

**Baseline Tracking:**
```go
type Client struct {
    LastAckTick uint64  // Last snapshot tick client acknowledged
}
```

**Delta Snapshot:**
- Client sends ACK: "I have snapshot at tick 1950"
- Server sends only entities changed since tick 1950
- Includes full snapshot every N ticks as fallback

**Implementation Deferred:**
- Structure exists in code
- `baselineTick` field in snapshot
- `createSnapshotForClient(client)` function with TODO comments
- Always returns full snapshot for now

### Lag Compensation
- Server rewinds time to client's command timestamp
- Hit detection at past state
- Prevents "shoot behind the player" issues

### Interest Management
- Only send entities near player
- Spatial partitioning (quadtree)
- Reduces bandwidth for large maps

---

## Error Handling

### Connection Timeout
- Server tracks `lastSeen` per client
- Timeout after 10 seconds of no ping/input
- Client removed from game, entities destroyed

### Invalid Commands
- Server silently ignores malformed/invalid commands
- No error response (prevents spoofing)
- Client detects failure via snapshot mismatch

### Version Mismatch
- Client sends version in Hello
- Server rejects if incompatible (no response)
- Future: version negotiation

---

## Constants & Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| TickRate | 20 Hz | Server simulation rate |
| SnapshotRate | 20 Hz | Snapshot broadcast rate (same as tick) |
| InputRedundancy | 3 | Commands per input message |
| ClientTimeout | 10 sec | Disconnect if no ping/input |
| HeartbeatInterval | 2 sec | Ping frequency |
| ArenaWidth | 800 | Map width |
| ArenaHeight | 600 | Map height |
| PlayerSpeed | 200 units/sec | Movement speed |
| BuildingCost | 50 | Generator cost |
| BuildingSize | 40x40 | Generator dimensions |
| GeneratorIncome | 10/sec | Money generation rate |
| AttackDamage | 25 | Damage per attack |

---

## Message Size Budget

**Target:** <10 KB/sec per client at 20 Hz

**Input Message:** ~200 bytes (3 redundant commands)
**Snapshot Message:** ~500 bytes (10 entities, 4 players)
**Heartbeat:** ~50 bytes

**Total:** ~750 bytes/sec = 6 KB/sec ✅

---

## Security Considerations

1. **Server Authoritative** - never trust client state
2. **Input Validation** - bounds, collision, resources checked server-side
3. **Rate Limiting** - TODO: limit commands per second per client
4. **Sequence Validation** - TODO: detect sequence number manipulation
5. **No Sensitive Data** - don't send data about hidden/fog-of-war entities

---

## Testing & Debugging

### Simulation Tools
- Packet loss simulator (drop N% of packets)
- Latency simulator (delay packets by X ms)
- Clock drift tester

### Debug Visualization
- Show predicted vs authoritative position
- Highlight reconciliation corrections
- Display packet loss / latency metrics

### Logging
- Server: log all input processing (tick, client, sequence, command)
- Client: log predictions and reconciliations
- Network: log packet send/receive with timestamps

---

## References

- [Quake 3 Networking Model](https://fabiensanglard.net/quake3/network.php)
- [Valve Source Multiplayer Networking](https://developer.valvesoftware.com/wiki/Source_Multiplayer_Networking)
- [Gaffer On Games - Networked Physics](https://gafferongames.com/post/introduction_to_networked_physics/)
