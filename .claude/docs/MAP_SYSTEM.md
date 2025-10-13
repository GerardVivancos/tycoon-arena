# Map System Design

**Status:** Planning Phase
**Target:** Larger, feature-rich maps with terrain and camera controls
**Current Map:** 25×18 tiles (hardcoded) → Target: 80×60 to 120×100 tiles (file-based)

---

## Goals

1. **Larger maps** for expansive gameplay (4-6x current size)
2. **Terrain features** (rocks, water, forests) with visual variety
3. **Multi-tile features** (mountains spanning 3×3, forests 4×4, etc.)
4. **Passability system** (some tiles block movement/building)
5. **Zoomable camera** with pan controls
6. **Occlusion system** (tall objects become transparent near cursor)
7. **Shared format** (both server and client understand the same map files)

---

## Map Format Specification

### File Structure (JSON)

**Location:** `maps/` directory (shared or separate server/client copies)

```json
{
  "version": "1.0",
  "name": "Default Arena",
  "width": 80,
  "height": 60,
  "tileSize": 32,

  "terrain": {
    "default": {
      "type": "grass",
      "passable": true,
      "height": 0,
      "visual": "grass"
    },
    "tiles": [
      {"x": 10, "y": 10, "type": "rock", "passable": false, "height": 2},
      {"x": 15, "y": 20, "type": "water", "passable": false, "height": -1},
      {"x": 25, "y": 25, "type": "tree", "passable": false, "height": 1}
    ]
  },

  "features": [
    {
      "type": "mountain",
      "x": 40,
      "y": 30,
      "width": 3,
      "height": 3,
      "passable": false,
      "visualHeight": 3
    },
    {
      "type": "forest",
      "x": 60,
      "y": 45,
      "width": 4,
      "height": 4,
      "passable": false,
      "visualHeight": 2
    },
    {
      "type": "lake",
      "x": 20,
      "y": 40,
      "width": 5,
      "height": 5,
      "passable": false,
      "visualHeight": -1
    }
  ],

  "spawnPoints": [
    {"team": 0, "x": 10, "y": 10, "radius": 5},
    {"team": 1, "x": 70, "y": 50, "radius": 5}
  ],

  "metadata": {
    "author": "System",
    "created": "2025-10-13",
    "description": "Large arena with varied terrain"
  }
}
```

### Tile Types

| Type | Passable | Height | Visual | Notes |
|------|----------|--------|--------|-------|
| `grass` | ✅ Yes | 0 | Green flat | Default terrain |
| `dirt` | ✅ Yes | 0 | Brown flat | Alternate ground |
| `sand` | ✅ Yes | 0 | Tan flat | Beach areas |
| `rock` | ❌ No | 2 | Gray tall | Small obstacles |
| `water` | ❌ No | -1 | Blue low | Rivers, lakes |
| `tree` | ❌ No | 1 | Green tall | Single trees |
| `bush` | ❌ No | 0.5 | Green medium | Low vegetation |

### Multi-Tile Features

**Mountains:**
- Size: 3×3 to 5×5 tiles
- Height: 3 (very tall, occludes)
- Visual: Gray/brown rocky texture
- Blocks: Movement, building, line of sight

**Forests:**
- Size: 3×3 to 6×6 tiles
- Height: 2 (tall, occludes)
- Visual: Dense trees with canopy
- Blocks: Movement, building (units can hide)

