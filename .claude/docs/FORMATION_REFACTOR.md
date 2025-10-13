# Formation Orientation Refactor

**Date:** 2025-10-13
**Status:** 🔄 In Progress
**Breaking Change:** Yes - formation positioning behavior changes

---

## Problem Statement

Current formation system places formations centered on the click point, which is unintuitive for RTS gameplay.

**Issues:**
1. **Click point = formation center** - User clicks where they want units to go, but center unit goes there instead of front unit
2. **Line always horizontal** - Line formation is always horizontal regardless of movement direction
3. **No direction awareness** - Formations don't orient toward movement direction

**User Expectation (from RTS games like StarCraft, AoE):**
- Click point should be the **tip/front** of the formation
- Formation should **orient toward movement direction**
- Line should be **perpendicular to movement direction**

---

## Current Implementation (BEFORE)

### Formation Functions

**Location:** `server/main.go` lines 940-1096

**Entry Point:**
```go
func (s *GameServer) calculateFormation(formation string, centerX, centerY, numUnits int) []TilePosition
```

**Formations:**

1. **Box Formation** (`calculateBoxFormation`)
   - Creates √n × √n grid
   - **Centers grid on click point** ← PROBLEM
   - Always oriented N-S and E-W (no rotation)
   - Example: 5 units = 3×3 grid, center unit at click point

2. **Line Formation** (`calculateLineFormation`)
   - Creates horizontal line
   - **Always horizontal** ← PROBLEM
   - Centers line on click point
   - Example: 5 units in row, middle unit at click point

3. **Spread/Spiral Formation** (`calculateSpiralFormation`)
   - Creates spiral from center outward
   - Center unit at click point (this is OK for spiral)
   - No changes needed for this formation

### Current Behavior Examples

**Box Formation - 5 units, click at (10, 10):**
```
     8  9  10 11 12
  8  ·  ·  ·  ·  ·
  9  ·  U  U  U  ·
 10  ·  U  ★  U  ·   ← Click point (center)
 11  ·  ·  U  ·  ·
```

**Line Formation - 5 units, click at (10, 10), moving from left:**
```
     8  9  10 11 12
 10  U  U  ★  U  U   ← Always horizontal
```

---

## Desired Implementation (AFTER)

### Behavior Changes

**1. Direction-Based Orientation**
- Calculate unit centroid (average position)
- Determine direction: centroid → click point
- Orient formation to face that direction

**2. Tip at Click Point**
- Formation tip (front-most unit) goes to click point
- Formation extends backward from tip

**3. Line Parallel to Movement**
- Moving horizontally → horizontal line
- Moving vertically → vertical line
- Moving diagonally → diagonal line (same direction)

### New Behavior Examples

**Box Formation - Moving RIGHT (units at x=5, click at x=15):**
```
Direction: East (→)

Before (centered):          After (tip at click):
     13 14 15 16 17             13 14 15 16 17
 14  U  U  ★  U  U          14  ·  ·  ·  ·  ·
 15  ·  ·  U  ·  ·          15  U  U  ★  U  U   ← Tip at click
                            16  U  ·  ·  ·  ·   ← Formation extends back
```

**Line Formation - Moving UP (units at y=15, click at y=5):**
```
Direction: North (↑)

Before (horizontal):        After (parallel):
      8  9  10 11 12             8  9  10 11 12
  5  U  U  ★  U  U           5  ·  ·  ★  ·  ·   ← Tip at click
                             6  ·  ·  U  ·  ·   ← Vertical line (parallel to north)
                             7  ·  ·  U  ·  ·
                             8  ·  ·  U  ·  ·
                             9  ·  ·  U  ·  ·
```

---

## Implementation Plan

### Stage 1: Direction Calculation (Foundation)

**New Functions:**
```go
// Calculate average position of units (centroid)
func (s *GameServer) calculateUnitCentroid(unitIds []uint32) (float64, float64)

// Calculate normalized direction vector from centroid to target
func (s *GameServer) calculateMovementDirection(unitIds []uint32, targetX, targetY int) (dx, dy float64)

// Convert direction vector to cardinal/ordinal direction string
func getPrimaryDirection(dx, dy float64) string // Returns: N, NE, E, SE, S, SW, W, NW
```

**Testing:**
- Test centroid calculation with various unit positions
- Test direction classification (8 directions)
- Test edge cases (units at target, single unit)

### Stage 2: Oriented Box Formation

**Modified Function:**
```go
func (s *GameServer) calculateBoxFormationOriented(tipX, tipY, numUnits int, direction string) []TilePosition
```

