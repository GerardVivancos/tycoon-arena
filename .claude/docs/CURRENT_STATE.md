# Current Project State - Quick Reference

**Last Updated:** 2025-10-13
**Sprint:** Sprint 3 (RTS Controls & Formations) - In Progress

---

## TL;DR - What Works Right Now

**Multiplayer RTS Game with:**
- ✅ 5 workers per player, multi-unit selection and control
- ✅ Tile-based movement with 3 formation types (Box, Line, Spread)
- ✅ Isometric rendering (diamond grid visualization)
- ✅ Drag-to-select box selection
- ✅ Building placement (generators that produce $10/sec)
- ✅ Combat system (attack enemy buildings)
- ✅ Server-authoritative networking (UDP, 20Hz tick rate)
- ✅ Client-side prediction and interpolation

**Current Map:** 25×18 tiles (800×576 px)
**Next Feature:** Expanding map system with terrain and camera controls

---

## Quick Start

```bash
# Start server
cd server && go run main.go

# In another terminal, start client(s)
./launch_all.sh 2  # Starts server + 2 clients

# Or manually:
/Applications/Godot_mono.app/Contents/MacOS/Godot --path client
```

**Controls:**
- **Left-click**: Select unit(s)
- **Drag-select**: Box select multiple units
- **Right-click**: Move selected units
- **1/2/3 keys**: Change formation (Box/Line/Spread)
- **Q key**: Attack selected building
- **Build button**: Place generator ($50 cost)

---

## Technology Stack

| Component | Technology | Version |
|-----------|------------|---------|
| **Server** | Go | 1.21+ |
| **Client** | Godot | 4.4.1 |
| **Protocol** | JSON over UDP | - |
| **Language** | Go + GDScript | - |

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Go Server                             │
│  - Tile-based game logic (25×18 tiles, 32px each)          │
│  - Formation calculation (Box, Line, Spread)                │
│  - Movement validation & bounds checking                     │
│  - Building placement & collision detection                  │
│  - Combat system (damage, destruction)                       │
│  - 20Hz tick rate, UDP :8080                                │
└─────────────────────────────────────────────────────────────┘
                              ▲▼ JSON Messages
┌─────────────────────────────────────────────────────────────┐
│                      Godot Client                            │
│  - Isometric rendering (tile_to_iso projection)            │
│  - Drag-to-select box selection                             │
│  - Multi-unit control with formations                        │
│  - Client-side prediction & interpolation                    │
│  - UI: Formation buttons, event log, money display          │
└─────────────────────────────────────────────────────────────┘
```

---

## Current Systems

### 1. Movement & Formation System ✅
- **Tile-based**: All positions are tile coordinates (tileX, tileY)
- **Movement speed**: 4 tiles/second
- **Server calculates**: Formation positions (deterministic)
- **Client sends**: unitIds[], targetTile, formation type
- **Formations**: Box (grid), Line (horizontal), Spread (spiral)

### 2. Multi-Unit RTS Control ✅
- **5 workers per player** (spawned at start)
- **Selection**: Single-click or drag-to-select
- **Visual feedback**: Bright yellow + dark outline double-ring
- **Commands**: Move, build, attack (all accept unit ID arrays)

### 3. Isometric Rendering ✅
- **Projection**: Square tiles → diamond grid
- **Constants**: ISO_TILE_WIDTH=64, ISO_TILE_HEIGHT=32
- **Functions**: `tile_to_iso()`, `iso_to_tile()`
- **Visuals**: 3D-style buildings, units with shadows

### 4. Building System ✅
- **Generator**: Costs $50, produces $10/sec
- **Placement**: Tile-based (2×2 footprint)
- **Validation**: Server checks money, bounds, collision
- **Rendering**: 3D box (top face + 2 sides, shaded)

### 5. Combat System ✅
- **Damage**: 25 HP per attack
- **Targeting**: Click to select enemy building, Q to attack
- **Health**: 100 HP → 4 hits to destroy
- **Validation**: Server prevents friendly fire

### 6. Networking ✅
- **Protocol**: JSON over UDP
- **Tick rate**: 20 Hz (50ms per tick)
- **Input redundancy**: Last 3 command frames per message
- **Authority**: Server is authoritative for all game state

---

## File Structure

```
realtime-game-engine/
├── server/
│   ├── main.go              # Entire server (700+ lines)
│   └── go.mod
├── client/
│   ├── project.godot        # Godot project config
│   ├── Main.tscn            # Main scene
│   ├── Player.tscn          # Worker unit prefab
│   ├── GameController.gd    # Main game logic (600+ lines)
│   ├── NetworkManager.gd    # UDP networking
│   └── Player.gd            # Unit visuals & interpolation
├── Claude.md               # Project overview & instructions
├── launch_all.sh           # Multi-client test script
└── .claude/docs/
    ├── CURRENT_STATE.md         # This file
    ├── ARCHITECTURE.md          # Detailed architecture
    ├── NETWORK_PROTOCOL.md      # Protocol specification
    ├── sprints/
    │   ├── SPRINT_1_COMPLETE.md
    │   ├── SPRINT_2_COMPLETE.md
    │   └── SPRINT_3_PROGRESS.md # Latest work
    └── README.md                # Documentation index
