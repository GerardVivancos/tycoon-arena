# Current Project State - Quick Reference

**Last Updated:** 2025-10-13
**Sprint:** Pathfinding & Testing - ✅ Complete
**Previous:** Map System (Phases 1-3) - ✅ Complete

---

## TL;DR - What Works Right Now

**Multiplayer RTS Game with:**
- ✅ 5 workers per player, multi-unit selection and control
- ✅ **A* pathfinding** - Units navigate around obstacles intelligently
- ✅ **Dynamic collision avoidance** - Units wait/reroute when blocked
- ✅ Tile-based movement with 3 formation types (Box, Line, Spread)
- ✅ Isometric rendering with terrain visualization
- ✅ Drag-to-select box selection
- ✅ Building placement (generators that produce $10/sec)
- ✅ Combat system (attack enemy buildings)
- ✅ Server-authoritative networking (UDP, 20Hz tick rate)
- ✅ Client-side prediction and interpolation
- ✅ 40×30 tile maps with terrain (grass, rocks, obstacles)
- ✅ Camera zoom (0.5× to 2.0×) and pan (WASD/arrows/trackpad)
- ✅ **Unit tests** - 5 passing tests for pathfinding and game logic
- ✅ **Visual test framework** - JSON scenarios → SVG diagrams

**Current Map:** 40×30 tiles with 7 rock obstacles
**Testing:** Declarative scenario framework with visual output
**Next Feature:** Scenario runner, win conditions, or visual editor

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
- **Mouse wheel / Trackpad scroll**: Zoom in/out
- **WASD / Arrow keys**: Pan camera

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
- **Visuals**: 3D-style buildings, units with shadows, terrain tiles

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

### 7. Map System ✅
- **File-based maps**: JSON format in `maps/` directory
- **Dynamic size**: Server sends dimensions to client (currently 40×30)
- **Terrain types**: Grass (passable), Rock (blocks movement/building)
- **Server validation**: `isTilePassable()` checks terrain + buildings
- **Spawn points**: Team-based spawn locations defined in map file

### 8. Terrain Rendering ✅
- **Visual tiles**: 1200 Polygon2D nodes (40×30 tiles)
- **Grass background**: Green (0.2, 0.8, 0.2)
- **Rock obstacles**: Gray (0.5, 0.5, 0.5) - 7 rocks in default map
- **Z-indexing**: Terrain below entities, height-based ordering
- **Metadata**: Height and type stored for future occlusion

### 9. Camera System ✅
- **Viewport**: 1280×720 (resizable)
- **Zoom**: Mouse wheel / trackpad (0.5× to 2.0×)
- **Pan**: WASD / Arrow keys (500 px/sec)
- **Bounds**: Dynamic based on map size with 20% edge padding
- **Zoom-aware**: Boundaries adjust for current zoom level

### 10. Pathfinding System ✅
- **Algorithm**: A* with Manhattan distance heuristic
- **Movement**: 4-directional (N, E, S, W)
- **Avoids**: Terrain obstacles, buildings, other unit destinations
- **Dynamic collision**: Units pause when next waypoint occupied
- **Rerouting**: Automatic after 1 second of blocking
- **Path following**: Waypoint-by-waypoint, smooth interpolation
- **Server-side only**: Client receives current+next tile, not full path

### 11. Testing Framework ✅
- **Unit tests**: 5 tests covering pathfinding, formations, collisions
- **Test maps**: 3 specialized maps (single rock, cluster, corridor)
- **Declarative scenarios**: JSON-based test definitions
- **Visual output**: Automatic SVG diagram generation
- **CLI tool**: `scenario-viz` generates visuals from JSON
- **Example scenarios**: 2 working examples with SVG output

---

## File Structure

```
realtime-game-engine/
├── server/
│   ├── main.go              # Entire server (1400+ lines with pathfinding)
│   ├── game_test.go         # Unit tests (5 tests)
│   ├── testutil/
│   │   ├── scenario.go           # Scenario schema + loader
│   │   ├── scenario_renderer.go  # SVG generation
│   │   ├── test_server.go        # Test utilities
│   │   └── assertions.go         # Test assertions
│   ├── cmd/
│   │   └── scenario-viz/
│   │       └── main.go      # CLI visualization tool
│   └── go.mod
├── client/
│   ├── project.godot        # Godot project config
│   ├── Main.tscn            # Main scene
│   ├── Player.tscn          # Worker unit prefab
│   ├── GameController.gd    # Main game logic (880+ lines)
│   ├── NetworkManager.gd    # UDP networking
│   └── Player.gd            # Unit visuals & interpolation
├── maps/
│   ├── default.json         # 40×30 main map
│   ├── test_single_rock.json    # Test map
│   ├── test_rock_cluster.json   # Test map
│   ├── test_corridor.json       # Test map
│   └── scenarios/
│       ├── navigate_around_rock.json
│       ├── formation_around_cluster.json
│       └── visuals/
│           ├── navigate_around_rock.svg
│           └── formation_around_cluster.svg
├── Claude.md               # Project overview & instructions
├── launch_all.sh           # Multi-client test script
└── .claude/docs/
    ├── CURRENT_STATE.md         # This file
    ├── ARCHITECTURE.md          # Detailed architecture
    ├── NETWORK_PROTOCOL.md      # Protocol specification
    ├── PATHFINDING_IMPLEMENTATION.md  # Pathfinding docs
    ├── TEST_FRAMEWORK.md        # Testing framework docs
    ├── PATHFINDING_PLAN.md      # Original pathfinding plan
    ├── sprints/
    │   ├── SPRINT_1_COMPLETE.md
    │   ├── SPRINT_2_COMPLETE.md
    │   ├── SPRINT_3_PROGRESS.md
    │   └── MAP_SYSTEM_PHASES_1-3_COMPLETE.md
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
1. **Formation edge cases**: Units can still pile up slightly at map edges when blocked
2. **Single terrain layer**: No multi-tile features (forests, mountains) yet
3. **No win conditions**: Game continues indefinitely
4. **No fog of war**: All terrain visible at all times
5. **4-directional movement only**: No diagonal pathfinding yet

### Known Quirks
1. **Client IDs skip numbers**: Due to shared ID counter (cosmetic only)
2. **Floor() for tiles**: Using `floor()` not `round()` for consistent tile ownership
3. **Direct node references**: Selection rings use stored references (not `has_node()`)
4. **Terrain overlap**: Rocks rendered as same tile size (no visual "tallness" beyond z-index)

---

## Recently Completed

### Map System (Phases 1-3) ✅
- ✅ 40×30 tile maps (expandable to 80×60 or larger)
- ✅ Terrain rendering (grass, rocks)
- ✅ Map file format (JSON)
- ✅ Passability system (terrain + buildings)
- ✅ Camera zoom and pan
- ✅ Dynamic camera boundaries with zoom awareness

### Sprint 3 (RTS Controls) ✅
- ✅ Multi-unit selection (5 workers per player)
- ✅ Formation system (Box, Line, Spread)
- ✅ Drag-to-select
- ✅ Isometric rendering

## Next Steps

### Potential Features
- [ ] Win conditions (resource threshold, building destruction, etc.)
- [ ] Different unit types (ranged, melee, fast scouts)
- [ ] Pathfinding around obstacles (A* algorithm)
- [ ] Multi-tile terrain features (forests 3×3, mountains 5×5)
- [ ] Fog of war / line of sight
- [ ] Minimap
- [ ] More building types (barracks, towers, walls)
- [ ] Unit production buildings
- [ ] Occlusion/transparency for tall objects

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
