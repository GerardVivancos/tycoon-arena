package testutil

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// ScenarioResult contains the result of running a scenario
type ScenarioResult struct {
	Passed        bool
	Violations    []string
	FinalState    *ActualState
	ExecutionTime int // in ticks
}

// ActualState captures the actual state after scenario execution
type ActualState struct {
	Units     map[string]ActualUnit     // keyed by unit ID
	Buildings map[string]ActualBuilding // keyed by building ID
}

// ActualUnit represents the actual state of a unit
type ActualUnit struct {
	ID       string
	Position [2]int
	State    string // "stopped" or "moving"
	Path     [][2]int // Path tiles the unit visited (for pathMustAvoid check)
}

// ActualBuilding represents the actual state of a building
type ActualBuilding struct {
	ID       string
	Position [2]int
	Exists   bool
}

// RunScenario executes a test scenario and returns the result
// This requires access to main package types, so we'll define an interface
func RunScenario(scenario *TestScenario, gameServer GameServerInterface) (*ScenarioResult, error) {
	if scenario == nil {
		return nil, fmt.Errorf("scenario is nil")
	}

	result := &ScenarioResult{
		Passed:     true,
		Violations: []string{},
		FinalState: &ActualState{
			Units:     make(map[string]ActualUnit),
			Buildings: make(map[string]ActualBuilding),
		},
	}

	// Load map
	mapPath := scenario.Map
	if !filepath.IsAbs(mapPath) {
		// Relative paths are relative to maps/ directory
		mapPath = filepath.Join("../maps", mapPath)
	}

	if err := gameServer.LoadMap(mapPath); err != nil {
		return nil, fmt.Errorf("failed to load map: %w", err)
	}

	// Spawn units from setup
	unitIDMap := make(map[string]uint32) // scenario ID -> game entity ID
	for _, unit := range scenario.Setup.Units {
		entityID := gameServer.SpawnUnit(unit.Type, unit.Team, unit.Position[0], unit.Position[1])
		unitIDMap[unit.ID] = entityID
	}

	// Spawn buildings from setup
	buildingIDMap := make(map[string]uint32) // scenario ID -> game entity ID
	for _, building := range scenario.Setup.Buildings {
		entityID := gameServer.SpawnBuilding(building.Type, building.Team, building.Position[0], building.Position[1])
		buildingIDMap[building.ID] = entityID
	}

	// Track paths for pathMustAvoid constraint
	unitPaths := make(map[string][][2]int)
	for id := range unitIDMap {
		unitPaths[id] = [][2]int{}
	}

	// Run simulation
	for tick := 0; tick < scenario.Expectations.MaxTicks; tick++ {
		// Execute actions scheduled for this tick
		for _, action := range scenario.Actions {
			if action.Tick == tick {
				if err := executeAction(action, unitIDMap, buildingIDMap, gameServer); err != nil {
					return nil, fmt.Errorf("failed to execute action at tick %d: %w", tick, err)
				}
			}
		}

		// Advance game state
		gameServer.Tick()

		// Record unit positions for path tracking
		for scenarioID, entityID := range unitIDMap {
			pos := gameServer.GetEntityPosition(entityID)
			if pos != nil {
				// Only add if position changed
				if len(unitPaths[scenarioID]) == 0 || unitPaths[scenarioID][len(unitPaths[scenarioID])-1] != *pos {
					unitPaths[scenarioID] = append(unitPaths[scenarioID], *pos)
				}
			}
		}
	}

	result.ExecutionTime = scenario.Expectations.MaxTicks

	// Capture final state
	for scenarioID, entityID := range unitIDMap {
		pos := gameServer.GetEntityPosition(entityID)
		isMoving := gameServer.IsEntityMoving(entityID)

		state := "stopped"
		if isMoving {
			state = "moving"
		}

		if pos != nil {
			result.FinalState.Units[scenarioID] = ActualUnit{
				ID:       scenarioID,
				Position: *pos,
				State:    state,
				Path:     unitPaths[scenarioID],
			}
		}
	}

	for scenarioID, entityID := range buildingIDMap {
		pos := gameServer.GetEntityPosition(entityID)
		exists := gameServer.EntityExists(entityID)

		if pos != nil {
			result.FinalState.Buildings[scenarioID] = ActualBuilding{
				ID:       scenarioID,
				Position: *pos,
				Exists:   exists,
			}
		}
	}

	// Verify expectations
	violations := VerifyExpectations(scenario, result.FinalState)
	result.Violations = violations
	result.Passed = len(violations) == 0

	return result, nil
}