```

---

## Critical Code Locations

### Server (`server/main.go`)

| Function | Purpose | Line # (approx) |
|----------|---------|-----------------|
| `gameTick()` | Main game loop, processes inputs | ~240 |
| `handleMoveCommand()` | Movement with formations | ~626 |
| `calculateFormation()` | Formation algorithms | ~502 |
| `handleBuildCommand()` | Building placement | ~730 |
| `handleAttackCommand()` | Combat system | ~790 |
| `broadcastSnapshot()` | Send world state to clients | ~290 |

### Client (`client/GameController.gd`)

| Function | Purpose | Line # (approx) |
|----------|---------|-----------------|
| `tile_to_iso()` | Tile → screen projection | ~62 |
| `iso_to_tile()` | Screen → tile projection | ~71 |
| `_unhandled_input()` | Selection & move commands | ~232 |
| `set_formation()` | Formation UI & re-form logic | ~363 |
| `_on_snapshot_received()` | Server state sync | ~86 |
| `update_selection_visual()` | Selection ring management | ~349 |

---

## Network Protocol (Key Messages)

### MoveCommand (Client → Server)
```json
{
  "type": "move",
  "data": {
    "unitIds": [10, 11, 12],
    "targetTileX": 15,
    "targetTileY": 8,
    "formation": "box"  // "box", "line", or "spread"
  }
}
```

### Snapshot (Server → Client)
```json
{
  "type": "snapshot",
  "data": {
    "tick": 1234,
    "entities": [
      {
        "id": 10,
        "ownerId": 1,
        "type": "worker",
        "tileX": 12,
        "tileY": 7,
        "targetTileX": 15,
        "targetTileY": 8,
        "moveProgress": 0.65,  // 0.0 to 1.0
        "health": 100,
        "maxHealth": 100
      }
    ],
    "players": {
      "1": {"id": 1, "name": "Player123", "money": 175.5}
    }
  }
}
```

---

## Known Issues & Limitations

### Current Limitations
1. **Map size**: Fixed 25×18 tiles (expanding in next phase)
2. **No terrain**: All tiles are flat and passable
3. **No camera controls**: Fixed camera (zoom/pan coming)
4. **Formation edge cases**: Units may stack if formation blocked
5. **No pathfinding**: Units move directly to target

### Known Quirks
1. **Client IDs skip numbers**: Due to shared ID counter (cosmetic only)
2. **Floor() for tiles**: Using `floor()` not `round()` for consistent tile ownership
3. **Direct node references**: Selection rings use stored references (not `has_node()`)

---

## Next Steps (In Progress)

### Map System (Current Focus)
- [ ] Larger maps (80×60 to 120×100 tiles)
- [ ] Terrain features (rocks, water, trees)
- [ ] Multi-tile features (forests, mountains)
- [ ] Map file format (JSON)
- [ ] Passability system (terrain + buildings)
- [ ] Camera zoom and pan
- [ ] Occlusion/transparency for tall objects

### Future Features
- [ ] Win conditions
- [ ] Different unit types (ranged, melee)
- [ ] Pathfinding around obstacles
- [ ] Fog of war
- [ ] Minimap
- [ ] More building types

---

## Documentation Map

For deep dives, see:
- **Architecture**: `.claude/docs/ARCHITECTURE.md` (needs update for Sprint 3)
- **Network Protocol**: `.claude/docs/NETWORK_PROTOCOL.md`
- **Sprint 3 Details**: `.claude/docs/sprints/SPRINT_3_PROGRESS.md`
- **Sprint 2 Details**: `.claude/docs/sprints/SPRINT_2_COMPLETE.md`
- **Project Overview**: `Claude.md`

---

**This document provides a snapshot of the current working state. For implementation details and historical context, consult the detailed documentation listed above.**
