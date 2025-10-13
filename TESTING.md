# Testing Guide

This project uses both unit tests and declarative JSON scenario tests. All tests run with standard Go testing tools.

## Quick Start

```bash
# Run all tests
cd server
go test -v

# Run only unit tests
go test -v -run "^Test(Pathfinding|Formation|Terrain|Unit|Cannot|Rock)"

# Run only scenario tests
go test -v -run TestAllScenarios

# Run a specific scenario
go test -v -run TestAllScenarios/navigate_around_rock
```

---

## Test Coverage

### Unit Tests (10 tests)

**Pathfinding:**
- `TestPathfindingAroundSingleRock` - Routes around 1 obstacle
- `TestPathfindingAroundCluster` - Navigates 3×2 rock cluster
- `TestPathfindingNoPath` - Returns nil for unreachable destination
- `TestPathDoesNotGoThroughRock` - Verifies path avoids rocks

**Terrain & Obstacles:**
- `TestCannotPathToRock` - Pathfinding TO a rock returns nil
- `TestCannotMoveToRock` - Unit command to rock is rejected
- `TestRockBlocksBuilding` - Building placement on rock fails
- `TestTerrainPassability` - Direct isTilePassable tests

**Game Mechanics:**
- `TestFormationCalculation` - Formation positioning
- `TestUnitCollisionDetection` - Unit occupancy checks

### Scenario Tests (2 tests)

**JSON-based declarative tests:**
- `navigate_around_rock` - Single unit pathfinds around obstacle
- `formation_around_cluster` - Five units form up near blocked area

---

## Running Tests

### All Tests

```bash
cd server
go test -v
```

**Expected output:**
```
=== RUN   TestPathfindingAroundSingleRock
--- PASS: TestPathfindingAroundSingleRock (0.00s)
...
=== RUN   TestAllScenarios
=== RUN   TestAllScenarios/navigate_around_rock
--- PASS: TestAllScenarios/navigate_around_rock (0.00s)
PASS
ok      realtime-game-server    0.3s
```

### Specific Tests

**Unit tests:**
```bash
# Pathfinding tests
go test -v -run TestPathfinding

# Terrain tests
go test -v -run "TestCannot|TestRock|TestTerrain"

# Single test
go test -v -run TestCannotPathToRock
```

**Scenario tests:**
```bash
# All scenarios
go test -v -run TestAllScenarios

# Single scenario
go test -v -run TestAllScenarios/navigate_around_rock
```

---

## Creating Test Scenarios

Test scenarios are JSON files that define test cases declaratively.

### Location

Place scenarios in: `maps/scenarios/*.json`

### Example Scenario

```json
{
  "name": "My Test",
  "map": "test_single_rock.json",
  "description": "Tests basic unit movement",

  "setup": {
    "units": [
      {
        "id": "unit1",
        "team": 0,
        "type": "worker",
        "position": [5, 5]
      }
    ]
  },

  "actions": [
    {
      "tick": 0,
      "type": "move",
      "unitIds": ["unit1"],
      "target": [15, 5],
      "formation": "box"
    }
  ],

  "expectations": {
    "maxTicks": 100,
    "finalState": {
      "units": [
        {
          "id": "unit1",
          "position": [15, 5],
          "state": "stopped"
        }
      ]
    },
    "constraints": {
      "noStacking": true,
      "allStopped": true
    }
  }
}
```

### Schema Reference

**Required fields:**
- `name` - Test name
- `map` - Map file (relative to `maps/`)
- `setup.units` - Initial units
- `actions` - Commands to execute
- `expectations.maxTicks` - Max simulation time
- `expectations.finalState` - Expected end state

**Unit definition:**
```json
{
  "id": "unit1",           // Unique identifier
  "team": 0,               // Team number
  "type": "worker",        // Unit type
  "position": [10, 5]      // [x, y] tile coordinates
}
```

**Actions:**
```json
{
  "tick": 0,               // When to execute (0 = immediately)
  "type": "move",          // "move", "build", "attack"
  "unitIds": ["unit1"],    // Which units
  "target": [15, 5],       // Target position
  "formation": "box"       // "box", "line", "spread"
}
```

**Expectations - Exact position:**
```json
{
  "id": "unit1",
  "position": [15, 5],     // Must be exactly here
  "state": "stopped"
}
```

