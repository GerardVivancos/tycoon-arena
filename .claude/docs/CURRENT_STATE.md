# Current Project State - Quick Reference

**Last Updated:** 2025-10-10 (Phase 5 complete)
**Current Work:** ✅ **Phase 5** of [IMPROVEMENT_PLAN.md](../../IMPROVEMENT_PLAN.md) complete — scale & authoring tools (configurable map + data-driven `BalanceConfig`; delta-compressed snapshots; ranged `soldier` unit; visual scenario editor). Phases 0-5 done.
**Next Up:** Phase 6 (deferred) — fog of war (server-authoritative per-client visibility, composes with the delta snapshot path).

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
- ✅ Combat system — units attack-move to destroy enemy **units and buildings**
  (range/cooldown; buildings non-combatant)
- ✅ Server-authoritative networking (UDP, 20Hz tick rate)
- ✅ Client-side prediction and interpolation
- ✅ 40×30 tile maps with terrain (grass, rocks, obstacles)
- ✅ Camera zoom (0.5× to 2.0×) and pan (WASD/arrows/trackpad)
- ✅ **Comprehensive test suite** - 84 test functions passing
  - 83 unit tests (pathfinding, formations, terrain, collisions, event bus,
    statistics, rule engine incl. per-player resolution, game modes incl.
    win/lose/periodic/E2E, game-mode plumbing, unit combat, concurrency,
    balance config, snapshot deltas, soldier/ranged combat)
  - Scenario harness runs 4 declarative JSON scenarios end-to-end
    (incl. `unit_combat` — a worker destroys an enemy generator)
  - `TestAllUnitsReceivePaths` ensures every unit in formation gets a path
- ✅ **Strict test expectations** - Tests properly catch broken behavior
- ✅ **Game modes wired into the loop (Phases 1-2)** - `main()` loads `resource_race`,
  `gameTick()` runs mode periodic checks + evaluates win/lose conditions each
  tick, `game_over` message + client HUD (objective/victory/defeat),
  inputs/joins frozen after match end. Rule engine resolves `player:"self"`/
  `"opponent"`/`"any"`/numeric id; winner strategies `player_who_triggered`,
  `highest_money`, `highest_stat` all work. The three shipped modes
  (`resource_race`, `survival`, `timed_race`) can now actually end a match.
- ✅ **Lose conditions live** - self-referencing rules (bankruptcy/elimination)
  evaluated per player; lone survivor wins (`last_player_standing`); fresh
  players seed `money_current` so bankruptcy never false-positives on join
- ✅ **Unit combat (Phase 3)** - `attack` command (with `unitIds`) assigns a target;
  `processCombat()` walks attackers into melee range and applies damage on a
  per-type cooldown (`unitCombatStats`); combat stats `damage_dealt`/`damage_taken`/
  `attacks_made`/`attacks_received`/`units_killed`/`units_lost` are wired

**Current Map:** Default JSON map `maps/default.json` (40×30 tiles, 7 rock obstacles)
**Testing:** Full declarative test framework (Phases 1-3 complete)
**Next Feature:** Concurrency hardening — `Emit`-outside-lock + `go test -race` (IMPROVEMENT_PLAN.md Phase 4)

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
│  - Tile-based game logic (JSON-driven, default 40×30 tiles)│
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
- **Visuals**: 3D-style buildings, units with shadows, terrain tiles (dimensions from server welcome payload)

### 4. Building System ✅
- **Generator**: Costs $50, produces $10/sec
- **Placement**: Tile-based (2×2 footprint)
- **Validation**: Server checks money, bounds, collision
- **Rendering**: 3D box (top face + 2 sides, shaded)

### 5. Combat System ✅ (unit-capable, Phase 3)
- **Damage**: worker = 25 HP per hit (`unitCombatStats`), melee range 1 tile, 20-tick cooldown
- **Targets**: enemy **units and buildings**; buildings are non-combatant (HP only)
- **Flow**: `attack` command (with `unitIds`) assigns `AttackTargetId`; `processCombat()`
  in the tick pathfinds attackers into range then applies damage on cooldown
- **Health**: 100 HP → 4 hits to destroy
- **Client**: click an enemy unit/building to target it, Q (or Attack button) to attack
  with the selected units
- **Validation**: server rejects attacking own entities and non-attackable types

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
- **Unit tests**: 83 unit test functions (core, event bus, statistics, rule engine
  incl. per-player self/opponent/any/id, game modes incl. win/lose/periodic/E2E,
  unit combat incl. chase/cooldown/stats, concurrency incl. deadlock guard,
  balance config, snapshot delta/full, soldier ranged combat)
- **Test maps**: 3 specialized maps (single rock, cluster, corridor)
- **Declarative scenarios**: JSON-based test definitions
- **Scenario runner**: Executes scenarios in isolated test server
- **Test integration**: Auto-discovers and runs all scenarios with `go test`
- **Example scenarios**: 2 working examples (both passing)

