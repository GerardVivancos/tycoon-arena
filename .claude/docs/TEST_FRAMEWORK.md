# Declarative Visual Test Framework

**Status:** ✅ Phase 1 Complete (Schema, Renderer, CLI)
**Date:** 2025-10-13
**Goal:** JSON-based test scenarios with automatic SVG visualization

---

## Overview

Implemented a declarative test framework where scenarios are defined in JSON and automatically rendered as visual diagrams. This allows:
- **Writing tests without coding** - Pure JSON definitions
- **Visual verification** - See exactly what the test expects as a diagram
- **Fast iteration** - Change JSON, regenerate SVG instantly
- **Version control friendly** - JSON and SVG diff cleanly in git

---

## Architecture

```
JSON Scenario → Schema Loader → SVG Renderer → Visual Diagram
                     ↓
              Scenario Runner (TODO)
                     ↓
              Test Execution
```

### Implemented (Phase 1)
✅ **Schema & Loader** - Parse JSON scenarios into Go structs
✅ **SVG Renderer** - Generate visual diagrams from scenarios
✅ **CLI Tool** - Command-line tool to generate SVGs
✅ **Example Scenarios** - 2 working examples

### Pending (Phase 2)
⏳ **Scenario Runner** - Execute scenarios in test environment
⏳ **Test Integration** - Integrate with `go test`
⏳ **Result Comparison** - Compare expected vs actual outcomes

---

## JSON Scenario Format

### Complete Example
```json
{
  "name": "Navigate Around Single Rock",
  "map": "test_single_rock.json",
  "description": "Verifies pathfinding routes around obstacle",

  "setup": {
    "units": [
      {
        "id": "unit1",
        "team": 0,
        "type": "worker",
        "position": [5, 5],
        "label": "A"
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
          "state": "stopped",
          "label": "A'"
        }
      ]
    },
    "constraints": {
      "pathMustAvoid": [[10, 5]],
      "noStacking": true,
      "pathExists": true
    }
  },

  "visual": {
    "annotations": [
      {
        "type": "arrow",
        "from": [5, 5],
        "to": [15, 5],
        "style": "expected-path"
      }
    ]
  }
}
```

---

## Schema Components

### TestScenario (Root)
- `name` - Descriptive name for the test
- `map` - Which map file to use
- `description` - What the test verifies
- `setup` - Initial state
- `actions` - Commands to execute
- `expectations` - What should happen
- `visual` - Annotations for diagram

### ScenarioSetup
- `units[]` - Initial unit positions and properties
- `buildings[]` - Initial buildings (optional)

### ScenarioAction
- `tick` - When to execute (0 = immediately)
- `type` - "move", "build", "attack"
- `unitIds[]` - Which units to command
- `target` - Target tile position
- `formation` - Formation type (box/line/spread)

### ScenarioExpectations
- `maxTicks` - Maximum simulation time
- `finalState` - Expected end positions
- `constraints` - Additional verifications