**Expectations - Near position (with tolerance):**
```json
{
  "id": "unit1",
  "positionNear": [15, 5], // Within tolerance
  "tolerance": 3,          // Distance in tiles
  "state": "stopped"
}
```

**Constraints:**
- `pathMustAvoid` - Tiles path must not traverse: `[[10, 5]]`
- `noStacking` - No two units on same tile
- `allStopped` - All units stopped at end
- `pathExists` - Whether path should be found

---

## Test Workflow

### 1. Write the Test

Create `maps/scenarios/my_test.json`:

```json
{
  "name": "Test Movement",
  "map": "test_single_rock.json",
  "setup": {
    "units": [{"id": "u1", "team": 0, "type": "worker", "position": [2, 2]}]
  },
  "actions": [
    {"tick": 0, "type": "move", "unitIds": ["u1"], "target": [18, 2], "formation": "box"}
  ],
  "expectations": {
    "maxTicks": 150,
    "finalState": {
      "units": [{"id": "u1", "position": [18, 2], "state": "stopped"}]
    },
    "constraints": {
      "noStacking": true,
      "allStopped": true
    }
  }
}
```

### 2. Run the Test

```bash
cd server
go test -v -run TestAllScenarios/my_test
```

### 3. Iterate

- Test fails? Check expectations or fix code
- Test passes? Commit the scenario

---

## Tips & Best Practices

### Naming

**Scenarios:** `<action>_<condition>.json`
- Good: `navigate_around_rock.json`, `formation_blocked.json`
- Bad: `test1.json`, `my_scenario.json`

**Unit IDs:** Use descriptive names
- Good: `"unit1"`, `"scout_a"`, `"worker1"`
- Bad: `"u"`, `"1"`, `"x"`

### Tolerance Values

Use `positionNear` + `tolerance` when:
- Formation positions may vary
- Obstacles force alternate paths
- Multiple valid end positions exist

**Guidelines:**
- Tolerance 1-2: Tight positioning
- Tolerance 3-4: Formations around obstacles
- Tolerance 5+: Very loose (rarely needed)

### maxTicks Calculation

Based on:
- Distance to travel (tiles)
- Movement speed: 4 tiles/sec at 20Hz = 0.2 tiles/tick
- Formula: `ticks ≈ distance / 0.2 × 1.5` (1.5× for pathfinding)

**Examples:**
- 10 tiles: ~75 ticks
- 20 tiles: ~150 ticks
- 30 tiles: ~225 ticks

### Common Pitfalls

**❌ Exact position when path varies:**
```json
"position": [15, 5]  // Fails if obstacle forces different path
```
**✅ Use tolerance:**
```json
"positionNear": [15, 5],
"tolerance": 2
```

**❌ Not enough time:**
```json
"maxTicks": 50  // Unit doesn't reach destination
```
**✅ Calculate appropriately:**
```json
"maxTicks": 100  // Enough time for 20-tile journey
```

---

## Available Test Maps

Located in `maps/`:

| Map | Size | Description |
|-----|------|-------------|
| `default.json` | 40×30 | Main game map with 7 rocks |
| `test_single_rock.json` | 20×10 | One rock at center |
| `test_rock_cluster.json` | 20×15 | 3×2 rock cluster |
| `test_corridor.json` | 20×10 | Narrow 1-tile corridor |

---

## Troubleshooting

### Test fails with "position mismatch"

**Cause:** Unit didn't reach expected position

**Solutions:**
1. Increase `maxTicks`
2. Use `positionNear` + `tolerance` instead of exact `position`
3. Check if path is blocked by obstacles

### Test fails with "still moving"

**Cause:** `allStopped` constraint but units haven't finished

**Solutions:**
1. Increase `maxTicks`
2. Remove `allStopped` if not needed
3. Check for path blockages causing rerouting loops

### Scenario not found

**Cause:** File not in correct location

**Solutions:**
1. Verify file is in `maps/scenarios/*.json`
2. Check filename has `.json` extension
3. Run `go test -v` to see discovered scenarios

---

## Further Reading

- **Technical details:** [.claude/docs/TEST_FRAMEWORK.md](.claude/docs/TEST_FRAMEWORK.md)
- **Current state:** [.claude/docs/CURRENT_STATE.md](.claude/docs/CURRENT_STATE.md)
- **Pathfinding:** [.claude/docs/PATHFINDING_IMPLEMENTATION.md](.claude/docs/PATHFINDING_IMPLEMENTATION.md)
