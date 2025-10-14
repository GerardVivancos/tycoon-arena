# Current Project State - Quick Reference

**Last Updated:** 2025-10-14 (Evening Session)
**Current Work:** ✅ **Formation Movement Fixed** + Test Framework Overhaul
**Previous Sprint:** Formation Positioning + Movement Debugging - ✅ Complete

---

## TL;DR - What Works Right Now

**Multiplayer RTS Game with:**
- ✅ 5 workers per player, multi-unit selection and control
- ✅ **A* pathfinding** - Units navigate around obstacles intelligently
- ✅ **Friendly unit pass-through** - Teammates can pass through each other (enemies still block)
- ✅ **Direction-aware formations** - Box, Line, Spread formations orient based on movement direction
- ✅ **Formation movement** - All units pathfind to final positions independently (no bouncing)
- ✅ **Single unit optimization** - Solo units skip formation system entirely
- ✅ Isometric rendering with terrain visualization
- ✅ Drag-to-select box selection
- ✅ Building placement (generators that produce $10/sec)
- ✅ Combat system (attack enemy buildings)
- ✅ Server-authoritative networking (UDP, 20Hz tick rate)
- ✅ Client-side prediction and interpolation
- ✅ 40×30 tile maps with terrain (grass, rocks, obstacles)
- ✅ Camera zoom (0.5× to 2.0×) and pan (WASD/arrows/trackpad)
- ✅ **Comprehensive test suite** - 18/18 tests passing
  - 15 unit tests (pathfinding, formations, terrain, collisions)
  - 2 scenario tests (declarative JSON → automated execution)
  - 1 comprehensive formation test (ALL units move verification)
- ✅ **Strict test expectations** - Tests properly catch broken behavior

**Current Map:** 40×30 tiles with 7 rock obstacles
**Testing:** Full declarative test framework (Phase 1 & 2 complete)
**Next Feature:** Win conditions, more unit types, or visual scenario editor

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
- **Visual output**: Automatic SVG diagram generation (Phase 1)
- **CLI tool**: `scenario-viz` generates visuals from JSON
- **Scenario runner**: Executes scenarios in isolated test server (Phase 2)
- **Test integration**: Auto-discovers and runs all scenarios with `go test`
- **Example scenarios**: 2 working examples (both passing)

---

## File Structure

