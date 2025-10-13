# Map System Phases 1-3 - Complete

**Status:** ✅ Complete
**Date:** 2025-10-13
**Phases:** 1 (Foundation), 2 (Camera), 3 (Terrain Rendering)

---

## Overview

Implemented a complete map system with file-based maps, dynamic sizing, terrain passability, camera controls, and visual terrain rendering. The game now supports 40×30 tile maps (expandable) with obstacles that both block movement server-side and are visually rendered client-side.

---

## Phase 1: Core Map Format & Loading ✅

### Server Implementation

**Map Data Structures (`server/main.go`):**
```go
type MapData struct {
    Width          int
    Height         int
    TileSize       int
    DefaultTerrain TerrainType
    Tiles          map[TileCoord]TerrainType  // Sparse map
    Features       []Feature
    SpawnPoints    []SpawnPoint
}

type TerrainType struct {
    Type     string
    Passable bool
    Height   float32
    Visual   string
}
```

**Key Functions:**
- `LoadMap(filepath string)` - Parses JSON map files from `maps/` directory
- `isTilePassable(tileX, tileY int)` - Validates terrain passability
  - Checks map bounds
  - Checks terrain passability (sparse tile map)
  - Checks multi-tile features
  - Checks building occupation
- `getSpawnPosition(teamId int)` - Finds passable tiles near spawn points

**Map File (`maps/default.json`):**
- 40×30 tile arena (60% larger than previous 25×18)
- Grass default terrain (passable)
- 7 rock obstacles (impassable)
- 2 team spawn points with 3-tile radius

### Server Changes
- All hardcoded `ArenaTilesWidth`/`ArenaTilesHeight` replaced with `s.mapData.Width`/`Height`
- Bounds checking updated throughout (formations, movement, building)
- Spawn logic uses map-defined spawn points

### Client Adaptation
**No client changes needed!** Client already received dimensions dynamically via welcome message and adapted automatically.

---

## Phase 2: Camera Controls ✅

### Window Configuration (`client/project.godot`)
```ini
[display]
window/size/viewport_width=1280
window/size/viewport_height=720
window/size/resizable=true

[rendering]
textures/canvas_textures/default_texture_filter=1
```

### Camera System (`client/GameController.gd`)

**Variables:**
```gdscript
@onready var camera = $Camera2D
var camera_zoom_min: float = 0.5
var camera_zoom_max: float = 2.0
var camera_zoom_step: float = 0.1
var camera_pan_speed: float = 500.0
var camera_bounds: Rect2
```

**Zoom Controls:**
- Mouse wheel / trackpad scroll: zoom in/out
- Range: 0.5× to 2.0×
- Step: 0.1 per scroll

**Pan Controls:**
- WASD / Arrow keys: continuous panning (500 px/sec)
- Pan speed adjusts for zoom level

**Boundary System:**
- Calculates isometric diamond corners (north, east, south, west)
- Accounts for actual viewport size (handles resizing)
- 20% edge padding (keeps map edge at least 20% from window edge)
- Zoom-aware: boundaries expand when zoomed in, contract when zoomed out

**Key Functions:**
```gdscript
func zoom_camera(delta_zoom: float)
func pan_camera(offset: Vector2)
```

---

## Phase 3: Terrain Rendering ✅

### Server Changes

**Extended WelcomeMessage (`server/main.go`):**
```go
type WelcomeMessage struct {
    ClientId          uint32
    TickRate          int
    // ... existing fields ...
    TerrainData       TerrainData  // NEW
}

type TerrainData struct {
    DefaultType string
    Tiles       []TerrainTile
}

type TerrainTile struct {
    X      int
    Y      int
    Type   string
    Height float32
}
```

**Welcome Message Population:**
- Converts sparse `mapData.Tiles` map to array format
- Sends default terrain type ("grass")
- Sends 7 rock tiles with positions and heights

### Client Implementation

**Scene Updates (`client/Main.tscn`):**
- Added `TerrainLayer` Node2D (renders below entities, above background)

**NetworkManager Updates (`client/NetworkManager.gd`):**
- Signal updated: `connected_to_server(..., terrain_data: Dictionary)`
- Receives and passes terrain data from welcome message

**Terrain Rendering (`client/GameController.gd`):**

**Main Functions:**
```gdscript
func render_terrain(terrain_data: Dictionary):
    # Clear existing terrain
    # Render all tiles with default type (grass)
    # Override specific tiles (rocks)

func create_terrain_tile(tile_x: int, tile_y: int, type: String, height: float):
    # Create Polygon2D with diamond shape
    # Position at isometric coordinates
    # Apply color based on terrain type
    # Set z-index based on height
    # Store metadata for occlusion (future)
```

**Diamond Shape:**
```gdscript
PackedVector2Array([
    Vector2(0, ISO_TILE_HEIGHT / 2),              # Left
    Vector2(ISO_TILE_WIDTH / 2, 0),               # Top
    Vector2(ISO_TILE_WIDTH, ISO_TILE_HEIGHT / 2), # Right
    Vector2(ISO_TILE_WIDTH / 2, ISO_TILE_HEIGHT)  # Bottom
])
```

**Terrain Colors:**
- Grass: `Color(0.2, 0.8, 0.2)` - Medium green
- Rock: `Color(0.5, 0.5, 0.5)` - Gray
- Dirt: `Color(0.6, 0.4, 0.2)` - Brown
- Water: `Color(0.2, 0.4, 0.9)` - Blue
- Tree: `Color(0.1, 0.6, 0.1)` - Dark green

