# Formation Movement - Implementation Plan

**Date:** 2025-10-13
**Status:** 📋 Planned (Not Yet Implemented)
**Priority:** High - Core RTS feature
**Estimated Effort:** 2-3 days

---

## Overview

Formation movement allows units to move **as a cohesive group** while maintaining their formation shape during travel, not just at the destination. This is a core feature of RTS games like Age of Empires II.

**Current State:**
- ✅ Formation positioning at destination (tip at click point, extends backward)
- ✅ Units sorted by distance (closest becomes tip)
- ❌ Units pathfind independently to formation positions
- ❌ Formation shape not maintained during movement

**Desired State:**
- Units move together as a "squad"
- Formation shape maintained during travel
- Leader pathfinding, followers maintain relative positions
- Formation adapts to obstacles (break/reform)

---

## User Experience Goals

### Age of Empires II Reference Behavior

**Line Formation Moving South:**
```
Start:                  During Movement:        Destination:
  U U U U U               ·  ·  ·  ·  ·           ·  ·  ·  ·  ·
  ·  ·  ·  ·  ·           U  U  U  U  U           ·  ·  ·  ·  ·
  ·  ·  ·  ·  ·    →      ·  ·  ·  ·  ·     →     ·  ·  ·  ·  ·
  ·  ·  ·  ·  ·           ·  ·  ·  ·  ·           U  U  U  U  U
  ·  ·  ·  ·  ·           ·  ·  ·  ·  ·           ·  ·  ·  ·  ·

Units maintain horizontal line shape while moving downward
```

**Box Formation Moving East:**
```
Start:                  During Movement:        Destination:
  U U ·  ·  ·             ·  U  U  ·  ·           ·  ·  U  U  ·
  U U ·  ·  ·             ·  U  U  ·  ·           ·  ·  U  U  ·
  ·  ·  ·  ·  ·    →      ·  ·  ·  ·  ·     →     ·  ·  ·  ·  ·
  ·  ·  ·  ·  ·           ·  ·  ·  ·  ·           ·  ·  ·  ·  ·

2×2 box shape maintained while moving rightward
```

---

## Design Approach

### Option 1: Leader-Follower System (Recommended)

**Concept:**
- One unit (tip/leader) pathfinds to destination
- Other units maintain fixed offset from leader
- Formation moves as a rigid body

**Pros:**
- Simpler to implement
- Good performance (only one pathfinding calculation)
- Predictable behavior

**Cons:**
- Formation can get stuck if any unit blocked
- Less adaptable to terrain

**Implementation:**
```go
type Formation struct {
    LeaderID   uint32
    MemberIDs  []uint32
    Type       string  // "box", "line", "spread"
    Offsets    map[uint32]TilePosition  // Relative to leader
}

func (s *GameServer) moveFormation(formation *Formation, targetX, targetY int) {
    leader := s.entities[formation.LeaderID]

    // Only leader pathfinds
    leaderPath := s.findPath(leader.TileX, leader.TileY, targetX, targetY, leader.Id)

    // Update all units relative to leader's current position
    for _, memberID := range formation.MemberIDs {
        member := s.entities[memberID]
        offset := formation.Offsets[memberID]

        // Follower target = leader position + offset
        member.TargetTileX = leader.TileX + offset.X
        member.TargetTileY = leader.TileY + offset.Y
    }
}
```

### Option 2: Adaptive Formation (More Complex)

**Concept:**
- All units pathfind, but with formation constraints
- Units try to maintain formation while adapting to obstacles
- Formation can stretch/compress based on terrain

**Pros:**
- More natural around obstacles
- Units can navigate independently when needed
- Formation reforms after obstacles

**Cons:**
- Much more complex
- Higher performance cost (multiple pathfinding)
- Can look chaotic if units diverge too much

---

## Implementation Stages

### Stage 1: Basic Formation Groups (2-4 hours)

**Add Formation Tracking:**
```go
type FormationGroup struct {
    ID          uint32
    Type        string  // "box", "line", "spread"
    LeaderID    uint32
    MemberIDs   []uint32
    Offsets     map[uint32]TilePosition
    TargetX     int
    TargetY     int
    IsMoving    bool
}

type GameServer struct {
    // ... existing fields
    formations  map[uint32]*FormationGroup
    nextFormationID uint32
}
```

**Modify handleMoveCommand:**
- Create FormationGroup for each move command
- Store formation data
- Track which units are in formations

**Testing:**
- Formation groups created correctly
- Leader identified (closest to click)
- Offsets calculated correctly

### Stage 2: Leader Pathfinding (3-4 hours)