```
realtime-game-engine/
├── server/
│   ├── main.go              # Entire server (1400+ lines with pathfinding)
│   ├── game_test.go         # Unit tests (5 tests)
│   ├── scenario_test.go     # Scenario tests (2 tests) + adapter
│   ├── testutil/
│   │   ├── scenario.go           # Scenario schema + loader
│   │   ├── scenario_renderer.go  # SVG generation
│   │   ├── scenario_runner.go    # Scenario execution engine
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

## 🔄 Work In Progress

### Formation Movement (Stage 1 - Basic Implementation)

**Status:** Partially working, needs refinement for complex terrain

**What's Implemented:**
- ✅ `FormationGroup` struct tracks leader, members, offsets, destination
- ✅ Leader pathfinding to destination (closest unit becomes leader)
- ✅ Follower offset calculation (maintain formation shape relative to leader)
- ✅ `tickFormations()` updates follower positions each tick
- ✅ Formation disbands when leader reaches destination
- ✅ Helper: `isTileOccupiedByUnit()` for collision detection

**How It Works:**
1. User issues move command with formation type
2. Closest unit to click point becomes leader
3. Offsets calculated for each member (relative to leader's final position)
4. Leader pathfinds to destination
5. Each tick, followers attempt to maintain offset from leader's current position
6. Formation disbands when leader arrives

**Current Issues:**
- ⚠️ **Follower movement too simple** - One-tile-per-tick toward offset position
- ⚠️ **No follower pathfinding** - Followers can't navigate around obstacles
- ⚠️ **Followers lag on complex terrain** - Get stuck when direct path blocked
- ⚠️ **1 scenario test failing** - `formation_around_cluster` needs adjustment
- ⚠️ **No formation breaking logic** - Formation doesn't break when blocked

**Next Steps to Complete:**
1. Add follower pathfinding when direct path blocked
2. Implement formation breaking when followers can't keep up
3. Add speed synchronization (leader waits for stragglers)
4. Adjust failing scenario test expectations
5. Test with various terrain layouts

**Files Modified:**
- `server/main.go:158-168` - FormationGroup struct
- `server/main.go:243-255` - Added formations map to GameServer
- `server/main.go:405` - Call tickFormations() in game loop
- `server/main.go:752-837` - tickFormations() implementation
- `server/main.go:1458-1533` - handleMoveCommand creates formations

**Test Results:**
- 16/17 tests passing
- All 15 unit tests pass
- 1/2 scenario tests pass (formation_around_cluster fails - units don't reach expected positions)

---

## Recently Completed

### Formation Movement Fix + Test Overhaul ✅ (2025-10-14 Evening)
- ✅ **Fixed friendly unit collision** - Teammates pass through each other (enemies still block)
  - **Bug**: Units in same formation blocked each other → leader got stuck
  - **Fix**: `main.go:690-693` - Skip collision check for same `OwnerId`
- ✅ **Fixed formation disbanding** - Formations now properly detect when all units arrive
  - **Bug**: Formation stored adjusted click target instead of leader's actual destination
  - **Fix**: `main.go:1586-1598` - Use `formationPositions[0]` as formation target
- ✅ **Test framework overhaul** - Tests now catch broken behavior
  - Reverted weakened expectations (tolerance 10→4, maxTicks 300→150, allStopped false→true)
  - Added `TestAllUnitsReceivePaths` - Verifies ALL units get paths AND move (catches stuck units)
  - Debug logging added (commented out for performance)
- ✅ **18/18 tests passing** - No more false confidence from weak tests
- ✅ **No bouncing, no speed issues** - All units move smoothly to formation positions

### Formation Positioning Fix ✅ (2025-10-14 Morning)
- ✅ **Closest unit becomes tip** - Units sorted by distance to click point
- ✅ **Line extends backward** - Position[0] at click, rest extend toward origin
- ✅ **Box tip at click** - Position[0] at click point, grid extends backward
- ✅ **Age of Empires II behavior** - Matches expected RTS formation positioning
- ✅ New test: TestLineFormationBackwardExtension (2 scenarios)

### Formation Orientation Refactor ✅
- ✅ Direction-based formation orientation (8 compass directions)
- ✅ Formation tip at click point (not center)
- ✅ Line formations parallel to movement direction
- ✅ Box formations extend backward from tip
- ✅ Comprehensive tests (direction calculation, oriented formations)

### Test Framework Phase 2 ✅
- ✅ Scenario runner executes JSON scenarios in isolated server
- ✅ Test integration with `go test` (auto-discovery)
- ✅ Comprehensive expectation verification
- ✅ Constraint checking (paths, collisions, states)
- ✅ All 7 tests passing (5 unit + 2 scenario)

### Pathfinding & Testing Phase 1 ✅
- ✅ A* pathfinding with collision avoidance
- ✅ 5 unit tests for pathfinding/formations
- ✅ Visual test framework (JSON → SVG)
- ✅ CLI tool for generating scenario visuals

### Map System (Phases 1-3) ✅
- ✅ 40×30 tile maps with terrain rendering
- ✅ Camera zoom and pan with dynamic boundaries
- ✅ Server-side passability validation

## Next Steps

### High Priority
- [ ] **Advanced Formation Movement** (Optional Enhancement)
  - Current: Units pathfind independently to final positions (works well)
  - Enhancement: Maintain formation shape DURING travel (leader-follower with offsets)
  - Formation breaks on obstacles, reforms after passing
  - Speed synchronization (leader waits for stragglers)
  - See: `.claude/docs/FORMATION_MOVEMENT_PLAN.md` for detailed plan
  - Estimated: 1-2 days (Stages 3-5)
  - **Note**: Current implementation is acceptable - this is polish, not critical

### Potential Features
- [ ] Win conditions (resource threshold, building destruction, etc.)
- [ ] Different unit types (ranged, melee, fast scouts)
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
- **Formation Movement**: `.claude/docs/FORMATION_MOVEMENT_PLAN.md` (next major feature)
- **Formation Refactor**: `.claude/docs/FORMATION_REFACTOR.md` (completed)
- **Pathfinding**: `.claude/docs/PATHFINDING_IMPLEMENTATION.md`
- **Testing**: `.claude/docs/TEST_FRAMEWORK.md`
- **Sprint 3 Details**: `.claude/docs/sprints/SPRINT_3_PROGRESS.md`
- **Sprint 2 Details**: `.claude/docs/sprints/SPRINT_2_COMPLETE.md`
- **Project Overview**: `Claude.md`

---

**This document provides a snapshot of the current working state. For implementation details and historical context, consult the detailed documentation listed above.**
