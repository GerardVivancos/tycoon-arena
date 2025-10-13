package testutil

import (
	"encoding/json"
	"fmt"
	"os"
)

// TestScenario represents a complete test scenario defined in JSON
type TestScenario struct {
	Name         string                `json:"name"`
	Map          string                `json:"map"`
	Description  string                `json:"description"`
	Setup        ScenarioSetup         `json:"setup"`
	Actions      []ScenarioAction      `json:"actions"`
	Expectations ScenarioExpectations  `json:"expectations"`
}

// ScenarioSetup defines initial state of the scenario
type ScenarioSetup struct {
	Units     []ScenarioUnit     `json:"units"`
	Buildings []ScenarioBuilding `json:"buildings,omitempty"`
}

// ScenarioUnit defines a unit in the scenario
type ScenarioUnit struct {
	ID       string `json:"id"`
	Team     int    `json:"team"`
	Type     string `json:"type"` // "worker", "player"
	Position [2]int `json:"position"` // [x, y]
	Label    string `json:"label,omitempty"`
}

// ScenarioBuilding defines a building in the scenario
type ScenarioBuilding struct {
	ID       string `json:"id"`
	Team     int    `json:"team"`
	Type     string `json:"type"` // "generator"
	Position [2]int `json:"position"` // [x, y]
	Label    string `json:"label,omitempty"`
}

// ScenarioAction defines an action to perform during the scenario
type ScenarioAction struct {
	Tick      int      `json:"tick"`      // When to execute
	Type      string   `json:"type"`      // "move", "build", "attack"
	UnitIDs   []string `json:"unitIds,omitempty"`
	Target    [2]int   `json:"target,omitempty"`
	Formation string   `json:"formation,omitempty"` // "box", "line", "spread"

	// For build actions
	BuildingType string `json:"buildingType,omitempty"`

	// For attack actions
	TargetID string `json:"targetId,omitempty"`
}

// ScenarioExpectations defines what should happen
type ScenarioExpectations struct {
	MaxTicks    int          `json:"maxTicks"`    // Maximum ticks to run
	FinalState  FinalState   `json:"finalState"`  // Expected end state
	Constraints *Constraints `json:"constraints,omitempty"`
}

// FinalState defines expected state at end of scenario
type FinalState struct {
	Units     []ExpectedUnit     `json:"units"`
	Buildings []ExpectedBuilding `json:"buildings,omitempty"`
}

// ExpectedUnit defines expected state of a unit
type ExpectedUnit struct {
	ID           string  `json:"id"`
	Position     *[2]int `json:"position,omitempty"`     // Exact position
	PositionNear *[2]int `json:"positionNear,omitempty"` // Approximate position
	Tolerance    int     `json:"tolerance,omitempty"`    // Tolerance for PositionNear
	State        string  `json:"state,omitempty"`        // "stopped", "moving"
	Label        string  `json:"label,omitempty"`
}

// ExpectedBuilding defines expected state of a building
type ExpectedBuilding struct {
	ID       string `json:"id"`
	Position [2]int `json:"position"`
	Exists   bool   `json:"exists"` // false = should be destroyed
}

// Constraints defines additional constraints to verify
type Constraints struct {
	PathMustAvoid  [][2]int `json:"pathMustAvoid,omitempty"`  // Positions path must not go through
	NoStacking     bool     `json:"noStacking,omitempty"`     // No units on same tile
	PathExists     *bool    `json:"pathExists,omitempty"`     // Path should exist (true) or not (false)
	AllStopped     bool     `json:"allStopped,omitempty"`     // All units should have stopped
	FormationShape string   `json:"formationShape,omitempty"` // Expected formation type
}

// LoadScenario loads a test scenario from a JSON file
func LoadScenario(path string) (*TestScenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario file: %w", err)
	}

	var scenario TestScenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, fmt.Errorf("failed to parse scenario JSON: %w", err)
	}

	// Validate scenario
	if err := scenario.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scenario: %w", err)
	}

	return &scenario, nil
}

// Validate checks if the scenario is valid
func (s *TestScenario) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("scenario name is required")
	}
	if s.Map == "" {
		return fmt.Errorf("map is required")
	}
	if len(s.Setup.Units) == 0 && len(s.Setup.Buildings) == 0 {
		return fmt.Errorf("setup must have at least one unit or building")
	}
	if s.Expectations.MaxTicks <= 0 {
		return fmt.Errorf("maxTicks must be positive")
	}

	// Validate unit IDs are unique
	unitIDs := make(map[string]bool)
	for _, unit := range s.Setup.Units {
		if unit.ID == "" {
			return fmt.Errorf("unit ID is required")
		}
		if unitIDs[unit.ID] {
			return fmt.Errorf("duplicate unit ID: %s", unit.ID)
		}
		unitIDs[unit.ID] = true
	}

	return nil
}

// GetUnitByID finds a setup unit by ID
func (s *TestScenario) GetUnitByID(id string) *ScenarioUnit {
	for i := range s.Setup.Units {
		if s.Setup.Units[i].ID == id {
			return &s.Setup.Units[i]
		}
	}
	return nil
}

// GetExpectedUnitByID finds an expected unit by ID
func (s *TestScenario) GetExpectedUnitByID(id string) *ExpectedUnit {
	for i := range s.Expectations.FinalState.Units {
		if s.Expectations.FinalState.Units[i].ID == id {
			return &s.Expectations.FinalState.Units[i]
		}
	}
	return nil
}