**Update Movement Logic:**
```go
func (s *GameServer) tickFormations() {
    for _, formation := range s.formations {
        if !formation.IsMoving {
            continue
        }

        leader := s.entities[formation.LeaderID]

        // Leader follows path
        s.updateEntityMovement(leader)

        // Check if leader reached destination
        if leader.TileX == formation.TargetX && leader.TileY == formation.TargetY {
            formation.IsMoving = false
            delete(s.formations, formation.ID)  // Disband formation
        }
    }
}
```

**Update tickFormationFollowers:**
```go
func (s *GameServer) tickFormationFollowers(formation *FormationGroup) {
    leader := s.entities[formation.LeaderID]

    for _, memberID := range formation.MemberIDs {
        if memberID == formation.LeaderID {
            continue
        }

        member := s.entities[memberID]
        offset := formation.Offsets[memberID]

        // Calculate desired position
        desiredX := leader.TileX + offset.X
        desiredY := leader.TileY + offset.Y

        // Move toward desired position if passable
        if s.isTilePassable(desiredX, desiredY) {
            member.TargetTileX = desiredX
            member.TargetTileY = desiredY
            // ... update movement
        }
    }
}
```

**Testing:**
- Leader pathfinds to destination
- Followers maintain offset from leader
- Formation shape preserved during straight-line movement

### Stage 3: Obstacle Handling (4-6 hours)

**Break Formation on Block:**
```go
func (s *GameServer) checkFormationBlocked(formation *FormationGroup) bool {
    // If any follower can't reach desired position for N ticks
    for _, memberID := range formation.MemberIDs {
        member := s.entities[memberID]
        if member.BlockedTicks > 20 {  // 1 second at 20Hz
            return true
        }
    }
    return false
}

func (s *GameServer) breakFormation(formation *FormationGroup) {
    // Switch all units to independent pathfinding
    for _, memberID := range formation.MemberIDs {
        member := s.entities[memberID]
        // Each unit pathfinds to final formation position
        path := s.findPath(member.TileX, member.TileY, formation.TargetX, formation.TargetY, memberID)
        member.Path = path
    }

    delete(s.formations, formation.ID)
}
```

**Reform After Obstacle:**
```go
func (s *GameServer) attemptFormationReform(unitIDs []uint32, targetX, targetY int) {
    // Check if units are close enough to reform
    allNearDestination := true
    for _, id := range unitIDs {
        entity := s.entities[id]
        dist := abs(entity.TileX - targetX) + abs(entity.TileY - targetY)
        if dist > 5 {  // Within 5 tiles of destination
            allNearDestination = false
            break
        }
    }

    if allNearDestination {
        // Reform into destination formation
        // ... recreate formation
    }
}
```

**Testing:**
- Formation breaks when followers blocked
- Units switch to independent pathfinding
- Formation reforms near destination (optional)

### Stage 4: Speed Synchronization (2-3 hours)

**Wait for Stragglers:**
```go
func (s *GameServer) synchronizeFormationSpeed(formation *FormationGroup) {
    leader := s.entities[formation.LeaderID]

    // Check if any follower is lagging
    maxLag := 0
    for _, memberID := range formation.MemberIDs {
        if memberID == formation.LeaderID {
            continue
        }

        member := s.entities[memberID]
        offset := formation.Offsets[memberID]
        expectedX := leader.TileX + offset.X
        expectedY := leader.TileY + offset.Y

        lag := abs(member.TileX - expectedX) + abs(member.TileY - expectedY)
        if lag > maxLag {
            maxLag = lag
        }
    }

    // Leader waits if followers too far behind
    if maxLag > 2 {
        leader.WaitForFormation = true
    } else {
        leader.WaitForFormation = false
    }
}
```

**Testing:**
- Leader slows/stops when followers lagging
- Formation stays tight
- No excessive stopping

### Stage 5: Client Visual Updates (1-2 hours)

**Network Protocol Changes:**
```json
{
  "type": "snapshot",
  "data": {
    "formations": [
      {
        "id": 123,
        "type": "line",
        "leaderID": 10,
        "memberIDs": [10, 11, 12, 13, 14],
        "isMoving": true
      }
    ]
  }
}
```

**Client Rendering:**
- Optional: Draw formation outline
- Optional: Visual indicator for leader
- Smooth interpolation for formation movement

---

## Edge Cases & Considerations

### 1. Formation Breaking Conditions

**When to break formation:**
- Any unit blocked for >1 second
- Terrain splits formation (e.g., narrow passage)
- User issues new move command to subset of units

**When to maintain formation:**
- Minor path deviations (unit reroutes briefly)
- Temporary blocking (another unit passing through)

### 2. Multiple Formations

**Scenario:** Player has two separate groups in different formations

**Solution:**
- Each FormationGroup has unique ID
- Units can only be in one formation at a time
- New move command removes unit from old formation

