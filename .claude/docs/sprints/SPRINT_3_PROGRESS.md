# Sprint 3 Progress - RTS Controls & Formation System

**Status:** 🚧 In Progress
**Started:** 2025-10-13
**Focus:** Multi-unit RTS controls, formations, isometric rendering

## Overview

Sprint 3 is transforming the game from a single-character control system to a full RTS-style multi-unit control system with formation movement, drag-to-select, and isometric rendering.

---

## Accomplishments So Far

### 1. Multi-Unit RTS System ✅

**Transitioned from single player entity to multiple worker units:**
- **5 workers per player** (increased from 3 for better formation visibility)
- **Array-based selection**: `selected_units: Array[int]` replaces single entity ID
- **Multi-unit commands**: All commands accept `unitIds[]` array
- **Server validation**: Ownership and unit type checking for each unit

**Protocol Changes:**
```json
{
  "type": "move",
  "data": {
    "unitIds": [10, 11, 12],  // Multiple unit IDs
    "targetTileX": 15,
    "targetTileY": 8,
    "formation": "box"
  }
}
```

**Server Implementation:**
- `Client.OwnedUnits []uint32` tracks all units per player
- Commands validate each unit ID for ownership
- Spawns workers in horizontal line at spawn point

**Client Implementation:**
- Left-click to select owned units
- Right-click to move selected units
- Visual feedback with selection rings

---

### 2. Isometric Rendering ✅

**Visual transformation to isometric (diamond) projection:**

**Key Functions:**
```gdscript
func tile_to_iso(tile_x: float, tile_y: float) -> Vector2:
    var iso_x = (tile_x - tile_y) * (ISO_TILE_WIDTH / 2.0)
    var iso_y = (tile_x + tile_y) * (ISO_TILE_HEIGHT / 2.0)
    return Vector2(iso_x + ISO_OFFSET_X, iso_y + ISO_OFFSET_Y)

func iso_to_tile(screen_pos: Vector2) -> Vector2i:
    # Inverse projection with floor() for consistent tile ownership
    return Vector2i(int(floor(tile_x)), int(floor(tile_y)))
```

**Constants:**
- `ISO_TILE_WIDTH = 64` (diamond width)
- `ISO_TILE_HEIGHT = 32` (diamond height)
- `ISO_OFFSET_X = 400`, `ISO_OFFSET_Y = 100` (screen centering)

**Rendering:**
- Diamond grid overlay (green lines, 0.3 alpha)
- Red dot at tile (0,0) for reference
- Buildings: 3D box with top face + 2 sides (shaded)
- Units: Circle body with oval shadow

**Important Fixes:**
- Used `get_local_mouse_position()` instead of `event.position` (UI offset issue)
- Used `floor()` instead of `round()` in `iso_to_tile()` for consistent tile boundaries
- Server stays tile-based (no changes needed)

---

### 3. Drag-to-Select System ✅

**RTS-style box selection for multiple units:**

**State Tracking:**
```gdscript
var is_dragging: bool = false
var drag_start_pos: Vector2
var drag_current_pos: Vector2
const DRAG_THRESHOLD: float = 5.0  # pixels
```

**Behavior:**
- **Small drag** (< 5px): Single-click selection
- **Large drag**: Box selection of all owned units in rectangle
- **Visual feedback**: Green semi-transparent selection box while dragging
- **Click empty space**: Deselects all

**Implementation:**
- `_unhandled_input()` tracks mouse down/motion/up
- `make_rect()` creates normalized rectangle (handles drag in any direction)
- `get_entities_in_rect()` finds units within selection box
- `queue_redraw()` updates selection box visual

---

### 4. Formation System ✅

**User-controllable formation patterns (AoE II style):**

#### Server-Side (Deterministic)

**Formation Types:**
- **Box**: Grid pattern (√n × √n arrangement)
- **Line**: Horizontal line
- **Spread**: Spiral from center point