### 12. Game Event & Mode Framework ✅ (Phases 1-5 complete)
- **Event bus** (`event_bus.go`): typed game events, synchronous pub/sub. State is
  mutated under `s.mu`; events are **queued** (`queueEvent`) while the lock is held
  and **emitted after unlock** (`flushEvents()`, called by `gameTick`), so listeners
  never run under `s.mu` and may safely acquire it (Phase 4). `main.go` emits
  resource/build/attack/destroy events from the tick loop and command handlers
- **Statistics** (`statistics.go`): per-player + global counters and time series,
  wired to the event bus via `setupStatisticsTracking()`
- **Rule engine** (`rule_engine.go`): declarative comparison/logical rules parsed from
  JSON; `StatValue.Resolve` supports `player:"self"`, `"opponent"` (max excluding self),
  `"any"` (aggregate max), a numeric id, or `""` (legacy direct id) via `RuleContext`'s
  `CurrentPlayer`/`Players`
- **Game modes** (`game_mode.go`, `game_modes/*.json`): `resource_race` (default, loaded
  in `main()`), `survival`, `timed_race`; `gameTick()` → `mode.OnTick(ctx)` (periodic
  `grant_resources`/`increment_global_stat`) then `checkMatchEnd()` (both **after** unlock)
  evaluate win **and** lose conditions; winner strategies `player_who_triggered`
  (first self-rule pass over sorted players), `highest_money`, `highest_stat`
  (max-holder, ties→lowest id); `endMatch()` freezes inputs/joins, emits `match.ended`,
  and broadcasts `game_over` (handled by `NetworkManager.gd`/`GameController.gd`,
  `GameStatus` HUD label)
- **Per-player + deterministic**: rules iterate clients via `sortedClientIDs()` (ascending),
  so the same game state always produces the same winner; lose conditions crown the lone
  survivor (`last_player_standing`) and are skipped in solo matches
- **Data-driven balance** (`config.go`, `maps/balance.json`): `BalanceConfig` supplies
  unit HP/damage/range/cooldown/speed and building cost/health/income/footprint, loaded
  from JSON with Go defaults (worker/generator/soldier); `-map`/`-balance` flags choose
  files; `welcome.balance` surfaces costs/HP to the client (Phase 5)
- **Delta snapshots** (`buildSnapshotLocked`): broadcasts changed entities + `removed`
  ids, with a full snapshot on join and every `FullSnapshotInterval` ticks; clients
  reconcile full vs delta by `baselineTick` (Phase 5)

---

## File Structure

