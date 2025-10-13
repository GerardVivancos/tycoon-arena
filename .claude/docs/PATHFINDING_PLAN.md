# A* Pathfinding Implementation Plan

**Status:** Planning
**Date:** 2025-10-13
**Goal:** Implement server-side A* pathfinding for intelligent unit navigation

---

## Problem Statement

Current issues with movement system:
1. **Units pass through impassable terrain** - Units interpolate directly to target, ignoring rocks/obstacles in path
2. **Units stack when formations overlap** - Multiple units can be assigned same tile position

---

## Solution: A* Pathfinding

Implement server-side pathfinding that:
- Calculates intelligent paths around obstacles
- Avoids unit stacking through collision detection
- Maintains smooth visual movement on client
- Allows dynamic path updates if blocked

---

## Architecture Overview

### Server-Side (Authoritative)
- Calculates full path using A* algorithm
- Stores path as array of waypoints
- Follows path one tile at a time
- Sends: current tile, next waypoint, progress (0.0-1.0)
- Can recalculate path if needed

### Client-Side (No Changes)
- Continues interpolating between tiles
- Uses `lerp(currentTile, targetTile, moveProgress)`
- Smooth animation automatically follows path
- Doesn't need to know full path

---

## Implementation Plan

### Phase 1: Core Pathfinding (This Implementation)

#### 1. Add Path Storage to Entity
```go
type Entity struct {
    // ... existing fields ...
    Path            []TilePosition  // Full path from start to goal
    PathIndex       int             // Current waypoint index in path
}
```

#### 2. Implement A* Algorithm
```go
// Main pathfinding function
func (s *GameServer) findPath(startX, startY, goalX, goalY int, unitId uint32) []TilePosition

// Supporting structures
type pathNode struct {
    x, y    int
    gCost   float32  // Cost from start
    hCost   float32  // Heuristic to goal (Manhattan distance)
    fCost   float32  // gCost + hCost (total estimated cost)
    parent  *pathNode
}

// Priority queue for open set (using container/heap)
type nodeHeap []*pathNode
// Implements: Len(), Less(), Swap(), Push(), Pop()
```

**Algorithm Steps:**
1. Initialize open set (priority queue by fCost) and closed set (visited nodes)
2. Add start node to open set
3. While open set not empty:
   - Pop node with lowest fCost
   - If node is goal: reconstruct path by following parent pointers, return
   - Add node to closed set
   - For each neighbor (4-directional: N, E, S, W):
     - Skip if impassable or already in closed set
     - Calculate gCost = parent.gCost + 1.0
     - Calculate hCost = manhattanDistance(neighbor, goal)
     - If neighbor not in open set OR new path is better:
       - Update neighbor's costs and parent
       - Add to open set
4. If open set exhausted: return nil (no path exists)

**Heuristic:** Manhattan distance (good for tile-based grids)
```go
h = abs(goalX - currentX) + abs(goalY - currentY)
```

**Movement:** 4-directional only (no diagonals initially)
```go
directions := [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}  // N, E, S, W
```

#### 3. Unit Collision Detection
```go
// Check if another unit occupies this tile
func (s *GameServer) isTileOccupiedByUnit(tileX, tileY int, excludeId uint32) bool {
    // Check all worker units (excluding the one we're checking for)
    // A tile is occupied if:
    // - A unit's TileX/TileY equals this tile (currently there)
    // - OR unit's TargetTileX/TargetTileY equals this tile (moving there)
    // This prevents units from pathing through each other
}

// Combined availability check
func (s *GameServer) isTileAvailableForUnit(tileX, tileY int, unitId uint32) bool {
    // Returns true only if:
    // 1. Tile is passable (terrain + buildings)
    // 2. No other unit occupies/targets this tile
}
```

#### 4. Update handleMoveCommand()
**OLD:** Directly assign formation target to `entity.TargetTileX/Y`
**NEW:** Calculate path to formation target
```
For each unit in formation:
1. Get unit's formation position from calculateFormation()
2. Calculate path: findPath(unit.TileX, unit.TileY, formationX, formationY, unit.Id)
3. If path found:
   - entity.Path = path
   - entity.PathIndex = 0
   - entity.TargetTileX/Y = path[0] (first waypoint)
4. If no path found:
   - Log warning
   - Leave unit at current position (don't move)
```

