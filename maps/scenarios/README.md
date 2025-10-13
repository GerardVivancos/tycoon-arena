# Test Scenarios

This directory contains declarative test scenarios in JSON format. Each scenario defines a test case that automatically runs with `go test`.

## 📁 Directory Structure

```
maps/scenarios/
├── README.md                           # This file
├── navigate_around_rock.json           # Pathfinding around single obstacle
└── formation_around_cluster.json       # Formation near blocked area
```

---

## 🧪 Existing Scenarios

### 1. Navigate Around Single Rock
**File:** `navigate_around_rock.json`
**Map:** `test_single_rock.json` (20×10)
**Description:** Verifies that pathfinding correctly routes around a single obstacle

**Setup:**
- 1 worker at (5, 5)
- 1 rock obstacle at (10, 5)

**Action:**
- Move worker to (15, 5)

**Expected:**
- Worker reaches (15, 5)
- Path avoids rock at (10, 5)
- Worker stops moving
- No stacking

**Run:** `cd server && go test -v -run TestAllScenarios/navigate_around_rock`

---

### 2. Formation Around Rock Cluster
**File:** `formation_around_cluster.json`
**Map:** `test_rock_cluster.json` (20×15)
**Description:** Verifies that formations adapt when the target area is blocked by obstacles

**Setup:**
- 5 workers at (2, 5-9) in vertical line
- 3×2 rock cluster at (9-11, 7-8)

**Action:**
- Move all 5 workers to (10, 7) in box formation

**Expected:**
- All workers end within 4 tiles of (10, 7)
- No stacking
- All stopped
- Box formation maintained

**Run:** `cd server && go test -v -run TestAllScenarios/formation_around_cluster`

---

## ✨ Creating New Scenarios

See the main [TESTING.md](../../TESTING.md) guide for full instructions.

### Quick Template

```json
{
  "name": "My Test Name",
  "map": "test_single_rock.json",
  "description": "What this test verifies",

  "setup": {
    "units": [
      {"id": "unit1", "team": 0, "type": "worker", "position": [5, 5]}
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
        {"id": "unit1", "position": [15, 5], "state": "stopped"}
      ]
    },
    "constraints": {
      "noStacking": true,
      "allStopped": true
    }
  }
}
```

---

## 🏃 Running Scenarios

```bash
cd server

# Run all scenario tests
go test -v -run TestAllScenarios

# Run specific scenario
go test -v -run TestAllScenarios/navigate_around_rock
```

---

## 📝 JSON Schema Quick Reference

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Test name |
| `map` | string | Map file (relative to `maps/`) |
| `setup.units` | array | Initial units |
| `actions` | array | Commands to execute |
| `expectations.maxTicks` | number | Max simulation time |
| `expectations.finalState` | object | Expected end state |

### Unit Definition

```json
{
  "id": "unit1",              // Unique identifier
  "team": 0,                  // Team number (0, 1, 2...)
  "type": "worker",           // Unit type
  "position": [10, 5]         // [x, y] in tile coordinates
}
```

### Action Types

**Move:**
```json
{
  "tick": 0,
  "type": "move",
  "unitIds": ["unit1", "unit2"],
  "target": [15, 5],
  "formation": "box"          // "box", "line", "spread"
}
```

### Expectations

**Exact position:**
```json
{
  "id": "unit1",
  "position": [15, 5],        // Must be exactly here
  "state": "stopped"
}
```

**Near position (with tolerance):**
```json
{
  "id": "unit1",
  "positionNear": [15, 5],    // Within tolerance
  "tolerance": 3,             // Distance in tiles
  "state": "stopped"
}
```

### Constraints

| Constraint | Type | Description |
|------------|------|-------------|
| `pathMustAvoid` | array | Tiles path must not traverse `[[x, y], ...]` |
| `noStacking` | boolean | No two units on same tile |
| `allStopped` | boolean | All units stopped at end |
| `pathExists` | boolean | Path should be found |

---

## 🎯 Test Naming Conventions

**Format:** `<action>_<condition>.json`

**Examples:**
- `navigate_around_rock.json` ✅
- `formation_around_cluster.json` ✅
- `attack_building.json` ✅
- `test1.json` ❌ (not descriptive)

---

## 💡 Tips

1. **Start simple** - Test one thing at a time
2. **Use tolerance** - Exact positions are fragile with pathfinding
3. **Run frequently** - Tests are fast, run them often
4. **Check the map** - Make sure your test map has the terrain you expect
5. **Descriptive names** - Unit IDs and test names should be clear

---

## 📚 Further Reading

- **Full Testing Guide:** [../../TESTING.md](../../TESTING.md)
- **Technical Details:** [../../.claude/docs/TEST_FRAMEWORK.md](../../.claude/docs/TEST_FRAMEWORK.md)
- **Available Maps:** [../](../)