### Constraints
- `pathMustAvoid` - Tiles path should not traverse
- `noStacking` - No two units on same tile
- `pathExists` - Path should (or shouldn't) exist
- `allStopped` - All units should have stopped
- `formationShape` - Expected formation type

### Visual Annotations
- `arrow` - Draw arrow from/to
- `marker` - Mark a specific position
- `circle` - Highlight an area
- `text` - Add explanatory text

---

## SVG Output

### Visual Elements

**Colors:**
- 🔵 Blue circles - Initial positions
- 🟢 Green circles - Expected final positions
- ⬜ Gray squares - Obstacles (rocks)
- 🟠 Orange dashed lines - Expected paths
- ⬜ Light gray - Grid tiles

**Legend:**
- All diagrams include a legend explaining symbols
- Title shows scenario name
- Description shows what's being tested

**Output Size:**
- ~20KB per SVG file
- Scalable vector graphics (zoom without quality loss)
- Can be viewed in any web browser

---

## Implementation

### 1. Schema (`server/testutil/scenario.go` - 180 lines)

**Key Types:**
```go
type TestScenario struct {
    Name         string
    Map          string
    Description  string
    Setup        ScenarioSetup
    Actions      []ScenarioAction
    Expectations ScenarioExpectations
    Visual       *ScenarioVisual
}

type ScenarioUnit struct {
    ID       string
    Team     int
    Type     string
    Position [2]int
    Label    string
}

type ExpectedUnit struct {
    ID           string
    Position     *[2]int  // Exact position
    PositionNear *[2]int  // Approximate position
    Tolerance    int      // Distance tolerance
    State        string   // "stopped", "moving"
}
```

**Functions:**
- `LoadScenario(path) → *TestScenario` - Parse JSON file
- `Validate() → error` - Check scenario is valid
- `GetUnitByID(id) → *ScenarioUnit` - Lookup helpers

### 2. Renderer (`server/testutil/scenario_renderer.go` - 200 lines)

**Function:** `RenderScenarioSVG(scenario, mapData) → string`

**Process:**
1. Calculate SVG dimensions from map size
2. Draw grid background
3. Draw terrain (rocks) - TODO: integrate with actual map data
4. Draw initial unit positions (blue circles with labels)
5. Draw expected final positions (green circles with labels)
6. Draw visual annotations (arrows, markers)
7. Draw legend at bottom
8. Return complete SVG as string

**Constants:**
- `tileSizePx = 40` - SVG pixels per game tile
- `unitRadius = 12` - Circle radius for units
- `marginPx = 50` - Border around diagram
- `legendHeight = 60` - Space for legend

### 3. CLI Tool (`server/cmd/scenario-viz/main.go` - 110 lines)

**Usage:**
```bash
# Generate single scenario
go run main.go --scenario=my_test.json

# Generate all scenarios
go run main.go --all

# Specify output directory
go run main.go --all --output=../../some/path
```

**Flags:**
- `--scenario=<file>` - Specific scenario to render
- `--all` - Render all scenarios in maps/scenarios/
- `--output=<dir>` - Where to save SVGs (default: maps/scenarios/visuals/)

**Output:**
```
Found 2 scenarios

Rendering: Navigate Around Single Rock
  ✓ maps/scenarios/visuals/navigate_around_rock.svg
Rendering: Formation Around Rock Cluster
  ✓ maps/scenarios/visuals/formation_around_cluster.svg

Done! All SVGs saved to: maps/scenarios/visuals
```

---

## Example Scenarios

### 1. Navigate Around Rock (`navigate_around_rock.json`)
**Purpose:** Verify pathfinding routes around single obstacle

**Setup:**
- 1 unit at (5, 5)
- Rock at (10, 5)

**Action:**
- Move unit to (15, 5)

**Expectations:**
- Unit reaches (15, 5)
- Path avoids rock at (10, 5)
- No stacking
- Path exists

### 2. Formation Around Cluster (`formation_around_cluster.json`)
**Purpose:** Verify formation adapts when blocked by obstacles

**Setup:**
- 5 units at (2, 5-9)
- Rock cluster at (9-11, 7-8)

**Action:**
- Move all 5 units to (10, 7) in box formation

**Expectations:**
- All units end near (10, 7) within 3 tiles
- No stacking
- All stopped
- Box formation maintained

---

## Directory Structure

```
maps/
├── scenarios/              # JSON scenario definitions
│   ├── navigate_around_rock.json
│   ├── formation_around_cluster.json
│   └── visuals/           # Generated SVG diagrams
│       ├── navigate_around_rock.svg
│       └── formation_around_cluster.svg
├── test_single_rock.json  # Test map files
├── test_rock_cluster.json
└── default.json

server/
├── testutil/
│   ├── scenario.go         # Schema + JSON loader
│   ├── scenario_renderer.go # SVG generation
│   ├── scenario_runner.go  # TODO: Execution engine
│   ├── test_server.go      # Test utilities
│   └── assertions.go       # Test assertions
├── scenario_test.go        # TODO: go test integration
└── cmd/
    └── scenario-viz/
        └── main.go         # CLI tool
```

---

## Workflow

### Creating a New Test Scenario

1. **Write JSON** (`maps/scenarios/my_test.json`)
```json
{
  "name": "My Test",
  "map": "test_map.json",
  "setup": { "units": [...] },
  "actions": [...],
  "expectations": {...}
}
```

2. **Generate Visual**
```bash
cd server/cmd/scenario-viz
go run main.go --scenario=my_test.json
```

3. **View SVG**
```bash
open ../../../maps/scenarios/visuals/my_test.svg
```

4. **Verify** - Does the diagram match your intent?

5. **Iterate** - Adjust JSON, regenerate SVG

6. **Run Test** (when runner implemented)
```bash
cd server
go test -v -run TestScenario/my_test
```

---

## Future: Scenario Runner (Phase 2)

### Planned Implementation

**Function:** `RunScenario(scenario) → ScenarioResult`

**Process:**
1. Load map file
2. Create GameServer instance
3. Add units/buildings from setup
4. Execute actions at specified ticks
5. Run simulation for maxTicks
6. Check all expectations and constraints
7. Return pass/fail + violations list

**Result Type:**
```go
type ScenarioResult struct {
    Passed        bool
    Violations    []string
    FinalState    ActualState
    ExecutionTime time.Duration
}
```

### Test Integration

**Function:** `TestAllScenarios(t *testing.T)`

```go
func TestAllScenarios(t *testing.T) {
    scenarioFiles, _ := filepath.Glob("../maps/scenarios/*.json")

    for _, file := range scenarioFiles {
        t.Run(filepath.Base(file), func(t *testing.T) {
            scenario, _ := LoadScenario(file)
            result, _ := RunScenario(scenario)

            if !result.Passed {
                t.Errorf("Violations:\n%s",
                    strings.Join(result.Violations, "\n"))
            }
        })
    }
}
```

**Run:** `go test -v -run TestAllScenarios`

---

## Future: Visual Editor (Long-term Goal)

### Vision

A GUI tool to create scenarios visually:

1. **Map Editor Mode**
   - Load existing map or create new
   - Place units by clicking
   - Add buildings
   - Mark obstacles

2. **Action Editor**
   - Timeline view
   - Add move/build/attack commands
   - Specify tick timing
   - Set formation types

3. **Expectation Editor**
   - Drag expected end positions
   - Draw constraint areas (pathMustAvoid)
   - Toggle constraint checkboxes

4. **Preview**
   - Real-time SVG preview as you edit
   - Validate scenario
   - Export to JSON

5. **Test Integration**
   - Run scenario directly from editor
   - Show pass/fail results
   - Visualize actual vs expected (diff view)

### Technology Options

- **Godot-based:** Use same engine as game client
- **Web-based:** HTML5 canvas + React/Vue
- **Standalone:** Dear ImGui + OpenGL

---

## Benefits

✅ **No coding required** - Write tests in JSON
✅ **Visual verification** - See exactly what you're testing
✅ **Human readable** - Anyone can understand scenarios
✅ **Version controlled** - JSON + SVG diff cleanly
✅ **Fast iteration** - Change JSON, regenerate instantly
✅ **Documentation** - SVGs are living documentation
✅ **Shareable** - Send screenshot to explain test

---

## Current Status

### Completed
- ✅ JSON schema definition
- ✅ Scenario loader with validation
- ✅ SVG renderer with visual output
- ✅ CLI tool for generating diagrams
- ✅ 2 example scenarios
- ✅ All code documented and tested

### Remaining (Phase 2)
- ⏳ Scenario runner (execute scenarios)
- ⏳ Expectation verification
- ⏳ Test integration with `go test`
- ⏳ More example scenarios

### Future (Phase 3+)
- 🔮 Visual editor for creating scenarios
- 🔮 Diff view (expected vs actual)
- 🔮 Interactive playback of test execution
- 🔮 Animated SVG output (show movement over time)

---

## Summary

Created a declarative test framework that bridges the gap between technical testing and human understanding. Tests are now:
- **Visible** - Rendered as diagrams
- **Accessible** - Written in JSON, not code
- **Documented** - SVGs serve as visual specification
- **Verified** - Will execute automatically when runner implemented

**Key Innovation:** Tests are their own documentation. Looking at an SVG instantly shows what behavior is expected, making the test suite approachable for non-programmers while maintaining technical rigor.