#### 5. Replace updateEntityMovement()
**OLD:** Direct interpolation to final target
**NEW:** Follow path waypoint-by-waypoint
```go
func (s *GameServer) updateEntityMovement(entity *Entity, deltaTime float32) {
    // 1. Check if entity has a path
    if len(entity.Path) == 0 {
        entity.MoveProgress = 0.0
        return  // No path, unit is stationary
    }

    // 2. Get next waypoint
    if entity.PathIndex >= len(entity.Path) {
        // Path complete, clear it
        entity.Path = nil
        entity.PathIndex = 0
        entity.MoveProgress = 0.0
        return
    }

    waypoint := entity.Path[entity.PathIndex]
    entity.TargetTileX = waypoint.X
    entity.TargetTileY = waypoint.Y

    // 3. Interpolate to waypoint (existing logic)
    progressIncrement := MovementSpeed * deltaTime
    entity.MoveProgress += progressIncrement

    // 4. Check if waypoint reached
    if entity.MoveProgress >= 1.0 {
        // Move to waypoint
        entity.TileX = waypoint.X
        entity.TileY = waypoint.Y
        entity.MoveProgress = 0.0

        // Advance to next waypoint
        entity.PathIndex++

        // Check if path complete
        if entity.PathIndex >= len(entity.Path) {
            entity.Path = nil
            entity.PathIndex = 0
        }
    }
}
```

#### 6. Formation Integration
- Formations continue to work as before
- Each unit independently calculates path to its formation position
- If formation position unreachable: unit stays at current position
- Units navigate around obstacles to reach their spots

---

### Phase 2: Dynamic Replanning (Future Enhancement)

Add to Entity:
```go
PathRecalcTimer float32  // Seconds since last recalc
```

In updateEntityMovement(), add:
```go
// Periodically check if path is still valid
entity.PathRecalcTimer += deltaTime
if entity.PathRecalcTimer > 0.5 {  // Check every 0.5 seconds
    entity.PathRecalcTimer = 0.0

    // Check if next waypoint is now blocked
    nextWaypoint := entity.Path[entity.PathIndex]
    if !s.isTileAvailableForUnit(nextWaypoint.X, nextWaypoint.Y, entity.Id) {
        // Path blocked, recalculate
        goalWaypoint := entity.Path[len(entity.Path)-1]  // Final destination
        newPath := s.findPath(entity.TileX, entity.TileY, goalWaypoint.X, goalWaypoint.Y, entity.Id)
        if newPath != nil {
            entity.Path = newPath
            entity.PathIndex = 0
        } else {
            // No path available, stop moving
            entity.Path = nil
            entity.PathIndex = 0
        }
    }
}
```

---

## Performance Considerations

### Complexity
- **A* worst case:** O(tiles × log(tiles))
- **Map size:** 40×30 = 1200 tiles
- **Operations per path:** ~1200 × log(1200) ≈ 12,000 operations
- **Per command:** 10 units × 12k ops = 120k operations (acceptable)

### Optimizations
- Early exit when goal reached
- Limit search depth (max path length)
- Cache pathfinding for identical start/goal pairs (optional)
- Pathfinding happens on command, not every frame

### Memory
- Path storage: ~10-50 tiles per unit × 10 units = ~500 tiles in memory
- Negligible compared to map data (1200 tiles)

---

## Network Protocol

**No changes needed** - Client continues receiving per-snapshot:
```json
{
  "id": 10,
  "type": "worker",
  "tileX": 5,              // Current tile
  "tileY": 3,
  "targetTileX": 6,        // Next waypoint (not final destination)
  "targetTileY": 3,
  "moveProgress": 0.45     // Interpolation progress (0.0 to 1.0)
}
```

Full path remains server-side only. Client doesn't need it.

---

## Edge Cases & Handling

