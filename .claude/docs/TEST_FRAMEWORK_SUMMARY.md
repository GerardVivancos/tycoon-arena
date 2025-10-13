# Test Framework - Summary

**Status:** ✅ Complete (Simplified & Improved)
**Date:** 2025-10-13
**Tests:** 12 total (10 unit + 2 scenario)

---

## Overview

The test framework provides comprehensive coverage through:
1. **Unit tests** - Explicit tests for pathfinding, terrain, and mechanics
2. **Scenario tests** - Declarative JSON-based tests for integration

**Key improvement:** Added explicit negative tests that verify "can't go over rocks" instead of only testing routing around obstacles.

---

## Test Coverage

### Unit Tests (10 tests)

**Pathfinding (4 tests):**
- `TestPathfindingAroundSingleRock` - Routes around 1 obstacle
- `TestPathfindingAroundCluster` - Navigates 3×2 rock cluster
- `TestPathfindingNoPath` - Returns nil for unreachable destination
- `TestPathDoesNotGoThroughRock` - ✨ **NEW**: Verifies path avoids rocks

**Terrain & Obstacles (4 tests):**
- `TestCannotPathToRock` - ✨ **NEW**: Pathfinding TO rock returns nil
- `TestCannotMoveToRock` - ✨ **NEW**: Unit command to rock rejected
- `TestRockBlocksBuilding` - ✨ **NEW**: Building on rock fails
- `TestTerrainPassability` - ✨ **NEW**: Direct passability tests (bounds, terrain, buildings)

**Game Mechanics (2 tests):**
- `TestFormationCalculation` - Formation positioning
- `TestUnitCollisionDetection` - Unit occupancy checks

### Scenario Tests (2 tests)

- `navigate_around_rock` - Single unit pathfinding
- `formation_around_cluster` - Multi-unit formation

---

## Design Decisions

### Removed: SVG Visualization (~400 lines)

**Why removed:**
1. **Tests should be self-explanatory** - Clear names and assertions > diagrams
2. **Maintenance burden** - SVG generation adds complexity
3. **JSON is readable** - Scenario files are declarative
4. **Explicit tests are better** - Unit tests verify constraints directly

**What was deleted:**
- `server/testutil/scenario_renderer.go` (210 lines)
- `server/cmd/scenario-viz/` (175 lines + CLI tool)
- `maps/scenarios/visuals/` (SVG files)
- `Visual` field from JSON schema

### Kept: Scenario Testing

**Why kept:**
- JSON scenarios are declarative and easy to write
- Automatic test execution with `go test`
- Comprehensive expectation checking
- No external dependencies

---

## Running Tests

```bash
cd server

# All tests
go test -v

# Unit tests only
go test -v -run "^Test(Pathfinding|Formation|Terrain|Unit|Cannot|Rock)"

# Scenario tests only
go test -v -run TestAllScenarios

# Specific test
go test -v -run TestCannotPathToRock
```

**Expected output:**
```
=== RUN   TestPathfindingAroundSingleRock
--- PASS: TestPathfindingAroundSingleRock (0.00s)
...
=== RUN   TestTerrainPassability
--- PASS: TestTerrainPassability (0.00s)
=== RUN   TestAllScenarios
=== RUN   TestAllScenarios/navigate_around_rock
--- PASS: TestAllScenarios/navigate_around_rock (0.00s)
PASS
ok      realtime-game-server    0.24s
```

---

## Creating Scenario Tests

Place JSON files in `maps/scenarios/*.json`:

```json
{
  "name": "My Test",
  "map": "test_single_rock.json",
  "description": "Tests unit movement",

  "setup": {
    "units": [
      {"id": "u1", "team": 0, "type": "worker", "position": [5, 5]}
    ]
  },

  "actions": [
    {
      "tick": 0,
      "type": "move",
      "unitIds": ["u1"],
      "target": [15, 5],
      "formation": "box"
    }
  ],

  "expectations": {
    "maxTicks": 100,
    "finalState": {
      "units": [
        {"id": "u1", "position": [15, 5], "state": "stopped"}
      ]
    },
    "constraints": {
      "noStacking": true,
      "allStopped": true
    }
  }
}
```

Tests auto-discover and run with `go test`.

---

## Files

**Test files:**
- `server/game_test.go` - 10 unit tests (~390 lines)
- `server/scenario_test.go` - Scenario test integration (~260 lines)
- `server/testutil/scenario.go` - JSON schema (~160 lines)
- `server/testutil/scenario_runner.go` - Execution engine (~370 lines)

**Test maps:**
- `maps/test_single_rock.json` - 20×10, one rock
- `maps/test_rock_cluster.json` - 20×15, 3×2 cluster
- `maps/test_corridor.json` - 20×10, narrow passage

**Scenario files:**
- `maps/scenarios/navigate_around_rock.json`
- `maps/scenarios/formation_around_cluster.json`

**Documentation:**
- `/TESTING.md` - User guide for running and creating tests
- `maps/scenarios/README.md` - Scenario directory guide

---

## Future Enhancements

- ⏳ More scenarios (combat, building, resources)
- ⏳ Performance benchmarks
- ⏳ Fuzzing tests for edge cases
- ⏳ Visual editor (only if complexity is justified)

---

## Summary

**Net changes:**
- ✅ +5 explicit negative tests ("can't go over rocks")
- ✅ +1 explicit path avoidance test
- ✅ -400 lines (removed visualization)
- ✅ Simplified, clearer, better coverage