**Lakes:**
- Size: 4×4 to 8×8 tiles
- Height: -1 (low, doesn't occlude)
- Visual: Blue water with waves
- Blocks: Movement, building

### Coordinate System

```
Tile (0, 0) = Top-left corner
Tile (width-1, height-1) = Bottom-right corner

┌──────────────────────► X
│  (0,0)    (1,0)    (2,0)
│
│  (0,1)    (1,1)    (2,1)
│
│  (0,2)    (1,2)    (2,2)
▼
Y
```

**Multi-tile features:**
- `x, y` = Top-left corner tile
- `width, height` = Span in tiles
- Occupies tiles from (x, y) to (x+width-1, y+height-1)

---

## Server Architecture

### Map Loading System

**File:** `server/main.go`

```go
type MapData struct {
    Width          int
    Height         int
    TileSize       int
    DefaultTerrain TerrainType
    Tiles          map[TileCoord]TerrainType  // Sparse map for non-default tiles
    Features       []Feature
    SpawnPoints    []SpawnPoint
}

type TerrainType struct {
    Type      string
    Passable  bool
    Height    float32
    Visual    string
}

type Feature struct {
    Type         string
    X, Y         int
    Width        int
    Height       int
    Passable     bool
    VisualHeight float32
}

type TileCoord struct {
    X, Y int
}

func LoadMap(filepath string) (*MapData, error) {
    // Parse JSON file
    // Validate dimensions, tile references
    // Build sparse map for efficient lookup
    // Return MapData
}
```

### Passability System

```go
func (s *GameServer) isTilePassable(tileX, tileY int) bool {
    // 1. Check bounds
    if tileX < 0 || tileX >= s.mapData.Width || tileY < 0 || tileY >= s.mapData.Height {
        return false
    }

    // 2. Check terrain
    coord := TileCoord{tileX, tileY}
    if terrain, exists := s.mapData.Tiles[coord]; exists {
        if !terrain.Passable {
            return false
        }
    }

    // 3. Check features (multi-tile)
    for _, feature := range s.mapData.Features {
        if tileX >= feature.X && tileX < feature.X+feature.Width &&
           tileY >= feature.Y && tileY < feature.Y+feature.Height {
            if !feature.Passable {
                return false
            }
        }
    }

    // 4. Check buildings (existing logic)
    if s.isTileOccupiedByBuilding(tileX, tileY) {
        return false
    }

    return true
}
```

### Spawn Point Selection

```go
func (s *GameServer) getSpawnPointForTeam(teamId int) (int, int) {
    for _, spawn := range s.mapData.SpawnPoints {
        if spawn.Team == teamId {
            // Find first passable tile near spawn point
            for attempt := 0; attempt < 100; attempt++ {
                offsetX := rand.Intn(spawn.Radius*2) - spawn.Radius
                offsetY := rand.Intn(spawn.Radius*2) - spawn.Radius
                x := spawn.X + offsetX
                y := spawn.Y + offsetY

                if s.isTilePassable(x, y) {
                    return x, y
                }
            }
        }
    }

    // Fallback to default position
    return 10, 10
}
```

### Map Data in Welcome Message

**Option A: Send full map data** (for dynamically generated maps)
```json
{
  "type": "welcome",
  "data": {
    "clientId": 1,
    "tickRate": 20,
    "mapData": { ... }  // Entire map structure
  }
}
```

**Option B: Send map filename** (if clients have map files)
```json
{
  "type": "welcome",
  "data": {
    "clientId": 1,
    "tickRate": 20,
    "mapName": "default_arena"  // Client loads from maps/default_arena.json
  }
}
```

**Recommendation:** Option B for performance, Option A for flexibility

---

## Client Architecture

### Window & Viewport

**File:** `client/project.godot`

```ini
[display]
window/size/viewport_width=1280
window/size/viewport_height=720
window/size/resizable=true
window/size/borderless=false

[rendering]
textures/canvas_textures/default_texture_filter=1  # Linear filter for smooth zoom
```

### Camera System

**File:** `client/GameController.gd`

```gdscript
@onready var camera = $Camera2D

# Camera settings
var camera_zoom_min: float = 0.5
var camera_zoom_max: float = 2.0
var camera_zoom_step: float = 0.1
var camera_pan_speed: float = 500.0  # pixels per second

# Camera bounds (set after map loads)
var camera_bounds: Rect2

func _ready():
    camera.zoom = Vector2(1.0, 1.0)

func _unhandled_input(event):
    # Mouse wheel zoom
    if event is InputEventMouseButton:
        if event.button_index == MOUSE_BUTTON_WHEEL_UP:
            zoom_camera(camera_zoom_step)
        elif event.button_index == MOUSE_BUTTON_WHEEL_DOWN:
            zoom_camera(-camera_zoom_step)

    # Middle mouse drag (pan)
    if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_MIDDLE:
        if event.pressed:
            start_camera_drag(event.position)
        else:
            stop_camera_drag()

func _process(delta):
    # WASD/Arrow key panning
    var pan_direction = Vector2.ZERO

    if Input.is_key_pressed(KEY_W) or Input.is_key_pressed(KEY_UP):
        pan_direction.y -= 1
    if Input.is_key_pressed(KEY_S) or Input.is_key_pressed(KEY_DOWN):
        pan_direction.y += 1
    if Input.is_key_pressed(KEY_A) or Input.is_key_pressed(KEY_LEFT):
        pan_direction.x -= 1
    if Input.is_key_pressed(KEY_D) or Input.is_key_pressed(KEY_RIGHT):
        pan_direction.x += 1

    if pan_direction != Vector2.ZERO:
        pan_camera(pan_direction.normalized() * camera_pan_speed * delta)

func zoom_camera(delta_zoom: float):
    var new_zoom = camera.zoom.x + delta_zoom
    new_zoom = clamp(new_zoom, camera_zoom_min, camera_zoom_max)
    camera.zoom = Vector2(new_zoom, new_zoom)

func pan_camera(offset: Vector2):
    camera.position += offset / camera.zoom.x  # Adjust for zoom level

    # Clamp to map bounds
    if camera_bounds:
        camera.position.x = clamp(camera.position.x, camera_bounds.position.x, camera_bounds.end.x)
        camera.position.y = clamp(camera.position.y, camera_bounds.position.y, camera_bounds.end.y)
```

### Terrain Rendering

**Approach:** Custom rendering with Polygon2D nodes

```gdscript
@onready var terrain_layer = $TerrainLayer

func load_map_data(map_data: Dictionary):
    var width = map_data.get("width", 25)
    var height = map_data.get("height", 18)

    # Set camera bounds
    var map_screen_size = tile_to_iso(width, height)
    camera_bounds = Rect2(Vector2.ZERO, map_screen_size)

    # Render default terrain (grass)
    for x in range(width):
        for y in range(height):
            var tile_pos = tile_to_iso(float(x), float(y))
            create_terrain_tile(tile_pos, "grass", 0)

    # Override with specific tiles
    for tile_data in map_data.get("terrain", {}).get("tiles", []):
        var x = tile_data.get("x")
        var y = tile_data.get("y")
        var type = tile_data.get("type")
        var height = tile_data.get("height", 0)

        var tile_pos = tile_to_iso(float(x), float(y))
        create_terrain_tile(tile_pos, type, height)

    # Render multi-tile features
    for feature_data in map_data.get("features", []):
        create_terrain_feature(feature_data)

func create_terrain_tile(pos: Vector2, type: String, height: float):
    var tile = Polygon2D.new()
    tile.position = pos

    # Diamond shape for isometric tile
    var points = PackedVector2Array([
        Vector2(ISO_TILE_WIDTH / 2, 0),
        Vector2(ISO_TILE_WIDTH, ISO_TILE_HEIGHT / 2),
        Vector2(ISO_TILE_WIDTH / 2, ISO_TILE_HEIGHT),
        Vector2(0, ISO_TILE_HEIGHT / 2)
    ])
    tile.polygon = points

    # Color based on type
    match type:
        "grass":
            tile.color = Color(0.2, 0.8, 0.2)
        "dirt":
            tile.color = Color(0.6, 0.4, 0.2)
        "rock":
            tile.color = Color(0.5, 0.5, 0.5)
        "water":
            tile.color = Color(0.2, 0.4, 0.9)
        "tree":
            tile.color = Color(0.1, 0.6, 0.1)

    # Z-index based on height
    tile.z_index = int(height * -10)  # Negative so higher = drawn later

    # Store height for occlusion
    tile.set_meta("height", height)

    terrain_layer.add_child(tile)

func create_terrain_feature(feature: Dictionary):
    var type = feature.get("type")
    var x = feature.get("x")
    var y = feature.get("y")
    var width = feature.get("width")
    var height = feature.get("height")
    var visual_height = feature.get("visualHeight", 0)

    # Multi-tile features span multiple tiles
    # Create a larger polygon or multiple polygons
    # Example for mountain (3×3):
    var feature_node = Node2D.new()
    feature_node.position = tile_to_iso(float(x), float(y))
    feature_node.set_meta("height", visual_height)
    feature_node.set_meta("feature_type", type)

    # Create visual (simplified - could use sprites)
    var visual_width = width * ISO_TILE_WIDTH
    var visual_height = height * ISO_TILE_HEIGHT

    # ... create polygon or sprite for feature

    terrain_layer.add_child(feature_node)
```

---

## Occlusion/Transparency System

### Design Goals
- Tall objects (mountains, forests) can block view of units/buildings
- Make objects transparent when cursor is near them
- Smooth fade-in/fade-out

### Cursor-Proximity Approach

```gdscript
const OCCLUSION_RADIUS = 150.0  # pixels
const MIN_ALPHA = 0.3  # Minimum transparency

func _process(delta):
    var mouse_pos = get_global_mouse_position()

    # Check all terrain with height > 0
    for terrain_node in terrain_layer.get_children():
        var height = terrain_node.get_meta("height", 0)

        if height > 0:  # Only tall objects occlude
            var distance = terrain_node.global_position.distance_to(mouse_pos)

            if distance < OCCLUSION_RADIUS:
                # Fade based on distance (closer = more transparent)
                var fade_factor = distance / OCCLUSION_RADIUS
                terrain_node.modulate.a = lerp(MIN_ALPHA, 1.0, fade_factor)
            else:
                # Outside radius - fully opaque
                terrain_node.modulate.a = 1.0
```

### Y-Coordinate Approach (Alternative)

```gdscript
func _process(delta):
    var mouse_pos = get_global_mouse_position()

    for terrain_node in terrain_layer.get_children():
        var height = terrain_node.get_meta("height", 0)

        if height > 0:
            # If terrain is "above" cursor in screen space, make transparent
            if terrain_node.global_position.y < mouse_pos.y:
                terrain_node.modulate.a = 0.4
            else:
                terrain_node.modulate.a = 1.0
```

**Recommendation:** Cursor-proximity for more intuitive feel

---

## Implementation Phases

### Phase 1: Core Map Format & Loading (Foundation)

**Server:**
1. Create `MapData` struct and JSON parsing
2. Add `LoadMap()` function
3. Replace hardcoded arena size with map dimensions
4. Update `isTilePassable()` to check terrain

**Client:**
1. Receive map data in welcome message
2. Store map dimensions
3. Update camera bounds
4. No visual changes yet (still renders grid)

**Test:** Same gameplay, but map size is now configurable

---

### Phase 2: Window & Camera (Playability)

**Client:**
1. Update `project.godot` for larger window (1280×720)
2. Add camera zoom (mouse wheel)
3. Add camera pan (WASD/arrows)
4. Test with current grid rendering

**Server:** No changes needed

**Test:** Larger viewport, can zoom and pan around existing grid

---

### Phase 3: Basic Terrain Rendering (Visuals)

**Client:**
1. Create `TerrainLayer` node
2. Render grass tiles for entire map
3. Render rock tiles for test obstacles
4. Apply isometric transformation

**Server:**
1. Add sample terrain to map file
2. Include terrain in welcome message

**Test:** See varied terrain, rocks block movement

---

### Phase 4: Multi-Tile Features (Rich Environment)

**Both:**
1. Add mountain/forest/lake definitions to map format
2. Server validates passability for multi-tile features
3. Client renders large features (3D box or sprite)

**Test:** Large features visible, block movement correctly

---

### Phase 5: Occlusion System (Polish)

**Client:**
1. Add height metadata to terrain nodes
2. Implement cursor-proximity transparency
3. Smooth fade with lerp
4. Test with tall mountains near units

**Test:** Can see units behind mountains when cursor nearby

---

## Performance Considerations

### Server
- **Passability checks**: O(1) with sparse map + spatial hash for features
- **Map loading**: Once at startup, ~10ms for 100×100 map
- **Memory**: ~1KB per 100 tiles (sparse storage)

### Client
- **Rendering**: ~1000 tiles = 1000 Polygon2D nodes (acceptable)
- **Occlusion**: Check N terrain nodes per frame (limit to nearby nodes)
- **Optimization**: Use TileMap node instead of Polygon2D (10x faster rendering)

### Bandwidth
- **Option A** (send full map): ~10-50KB once at connect
- **Option B** (map name only): ~100 bytes, client loads locally

**Recommendation:** Start with Option A (simpler), move to Option B if needed

---

## Example Maps

### Small Test Map (40×30)
- Bordered by rocks
- Small lake in center
- Forest in one corner
- Good for testing

### Medium Arena (80×60)
- Multiple spawn points (2-4 teams)
- Rivers dividing sections
- Mountain ranges as barriers
- Forests for hiding

### Large Battlefield (120×100)
- Open plains
- Strategic chokepoints (narrow passes)
- Resource-rich center
- Defensive positions (elevated terrain)

---

## Future Enhancements

1. **Height-based combat**: Units on high ground get bonuses
2. **Fog of war**: Terrain blocks line of sight
3. **Destructible terrain**: Forests can be cleared, water can be frozen
4. **Dynamic terrain**: Tides, seasons, growing vegetation
5. **Procedural generation**: Generate random maps with noise

---

## Open Questions

1. **Map distribution**: Bundle maps with client or download on-demand?
2. **Tile variations**: Single grass tile or multiple variants for variety?
3. **Feature rendering**: Sprites vs. procedural (Polygon2D)?
4. **Pathfinding**: A* around obstacles (future requirement)?
5. **Map editor**: Build in Godot or separate tool?

---

**This document serves as the blueprint for the map system. Implementation will begin with Phase 1 and iterate based on testing.**