| Case | Behavior |
|------|----------|
| No path exists | Unit stays at current position, log warning |
| Formation target unreachable | Unit stays at current position |
| Unit in path during search | Treat as static obstacle (first unit has right-of-way) |
| Path blocked mid-movement | (Phase 1) Unit stops. (Phase 2) Recalculate path |
| Multiple units to same goal | Each finds independent path (may stack at goal tile) |
| Unit already at goal | Return path with single element (current position) |

---

## Testing Plan

### Unit Tests
1. ✅ Pathfinding around single rock obstacle
2. ✅ Pathfinding around cluster of rocks
3. ✅ Pathfinding around buildings
4. ✅ No path available (surrounded by obstacles)
5. ✅ Direct path (no obstacles)
6. ✅ Unit at goal (zero-length path)

### Integration Tests
1. ✅ Formation command with obstacles (units path around)
2. ✅ Two units crossing paths (avoid collision)
3. ✅ Unit commanded while moving (recalculate path)
4. ✅ Multiple units to same goal (independent paths)
5. ✅ Unit encounters new obstacle (Phase 2: replanning)

### Visual Tests
1. ✅ Smooth walking animation along path
2. ✅ No teleporting or jittering
3. ✅ Units turn naturally at corners
4. ✅ Formations maintain shape around obstacles

---

## Code Structure

### New Functions (server/main.go)
```go
// Pathfinding core
func (s *GameServer) findPath(startX, startY, goalX, goalY int, unitId uint32) []TilePosition
func (s *GameServer) manhattanDistance(x1, y1, x2, y2 int) float32
func (s *GameServer) getNeighbors(x, y int) []TilePosition
func (s *GameServer) reconstructPath(node *pathNode) []TilePosition

// Unit collision
func (s *GameServer) isTileOccupiedByUnit(tileX, tileY int, excludeId uint32) bool
func (s *GameServer) isTileAvailableForUnit(tileX, tileY int, unitId uint32) bool

// Priority queue for A*
type nodeHeap []*pathNode
func (h nodeHeap) Len() int
func (h nodeHeap) Less(i, j int) bool
func (h nodeHeap) Swap(i, j int)
func (h *nodeHeap) Push(x any)
func (h *nodeHeap) Pop() any
```

### Modified Functions (server/main.go)
```go
func (s *GameServer) handleMoveCommand(...)    // Use pathfinding instead of direct target
func (s *GameServer) updateEntityMovement(...) // Follow path waypoint-by-waypoint
```

---

## Estimated Code Size
- **Pathfinding algorithm:** ~150 lines
- **Priority queue (heap):** ~40 lines
- **Unit collision functions:** ~30 lines
- **Path following logic:** ~40 lines
- **Total:** ~260 lines of new code

---

## Files to Modify
- `server/main.go` - Add pathfinding system

---

## Expected Behavior After Implementation

**Before:**
- ❌ Units walk through rocks
- ❌ Units stack when formations overlap
- ❌ Unrealistic straight-line movement

**After:**
- ✅ Units navigate around obstacles intelligently
- ✅ Units avoid stacking (path around each other)
- ✅ Smooth visual movement along calculated paths
- ✅ Server-authoritative pathfinding (no client hacks)
- ✅ Dynamic replanning if paths blocked (Phase 2)

---

## Future Enhancements (Beyond This Plan)

### Pathfinding Improvements
- Diagonal movement (8-directional A*)
- Jump point search (faster pathfinding)
- Hierarchical pathfinding (for large maps)
- Flow fields (for groups of units)

### Collision Improvements
- Unit "pushing" (units move aside slightly)
- Predicted positions (anticipate where units will be)
- Priority-based pathing (important units get right-of-way)
- Formation-aware pathfinding (maintain formation shape)

### Visual Improvements
- Path preview (show path on client for selected units)
- Waypoint markers (debug visualization)
- Smooth turning (rotate unit sprite to face movement direction)

---

## References
- A* Algorithm: https://en.wikipedia.org/wiki/A*_search_algorithm
- Go heap implementation: https://pkg.go.dev/container/heap
- RTS Pathfinding: https://www.redblobgames.com/pathfinding/a-star/

---

**Next Steps:**
1. Implement A* pathfinding algorithm
2. Add path storage to Entity struct
3. Update movement system to follow paths
4. Test with various obstacle configurations
5. (Optional) Add dynamic replanning