```
realtime-game-engine/
├── server/
│   ├── main.go              # Core server (2000+ lines: tick loop, networking, commands)
│   ├── event_bus.go         # Game event pub/sub
│   ├── statistics.go        # Player/global stats + event wiring
│   ├── config.go            # BalanceConfig (data-driven unit/building balance) + helpers
│   ├── rule_engine.go       # Declarative JSON rule evaluation
│   ├── game_mode.go         # JSON-defined game modes
│   ├── game_modes/          # resource_race, survival, timed_race JSON
│   ├── game_test.go         # Core unit tests (16)
│   ├── event_bus_test.go    # Event bus tests (6)
│   ├── statistics_test.go   # Statistics tests (7)
│   ├── rule_engine_test.go  # Rule engine tests (13)
│   ├── game_mode_test.go    # Game mode tests (14)
│   ├── game_mode_plumbing_test.go # Game-mode server wiring tests (5)
│   ├── combat_test.go       # Unit combat tests (7)
│   ├── concurrency_test.go  # Event-lock/deadlock + race tests (2)
│   ├── balance_test.go      # BalanceConfig load/override + build/HP (6)
│   ├── snapshot_test.go     # Snapshot full/delta/periodic/force tests (4)
│   ├── soldier_combat_test.go # Ranged soldier config-driven combat (3)
│   ├── scenario_test.go     # Scenario harness + adapter
│   ├── testutil/
│   │   ├── scenario.go           # Scenario schema + loader
│   │   ├── scenario_runner.go    # Scenario execution engine
│   │   ├── test_server.go        # Test utilities
│   │   └── assertions.go         # Test assertions
│   └── go.mod
├── client/
│   ├── project.godot        # Godot project config
│   ├── Main.tscn            # Main scene
│   ├── Player.tscn          # Worker unit prefab
│   ├── GameController.gd    # Main game logic (880+ lines)
│   ├── NetworkManager.gd    # UDP networking
│   └── Player.gd            # Unit visuals & interpolation
├── client/editor/
│   ├── Editor.gd            # Visual map + scenario editor (paint/place/export/run)
│   └── Editor.tscn          # Editor scene
├── maps/
│   ├── default.json         # 40×30 main map
│   ├── balance.json         # Data-driven unit/building/economy balance
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

| Function | Purpose | Line # |
|----------|---------|--------|
| `gameTick()` | Main game loop, processes inputs | 354 |
| `handleMoveCommand()` | Movement with formations | 1469 |
| `calculateFormation()` | Formation algorithms | 1053 |
| `handleBuildCommand()` | Building placement | 1831 |
| `handleAttackCommand()` | Combat system | 1941 |
| `broadcastMessage()` | Send world state to clients | 2034 |

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
3. **Only three shipped modes**: `resource_race`, `survival`, `timed_race` are now
   winnable/losable live (Phases 1-2), but periodic income was removed in favor of the
   core `$10/s`/generator loop; richer modes/params are future work
4. **Combat scope**: units attack-move enemy units/buildings (Phase 3), but buildings
   are non-combatant (no defensive fire) and there is no auto-acquire, ranged unit
   type, or unit-attack cooldown UI — follow-ups in IMPROVEMENT_PLAN.md
5. **No fog of war**: All terrain visible at all times
6. **4-directional movement only**: No diagonal pathfinding yet

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

**Polish Opportunities:**
- ✳️ **Follower movement smoothing** - Could adopt leader/follower offsets for in-transit cohesion
- ✳️ **Adaptive follower pathfinding** - Followers currently recompute individually when blocked
- ✳️ **Formation breaking rules** - Optional logic for reforming after obstacle detours

**Next Enhancements (Optional):**
1. Add shared path planning for followers in tight spaces
2. Implement formation breaking when followers can't keep pace
3. Synchronize speeds so leaders wait for stragglers
4. Expand scenario coverage with dense obstacle layouts

**Files Modified:**
- `server/main.go:158-168` - FormationGroup struct
- `server/main.go:243-255` - Added formations map to GameServer
- `server/main.go:405` - Call tickFormations() in game loop
- `server/main.go:752-837` - tickFormations() implementation
- `server/main.go:1458-1533` - handleMoveCommand creates formations

**Test Results:**
- All 16 unit tests passing
- Scenario harness passes `navigate_around_rock` and `formation_around_cluster`

---

## Recently Completed

### Scale & Authoring Tools ✅ (Phase 5, 2025-10-10)
- ✅ **Configurable map + data-driven balance** — `-map`/`MAP_FILE` and
  `-balance`/`BALANCE_FILE` flags (`main.go`); `BalanceConfig` in `config.go`
  loaded from `maps/balance.json` with Go defaults (worker/generator/soldier), so
  tests are unchanged without the file. Unit HP/damage/range/cooldown/speed and
  building cost/health/income/footprint now come from config; `welcome` carries a
  `balance` block (start money, build cost, unit/building HP). `handleBuildCommand`
  and the income loop read the config; `isAttackableTarget` is config-driven
- ✅ **Delta-compressed snapshots** — `SnapshotMessage.Removed` + `buildSnapshotLocked()`
  (`main.go`): deltas carry changed entities only; a full snapshot (`BaselineTick 0`)
  is sent on the first build, when a client joins (`forceFullSnapshot`), and every
  `FullSnapshotInterval` (20) ticks to self-heal dropped UDP packets; the tiny
  players map is always full. `GameController.gd` `_on_snapshot_received` is
  delta-aware (applies changed entities + explicit `removed`, `_remove_entity` helper)
- ✅ **Ranged soldier unit** — defined purely in `BalanceConfig` (120 HP / 15 dmg /
  range 3 / 15-tick cooldown); spawned via `StartingSoldiers` (0 by default so the
  default match is unchanged; `maps/balance.json` sets 1); `newUnitEntity` builds
  units from config; client renders soldiers as units
- ✅ **Visual scenario editor** — `client/editor/Editor.gd` + `Editor.tscn`: paint
  terrain, place spawn points + scenario units/buildings, record move/attack via
  click-to-target, `Save Map`/`Export Scenario` write `maps/*.json` +
  `maps/scenarios/*.json` in the exact schemas `LoadMap`/harness consume, and
  `Run Scenario` shells `go -C server test -run TestAllScenarios/<name>`
- ✅ **+13 tests (84 green)** — `balance_test.go` (6), `snapshot_test.go` (4),
  `soldier_combat_test.go` (3) + `attack_soldier` scenario (4th scenario); adapter
  `SpawnUnit`/`SpawnBuilding`/`Tick` made config-aware via `newUnitEntity`/
  `isUnitType`; `-race` clean, Godot `--check-only` clean on both scripts

### Concurrency & Event Ordering ✅ (Phase 4, 2025-10-09)
- ✅ **Events no longer emitted under `s.mu`** — added `s.eventQueue` +
  `queueEvent()` (append while the lock is held) and `flushEvents()` (drain via
  `eventBus.Emit` after the lock is released). The 9 previously in-lock `Emit`
  calls — tick resource generation, `handleBuildCommand` (money/placed/created),
  and `applyDamage` (attacked/damaged/building-destroyed/entity-destroyed) — now
  go through `queueEvent`; `gameTick()` calls `flushEvents()` right after
  `s.mu.Unlock()`, before `mode.OnTick`/`checkMatchEnd`
- ✅ **Why** — `EventBus.Emit` runs listeners synchronously; emitting under
  `s.mu` risked self-deadlock the moment a listener acquires `s.mu`, and
  serialized the tick loop on listener work. `grantMoneyToPlayer`/`endMatch`
  already emitted post-unlock; the rule ("never `Emit` under `s.mu`") is now
  documented in code
- ✅ **Regression tests** (`concurrency_test.go`) — `TestEmitOutsideLockNoDeadlock`
  (a listener that acquires `s.mu` proves the deferred path can't hang) and
  `TestConcurrentTickAndMatchEnd` (tick loop + concurrent grants + concurrent
  match-end checks hammer a live match; asserts exactly one `match.ended`)
- ✅ **+2 tests (71 green)**; `go test -race ./...` clean (was already race-free;
  now guarded). Test drivers that call `processCombat()` directly (combat tests,
  scenario adapter) updated to `flushEvents()` afterward

### Unit Combat ✅ (Phase 3, 2025-10-09)
- ✅ **Attack-move combat** (`main.go`) — `handleAttackCommand` now assigns
  `AttackTargetId` to the named `unitIds` (or the client's own units) instead of
  one-shot damage; `processCombat()` (in `gameTick`, after formations) walks
  attackers into melee range (`pathUnitToAttackRange`/`nearestAttackTile`) and
  applies damage via `applyDamage()` on a per-type cooldown
- ✅ **Combat table** — `unitCombatStats`: worker = 25 dmg / 1-tile range / 20-tick
  cooldown; `Entity` gained `AttackTargetId`/`NextAttackTick`; buildings are
  non-combatant; `handleMoveCommand` clears the attack target
- ✅ **Combat stats** (`statistics.go`) — `damage_taken` + `attacks_received` now
  land on the target's owner (carried as `targetOwnerId`); `units_killed`/`units_lost`
  fire on unit death via `entity.destroyed`
- ✅ **Protocol/client** — `attack` carries `unitIds`; client sends selected units and
  can click an enemy unit (not just buildings) to target it
- ✅ **+7 tests (69 green)** — `combat_test.go` (kill, chase, cooldown, out-of-range,
  non-combatant building, no self-attack, destroy building) + `unit_combat.json`
  scenario; test harness fixed to run combat on `Tick()`, send `unitIds`, and record
  destroyed buildings so `exists:false` verifies

### Game Modes Live + Rule Engine Per-Player Resolution ✅ (Phase 2, 2025-10-09)
- ✅ **Per-player rule resolution** (`rule_engine.go`) — `RuleContext` gained
  `CurrentPlayer`/`Players`; `StatValue` gained `PlayerRef` and resolves `"self"`,
  `"opponent"` (max excl. self), `"any"` (aggregate max), numeric id, `""` (legacy)
- ✅ **Winner strategies** (`game_mode.go`) — removed the map-iteration
  `determineWinner` stub; `CheckWinConditions` honors `player_who_triggered`
  (first self-rule pass over sorted players), `highest_money`, `highest_stat`
  (max-holder, ties→lowest id) — deterministic via `sortedClientIDs()`
- ✅ **Periodic checks** — `executePeriodicCheck` implements `grant_resources`
  (via new `grantMoneyToPlayer`/`countBuildings`) and `increment_global_stat`;
  `gameTick()` now runs `mode.OnTick(ctx)` after unlock, then `checkMatchEnd()`
- ✅ **Live lose conditions** — `checkMatchEnd` evaluates lose rules per player and
  crowns the lone survivor (`last_player_standing`); `handleHello` seeds
  `money_current` so bankruptcy never false-positives on join
- ✅ **Mode JSONs** — per-player win rules moved `"any"`→`"self"`
  (`resource_race`, `survival`); removed redundant `grant_resources` periodic
  checks from all three (core loop already pays `$10/s`/generator)
- ✅ **+6 tests (62 green)** — per-player resolution, rewritten win/survival/timed
  tests with real clients, last-player-standing, fresh-player-not-broke,
  periodic grant/increment, and `TestResourceRaceEndToEnd` (one `gameTick()` ends match)

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
- ✅ **All automated tests passing** (16 unit + 2 scenario subtests)
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
- [x] Win conditions (resource threshold, building destruction, timed highest-money) —
      shipped as JSON game modes (Phases 1-2, `game_modes/*.json`)
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