**Key Functions:**
```go
func (s *GameServer) calculateFormation(formation string, centerX, centerY, numUnits int) []TilePosition

func (s *GameServer) calculateBoxFormation(centerX, centerY, numUnits int) []TilePosition {
    gridSize := int(math.Ceil(math.Sqrt(float64(numUnits))))
    // Creates centered grid...
}
```

**Features:**
- Sorts unit IDs for determinism
- Validates bounds and building collision for each position
- Falls back to center tile if insufficient valid positions

#### Client-Side

**Formation Selection:**
- **UI Buttons**: "Box (1)", "Line (2)", "Spread (3)"
- **Hotkeys**: Keys 1, 2, 3
- **Visual indicator**: `►` prefix on active formation button
- **Formation label**: Shows current formation name

**Re-form Feature:**
- When formation changes with units selected
- Calculates center position of selected units
- If all units within 5 tiles of center
- Automatically sends move command to center with new formation
- Allows quick formation changes during gameplay

**Protocol Addition:**
```go
type MoveCommand struct {
    UnitIds     []uint32 `json:"unitIds"`
    TargetTileX int      `json:"targetTileX"`
    TargetTileY int      `json:"targetTileY"`
    Formation   string   `json:"formation"`  // NEW
}
```

---

### 5. Enhanced Selection Visual ✅

**Much more visible unit selection:**

**Double-Ring Design:**
```gdscript
# Outer ring (dark, 20px radius)
outer_selection_ring.color = Color(0, 0, 0, 0.8)
outer_selection_ring.z_index = 0

# Inner ring (bright yellow, 18px radius)
selection_ring.color = Color(1, 1, 0, 1.0)  # Fully opaque
selection_ring.z_index = 1
```

**Technical Fix:**
- Added member variables for direct node references
- Changed from `has_node()` + `$NodeName` to stored references
- Ensures rings are always accessible and properly show/hide

**Result:** Clear, high-contrast selection indicator visible on all backgrounds

---

### 6. UI/UX Improvements ✅

**Event Log Auto-scroll:**
```gdscript
func log_event(message: String):
    # ... update text ...
    event_log.scroll_to_line(event_log.get_line_count() - 1)
```

**Formation UI:**
- Formation label showing current mode
- Three formation buttons with visual feedback
- Active button shows `►` prefix
- Positioned below attack button

**Hotkeys:**
- `1`: Box formation
- `2`: Line formation
- `3`: Spread formation
- `Q`: Attack selected target (existing)

---

## Technical Details

### Files Changed

**Server (`server/main.go`):**
- Added `Formation` field to `MoveCommand` struct
- Implemented formation calculation functions:
  - `calculateFormation()` - dispatcher
  - `calculateBoxFormation()` - grid pattern
  - `calculateLineFormation()` - horizontal line
  - `calculateSpiralFormation()` - spiral from center
- Updated `handleMoveCommand()` to use formations
- Changed spawn from 3 to 5 workers
- Removed obsolete "player" entity type checks

**Client (`client/GameController.gd`):**
- Added formation system variables and UI references
- Implemented `set_formation()` with re-form logic
- Added drag-to-select state and handlers
- Implemented `make_rect()` and `get_entities_in_rect()`
- Updated move command to include formation
- Modified `_unhandled_input()` for hotkeys and drag selection
- Added auto-scroll to `log_event()`

**Client (`client/Player.gd`):**
- Added direct references for selection rings
- Implemented double-ring selection visual
- Updated `set_selected()` to use direct references
- Fixed `create_isometric_sprite()` cleanup list

**Client (`client/Main.tscn`):**
- Added formation label
- Added three formation buttons
- Repositioned event log to make room

---

## Known Issues & Quirks

### 1. Formation Edge Cases
**Issue:** If formation calculation finds no valid positions (e.g., all blocked by buildings), units fall back to center tile and may overlap.

**Status:** Acceptable for MVP. Future: pathfinding to spread around obstacles.

### 2. Re-form Distance Check
**Issue:** 5-tile threshold is arbitrary and not tunable via UI.