**Z-Indexing:**
- Terrain: `z_index = int(height * -10) - 100` (below entities)
- Higher terrain (rocks) drawn later (appears on top of grass)
- Entities remain above all terrain

---

## Visual Results

### Map Appearance
- **1200 green diamond tiles** forming grass background (40×30)
- **7 gray diamond tiles** as rock obstacles
- Grid lines overlay terrain (green, 0.3 alpha)
- Red dot at origin tile (0,0)

### Rock Positions
Default map rocks at:
1. (15, 10) - (16, 10): 2-rock cluster
2. (25, 15) - (26, 15): 2-rock cluster
3. (20, 20): Single rock
4. (10, 18): Single rock
5. (30, 12): Single rock

### Gameplay Integration
- Rocks **block movement** (server validates with `isTilePassable()`)
- Rocks **block building** (collision detection)
- Rocks **visually visible** (gray tiles)
- Formation system **avoids rocks** (skips impassable tiles)

---

## Technical Details

### Performance
- **Server:** Map loads once at startup (~1ms for 40×30 map)
- **Client:** 1200 Polygon2D nodes created on connect (~50ms)
- **Runtime:** No per-frame terrain updates needed
- **Memory:** Minimal (sparse storage server-side, static nodes client-side)

### Network Bandwidth
- Terrain data sent once in welcome message
- 7 rocks × ~20 bytes = ~140 bytes overhead
- Negligible compared to snapshot updates

### Code Locations

**Server:**
- Map structs: `server/main.go:138-202`
- LoadMap: `server/main.go:234-279`
- isTilePassable: `server/main.go:890-924`
- getSpawnPosition: `server/main.go:857-888`
- Welcome message: `server/main.go:527-551`

**Client:**
- Terrain layer: `client/Main.tscn:15`
- render_terrain: `client/GameController.gd:124-143`
- create_terrain_tile: `client/GameController.gd:145-183`
- Camera zoom: `client/GameController.gd:194-200`
- Camera pan: `client/GameController.gd:202-230`

---

## Testing Results

### Map System
- ✅ Server loads `maps/default.json` successfully
- ✅ 40×30 dimensions sent to client
- ✅ Terrain passability validation works (rocks block units)
- ✅ Spawn points work for both teams

### Camera System
- ✅ Zoom in/out with mouse wheel and trackpad
- ✅ Pan with WASD and arrow keys
- ✅ Boundaries keep map visible (20% padding)
- ✅ Zoom-aware boundaries work correctly
- ✅ Window resizing handled properly

### Terrain Rendering
- ✅ All 1200 grass tiles render
- ✅ 7 rocks render at correct positions
- ✅ Colors match specification
- ✅ Z-indexing correct (terrain below entities)
- ✅ Grid lines visible over terrain
- ✅ No performance issues

### Integration
- ✅ Units spawn at correct locations
- ✅ Units blocked by rocks
- ✅ Formations avoid rocks
- ✅ Buildings can't be placed on rocks
- ✅ All existing features still work

---

## Known Issues

1. **Rocks appear flat**: Same visual size as grass tiles, only z-index differs
   - Future: Phase 4 could add multi-tile features with larger visuals

2. **No pathfinding**: Units walk directly to target, stop at obstacles
   - Future: A* pathfinding around obstacles

3. **Static terrain**: No animated water, growing trees, etc.
   - Acceptable for current phase

4. **Single layer**: No multi-height terrain (cliffs, bridges)
   - Acceptable for current phase

---

## Next Steps (Phase 4+)

### Immediate Enhancements
- [ ] Multi-tile features (forests 3×3, mountains 5×5, lakes 4×4)
- [ ] Richer visuals (3D effect for tall rocks, textured terrain)
- [ ] More terrain types (sand, snow, lava)

### Future Features
- [ ] Occlusion/transparency (Phase 5 - see MAP_SYSTEM.md)
- [ ] Destructible terrain
- [ ] Dynamic terrain (growing vegetation, seasons)
- [ ] Height-based gameplay (high ground bonus)
- [ ] Procedural map generation

---

## Lessons Learned

### 1. Sparse Storage is Efficient
Storing only non-default tiles in a map reduces memory and serialization overhead. 1200 tiles × 0 bytes (default) + 7 tiles × 20 bytes = 140 bytes vs. 1200 × 20 bytes = 24KB.

### 2. Dynamic Viewport Sizing is Important
Using `get_viewport().get_visible_rect().size` instead of hardcoded dimensions ensures the camera system works with:
- Window resizing
- Different monitor resolutions
- Fullscreen mode

### 3. Isometric Boundary Calculation Requires All Corners
Initial implementation only used 2 corners (north, south), causing incorrect horizontal bounds. The diamond shape requires all 4 corners (N, E, S, W) to calculate the true bounding box.

### 4. Terrain Rendering is Cheap
1200 Polygon2D nodes render without performance issues on modern hardware. Godot's 2D renderer is well-optimized for this use case.

### 5. Server Authority Works
Terrain passability validated server-side prevents client hacks. Client rendering is purely cosmetic.

---

## Summary

Phases 1-3 of the map system successfully implemented:
- **Phase 1**: File-based maps with terrain passability (server-side)
- **Phase 2**: Camera zoom and pan controls (client-side)
- **Phase 3**: Visual terrain rendering (client-side)

The game now supports larger maps with visible obstacles, proper camera controls for navigation, and a solid foundation for future enhancements like multi-tile features, occlusion, and procedural generation.

**Key Achievement:** Seamless integration with existing systems. All RTS controls, formations, building, and combat still work perfectly with the new map system.