### 3. Mixed Unit Types (Future)

When different unit types added (fast scouts, slow tanks):
- Formation speed = slowest unit
- Or: allow fast units to scout ahead (break formation)

### 4. Formation Size Limits

**Question:** Should very large formations (20+ units) work differently?

**Options:**
- A) Same behavior (may be chaotic)
- B) Auto-split into sub-formations
- C) Limit formation size, excess units follow independently

**Recommendation:** Start with (A), add (B) if needed

---

## Testing Strategy

### Unit Tests

```go
func TestFormationGroupCreation(t *testing.T)
func TestLeaderIdentification(t *testing.T)
func TestOffsetCalculation(t *testing.T)
func TestFormationMovementStraightLine(t *testing.T)
func TestFormationBreakOnObstacle(t *testing.T)
func TestSpeedSynchronization(t *testing.T)
```

### Scenario Tests

**Scenario 1: Unobstructed Line Movement**
- 5 units in line, move across open ground
- Verify formation shape maintained
- Verify arrival at destination in formation

**Scenario 2: Formation Through Narrow Passage**
- Line formation approaching 1-tile-wide corridor
- Verify formation breaks
- Verify units pathfind individually through passage
- (Optional) Verify reformation on far side

**Scenario 3: Box Formation Around Obstacle**
- 4-unit box moving toward destination with obstacle in path
- Verify formation adapts or breaks appropriately
- Verify eventual arrival at destination

### Manual Testing

- Visual test with Godot client
- Verify formations "feel right" during movement
- Test with various terrain layouts
- Test with multiple simultaneous formations

---

## Performance Considerations

**Pathfinding Cost:**
- Leader-follower: O(1) pathfinding per formation (only leader)
- Independent: O(N) pathfinding per formation (every unit)

**Recommendation:** Start with leader-follower for performance

**Optimization Opportunities:**
- Cache formation offsets
- Only update follower targets when leader moves
- Skip formation tick for formations far from action

**Estimated Cost:**
- 5 formations of 5 units each = 25 units
- Leader-follower: 5 pathfinding calls
- Independent: 25 pathfinding calls

---

## Implementation Checklist

**Stage 1: Formation Tracking**
- [ ] Add FormationGroup struct
- [ ] Add formations map to GameServer
- [ ] Modify handleMoveCommand to create formations
- [ ] Calculate offsets for each formation type
- [ ] Unit tests for formation creation

**Stage 2: Leader Movement**
- [ ] Implement tickFormations()
- [ ] Leader pathfinds, followers use offsets
- [ ] Update entity movement to check formation membership
- [ ] Unit tests for leader-follower movement

**Stage 3: Obstacle Handling**
- [ ] Detect blocked followers
- [ ] Break formation logic
- [ ] Switch to independent pathfinding
- [ ] (Optional) Reform logic
- [ ] Scenario tests for obstacle navigation

**Stage 4: Speed Synchronization**
- [ ] Detect lagging followers
- [ ] Leader wait logic
- [ ] Tune lag threshold
- [ ] Unit tests for synchronization

**Stage 5: Client Updates**
- [ ] Add formation data to snapshot
- [ ] Client rendering (optional visual enhancements)
- [ ] Manual testing with Godot client

**Documentation:**
- [ ] Update CURRENT_STATE.md
- [ ] Update ARCHITECTURE.md with formation system
- [ ] Update NETWORK_PROTOCOL.md with formation messages

---

## Rollback Plan

If formation movement causes issues:

1. **Feature flag:** Add `--disable-formation-movement` server flag
2. **Graceful degradation:** Fall back to current behavior (independent pathfinding)
3. **Commit revert:** Tag implementation commit for easy revert

---

## Future Enhancements

**Beyond basic formation movement:**
- [ ] Formation facing/orientation while moving
- [ ] Attack-move in formation
- [ ] Formation stance (tight vs. loose)
- [ ] Custom formations (user-defined shapes)
- [ ] Formation AI (auto-adjust to terrain)
- [ ] Mixed unit type formations (infantry + archers)

---

## References

**Similar Systems:**
- Age of Empires II formation movement
- StarCraft II unit grouping
- Total War formation mechanics
- Command & Conquer unit groups

**Related Code:**
- `server/main.go:1385-1430` - Current handleMoveCommand (formation positioning)
- `server/main.go:1147-1221` - calculateLineFormationOriented
- `server/main.go:1012-1095` - calculateBoxFormationOriented

**Related Docs:**
- `.claude/docs/FORMATION_REFACTOR.md` - Formation positioning (completed)
- `.claude/docs/PATHFINDING_IMPLEMENTATION.md` - A* pathfinding system