// executeAction executes a scenario action
func executeAction(action ScenarioAction, unitIDMap, buildingIDMap map[string]uint32, gameServer GameServerInterface) error {
	switch action.Type {
	case "move":
		// Convert scenario unit IDs to entity IDs
		entityIDs := []uint32{}
		for _, scenarioID := range action.UnitIDs {
			if entityID, ok := unitIDMap[scenarioID]; ok {
				entityIDs = append(entityIDs, entityID)
			} else {
				return fmt.Errorf("unknown unit ID: %s", scenarioID)
			}
		}

		formation := action.Formation
		if formation == "" {
			formation = "box" // default
		}

		return gameServer.MoveUnits(entityIDs, action.Target[0], action.Target[1], formation)

	case "build":
		// For build actions, we need the unit to execute it
		// For now, just spawn the building directly (simplified)
		if len(action.UnitIDs) == 0 {
			return fmt.Errorf("build action requires at least one unit")
		}

		// Get the team from the first unit
		firstUnitID := unitIDMap[action.UnitIDs[0]]
		team := gameServer.GetEntityTeam(firstUnitID)

		gameServer.SpawnBuilding(action.BuildingType, team, action.Target[0], action.Target[1])
		return nil

	case "attack":
		// Convert unit IDs
		entityIDs := []uint32{}
		for _, scenarioID := range action.UnitIDs {
			if entityID, ok := unitIDMap[scenarioID]; ok {
				entityIDs = append(entityIDs, entityID)
			}
		}

		// Convert target ID
		var targetID uint32
		if buildingEntityID, ok := buildingIDMap[action.TargetID]; ok {
			targetID = buildingEntityID
		} else if unitEntityID, ok := unitIDMap[action.TargetID]; ok {
			targetID = unitEntityID
		} else {
			return fmt.Errorf("unknown target ID: %s", action.TargetID)
		}

		return gameServer.AttackTarget(entityIDs, targetID)

	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// VerifyExpectations checks if the final state matches expectations
func VerifyExpectations(scenario *TestScenario, actualState *ActualState) []string {
	violations := []string{}

	// Check unit positions
	for _, expected := range scenario.Expectations.FinalState.Units {
		actual, exists := actualState.Units[expected.ID]
		if !exists {
			violations = append(violations, fmt.Sprintf("Unit %s does not exist in final state", expected.ID))
			continue
		}

		// Check exact position
		if expected.Position != nil {
			if actual.Position != *expected.Position {
				violations = append(violations, fmt.Sprintf(
					"Unit %s position mismatch: expected (%d,%d), got (%d,%d)",
					expected.ID,
					(*expected.Position)[0], (*expected.Position)[1],
					actual.Position[0], actual.Position[1],
				))
			}
		}

		// Check position near (with tolerance)
		if expected.PositionNear != nil {
			tolerance := expected.Tolerance
			if tolerance == 0 {
				tolerance = 1 // default tolerance
			}

			distance := manhattanDistance(actual.Position, *expected.PositionNear)
			if distance > tolerance {
				violations = append(violations, fmt.Sprintf(
					"Unit %s not near expected position: expected within %d of (%d,%d), got (%d,%d) (distance %d)",
					expected.ID,
					tolerance,
					(*expected.PositionNear)[0], (*expected.PositionNear)[1],
					actual.Position[0], actual.Position[1],
					distance,
				))
			}
		}

		// Check state
		if expected.State != "" && actual.State != expected.State {
			violations = append(violations, fmt.Sprintf(
				"Unit %s state mismatch: expected %s, got %s",
				expected.ID, expected.State, actual.State,
			))
		}
	}

	// Check building expectations
	for _, expected := range scenario.Expectations.FinalState.Buildings {
		actual, exists := actualState.Buildings[expected.ID]
		if !exists {
			violations = append(violations, fmt.Sprintf("Building %s does not exist in final state", expected.ID))
			continue
		}

		if actual.Exists != expected.Exists {
			if expected.Exists {
				violations = append(violations, fmt.Sprintf("Building %s should exist but does not", expected.ID))
			} else {
				violations = append(violations, fmt.Sprintf("Building %s should not exist but does", expected.ID))
			}
		}
	}

	// Check constraints
	if scenario.Expectations.Constraints != nil {
		constraints := scenario.Expectations.Constraints

		// PathMustAvoid - check that units didn't go through forbidden tiles
		if len(constraints.PathMustAvoid) > 0 {
			forbiddenSet := make(map[string]bool)
			for _, pos := range constraints.PathMustAvoid {
				key := fmt.Sprintf("%d,%d", pos[0], pos[1])
				forbiddenSet[key] = true
			}

			for unitID, actual := range actualState.Units {
				for _, pos := range actual.Path {
					key := fmt.Sprintf("%d,%d", pos[0], pos[1])
					if forbiddenSet[key] {
						violations = append(violations, fmt.Sprintf(
							"Unit %s path went through forbidden tile (%d,%d)",
							unitID, pos[0], pos[1],
						))
						break // Only report once per unit
					}
				}
			}
		}

		// NoStacking - check that no two units are on the same tile
		if constraints.NoStacking {
			positionCounts := make(map[string][]string) // position -> list of unit IDs
			for unitID, actual := range actualState.Units {
				key := fmt.Sprintf("%d,%d", actual.Position[0], actual.Position[1])
				positionCounts[key] = append(positionCounts[key], unitID)
			}

			for pos, unitIDs := range positionCounts {
				if len(unitIDs) > 1 {
					violations = append(violations, fmt.Sprintf(
						"Units stacked at %s: %s",
						pos, strings.Join(unitIDs, ", "),
					))
				}
			}
		}

		// AllStopped - check that all units have stopped moving
		if constraints.AllStopped {
			for unitID, actual := range actualState.Units {
				if actual.State != "stopped" {
					violations = append(violations, fmt.Sprintf(
						"Unit %s is still moving (expected all stopped)",
						unitID,
					))
				}
			}
		}

		// PathExists - checked by whether units reached their destination
		// This is implicitly checked by position verification

		// FormationShape - this would require more complex shape detection
		// For now, we'll skip this as it's an advanced feature
		if constraints.FormationShape != "" {
			// TODO: Implement formation shape detection
		}
	}

	return violations
}

// manhattanDistance calculates Manhattan distance between two points
func manhattanDistance(a, b [2]int) int {
	return int(math.Abs(float64(a[0]-b[0])) + math.Abs(float64(a[1]-b[1])))
}

// GameServerInterface defines the interface for interacting with the game server
// This allows us to test without depending on the main package directly
type GameServerInterface interface {
	LoadMap(path string) error
	SpawnUnit(unitType string, team int, x, y int) uint32
	SpawnBuilding(buildingType string, team int, x, y int) uint32
	Tick()
	GetEntityPosition(entityID uint32) *[2]int
	GetEntityTeam(entityID uint32) int
	IsEntityMoving(entityID uint32) bool
	EntityExists(entityID uint32) bool
	MoveUnits(entityIDs []uint32, targetX, targetY int, formation string) error
	AttackTarget(entityIDs []uint32, targetID uint32) error
}