**Logic:**
- Calculate grid size (√n × √n)
- Position grid based on direction:
  - **E (East):** Tip on right, grid extends left and centered vertically
  - **W (West):** Tip on left, grid extends right and centered vertically
  - **N (North):** Tip on top, grid extends down and centered horizontally
  - **S (South):** Tip on bottom, grid extends up and centered horizontally
  - **Diagonals (NE, NW, SE, SW):** Rotate grid 45°

**Testing:**
- Test each of 8 directions
- Verify tip unit is at click point
- Verify formation shape is correct

### Stage 3: Oriented Line Formation

**Modified Function:**
```go
func (s *GameServer) calculateLineFormationOriented(tipX, tipY, numUnits int, direction string) []TilePosition
```

**Logic:**
- Line is parallel to movement direction
- Position line with tip at click point:
  - **E/W movement:** Create horizontal line along movement axis
  - **N/S movement:** Create vertical line along movement axis
  - **Diagonal movement:** Create diagonal line in same direction

**Testing:**
- Test each movement direction
- Verify line is parallel to movement
- Verify tip is at click point

### Stage 4: Integration

**Modified Function:**
```go
func (s *GameServer) handleMoveCommand(cmd Command, client *Client)
```

**Changes:**
1. Calculate direction before formation calculation
2. Call oriented formation functions instead of old ones
3. Keep spread/spiral unchanged

**Old Code:**
```go
formationPositions := s.calculateFormation(formation, tileX, tileY, len(validUnitIds))
```

**New Code:**
```go
dx, dy := s.calculateMovementDirection(validUnitIds, tileX, tileY)
direction := getPrimaryDirection(dx, dy)
formationPositions := s.calculateFormationOriented(formation, tileX, tileY, len(validUnitIds), direction)
```

### Stage 5: Testing & Refinement

**Unit Tests:**
- Add `TestFormationOrientation` to `game_test.go`
- Test all formations × all directions
- Test edge cases (blocked tiles, single unit, etc.)

**Manual Testing:**
- Server + client visual testing
- Verify formations look correct
- Verify formations feel intuitive

---

## Files Modified

**Primary:**
- `server/main.go` - Add ~150 lines of new formation logic

**Tests:**
- `server/game_test.go` - Add orientation tests

**Documentation:**
- This file (FORMATION_REFACTOR.md)
- `.claude/docs/CURRENT_STATE.md` - Note breaking change

---

## Rollback Plan

If refactor causes issues:

1. **Restore from snapshot:** `.claude/docs/FORMATION_BEFORE.md` contains original code
2. **Git revert:** Commit will be tagged for easy revert
3. **Feature flag:** Could add `--legacy-formations` flag if needed

---

## Design Decisions

### Why 8 Directions Instead of Continuous?

**Decision:** Use 8 cardinal/ordinal directions (N, NE, E, SE, S, SW, W, NW)

**Rationale:**
- Simpler to implement and test
- Matches tile-based grid system
- Easier for players to predict
- Good enough for RTS gameplay

**Alternative considered:** Continuous rotation (any angle)
- More complex math
- Harder to align with grid
- Minimal gameplay benefit

### Why Calculate Direction from Centroid?

**Decision:** Use average position of selected units as reference point

**Rationale:**
- Intuitive (where units are coming from)
- Handles multi-unit selection well
- Standard RTS behavior

**Alternative considered:** Use closest unit to click point
- Less intuitive when units are spread out
- Doesn't match player expectation

### Why Keep Spread/Spiral Unchanged?

**Decision:** Spiral formation stays centered on click point

**Rationale:**
- Spiral has no "front" - it's radially symmetric
- Centering makes sense for spread formations
- Matches player expectation for "spread out here"

---

## Breaking Changes

**For Users:**
- Formations will position differently than before
- May break muscle memory for existing players
- Should feel more intuitive after adjustment

**For Code:**
- Formation function signatures change
- Tests need updating
- Scenarios may need position adjustments

---

## Success Criteria

**Functional:**
- ✅ Tip of formation at click point
- ✅ Formation oriented toward movement direction
- ✅ Line perpendicular to movement
- ✅ All unit tests pass
- ✅ Scenario tests pass (may need tolerance adjustment)

**User Experience:**
- ✅ Formations feel intuitive
- ✅ No unexpected unit positioning
- ✅ Matches RTS game expectations

---

## Timeline

**Estimated:** 2-3 hours

- Stage 1 (Direction): 30 min
- Stage 2 (Box): 45 min
- Stage 3 (Line): 45 min
- Stage 4 (Integration): 15 min
- Stage 5 (Testing): 45 min

---

## References

**RTS Games with Similar Behavior:**
- StarCraft / StarCraft II
- Age of Empires II
- Warcraft III
- Command & Conquer

**Related Files:**
- `server/main.go:940-1096` - Current formation code
- `server/main.go:1098-1197` - handleMoveCommand
- `client/GameController.gd:512-523` - Client-side move command