**Status:** Works well in testing. Could add slider later if needed.

### 3. Isometric Click Precision
**Issue:** Small tiles (32px in screen space) require precise clicking, especially on overlapping units.

**Status:** Mitigated by 15px click radius and drag-to-select. Good enough for prototype.

---

## Testing Results

### Formations
- ✅ Box formation: 5 units form 3×2 grid
- ✅ Line formation: 5 units in horizontal line
- ✅ Spread formation: 5 units spiral from center
- ✅ All formations avoid buildings
- ✅ Formation changes trigger re-form when units are close

### Selection
- ✅ Single-click selection works reliably
- ✅ Drag-to-select captures multiple units
- ✅ Selection rings are highly visible (bright yellow + dark outline)
- ✅ Click empty space deselects

### Isometric Rendering
- ✅ Grid renders correctly
- ✅ Click-to-tile conversion accurate (floor() fix)
- ✅ Buildings aligned to grid
- ✅ Units move smoothly in isometric space

### UI/UX
- ✅ Formation buttons respond to clicks
- ✅ Hotkeys (1, 2, 3) switch formations
- ✅ Formation label updates correctly
- ✅ Event log auto-scrolls to latest message
- ✅ Active formation shows `►` indicator

---

## Performance

**No regressions observed:**
- Tick rate: 20 Hz (stable)
- Formation calculations: O(n) per command, negligible overhead
- Isometric projection: Simple math, no performance issues
- Selection checking: O(entities) on drag release, acceptable

---

## Next Steps

### Immediate (Complete Sprint 3)
- [ ] Add staggered formation (checkerboard pattern)
- [ ] Tune formation spacing/aesthetics
- [ ] Add formation preview on hover (stretch goal)
- [ ] Win condition implementation
- [ ] Balance pass (costs, damage, generation rates)

### Future Sprints
- Unit pathfinding around obstacles
- Different unit types (ranged, melee, workers)
- Fog of war
- Minimap
- More building types
- Sound effects and music

---

## Lessons Learned

### 1. Direct Node References vs. String Lookup
**Lesson:** Storing direct references to dynamically created nodes is more reliable than `has_node()` + `$NodeName` string lookup.

**Why:** Timing issues and scope can cause string lookup to fail. Direct references work every time.

### 2. Floor vs. Round for Tile Conversion
**Lesson:** Use `floor()` for tile coordinate conversion, not `round()`.

**Why:** Ensures consistent tile ownership (tile (x,y) owns all points where x ≤ tile_x < x+1). Prevents clicking different zones of the same visual tile producing different results.

### 3. Formation Determinism
**Lesson:** Server-side formation calculation with sorted unit IDs ensures all clients see the same result.

**Why:** Critical for authoritative server model. Client-side formation would desync.

### 4. get_local_mouse_position() vs. event.position
**Lesson:** Always use `get_local_mouse_position()` in `Node2D` input handlers.

**Why:** `event.position` is global viewport position. UI elements at top of screen offset this. Local position accounts for node transform.

---

## Code Quality

### Documentation
- ✅ In-code comments for non-obvious patterns (isometric conversion, floor() usage)
- ✅ Sprint progress documented (this file)
- 🚧 Need to update main `Claude.md` with Sprint 3 status

### Code Organization
- Clear separation: server logic (Go) vs. client rendering (GDScript)
- Formation logic entirely server-side (good!)
- UI/UX concerns entirely client-side (good!)

---

## Summary

Sprint 3 has successfully transformed the game into a proper RTS-style experience with:
- **Multi-unit control** (5 workers per player)
- **Formation movement** (Box, Line, Spread with server-side calculation)
- **Drag-to-select** (RTS-standard box selection)
- **Isometric rendering** (diamond grid visualization)
- **Enhanced UX** (bright selection visuals, auto-scroll, hotkeys)

The game now feels like a real RTS and is ready for gameplay iteration and balancing!

**Key Achievement:** Complete control system overhaul while maintaining server authority and network stability.
