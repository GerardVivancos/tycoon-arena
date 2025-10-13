package main

import (
	"path/filepath"
	"realtime-game-server/testutil"
	"strings"
	"testing"
)

// TestAllScenarios discovers and runs all scenario JSON files
func TestAllScenarios(t *testing.T) {
	// Find all scenario files
	scenarioFiles, err := filepath.Glob("../maps/scenarios/*.json")
	if err != nil {
		t.Fatalf("Failed to glob scenario files: %v", err)
	}

	if len(scenarioFiles) == 0 {
		t.Skip("No scenario files found in ../maps/scenarios/")
		return
	}

	t.Logf("Found %d scenario file(s)", len(scenarioFiles))

	// Run each scenario as a subtest
	for _, scenarioFile := range scenarioFiles {
		scenarioName := filepath.Base(scenarioFile)
		scenarioName = strings.TrimSuffix(scenarioName, ".json")

		t.Run(scenarioName, func(t *testing.T) {
			runScenarioTest(t, scenarioFile)
		})
	}
}

// runScenarioTest runs a single scenario test
func runScenarioTest(t *testing.T, scenarioFile string) {
	// Load scenario
	scenario, err := testutil.LoadScenario(scenarioFile)
	if err != nil {
		t.Fatalf("Failed to load scenario: %v", err)
	}

	t.Logf("Running scenario: %s", scenario.Name)
	if scenario.Description != "" {
		t.Logf("Description: %s", scenario.Description)
	}

	// Create test game server
	adapter := NewTestGameServerAdapter()

	// Run scenario
	result, err := testutil.RunScenario(scenario, adapter)
	if err != nil {
		t.Fatalf("Failed to run scenario: %v", err)
	}

	// Check result
	if !result.Passed {
		t.Errorf("Scenario failed with %d violation(s):", len(result.Violations))
		for i, violation := range result.Violations {
			t.Errorf("  %d. %s", i+1, violation)
		}
	}

	t.Logf("Scenario completed in %d ticks", result.ExecutionTime)
}

// TestGameServerAdapter adapts GameServer to implement testutil.GameServerInterface
type TestGameServerAdapter struct {
	server       *GameServer
	entityIDMap  map[uint32]*Entity // Quick lookup
	deltaTime    float32
	ticksPerStep int
}

// NewTestGameServerAdapter creates a new test adapter
func NewTestGameServerAdapter() *TestGameServerAdapter {
	return &TestGameServerAdapter{
		server:       NewGameServer(),
		entityIDMap:  make(map[uint32]*Entity),
		deltaTime:    1.0 / float32(TickRate), // 50ms per tick
		ticksPerStep: 1,
	}
}

// LoadMap loads a map file into the game server
func (a *TestGameServerAdapter) LoadMap(path string) error {
	mapData, err := LoadMap(path)
	if err != nil {
		return err
	}

	a.server.mapData = mapData
	return nil
}

// SpawnUnit creates a unit at the specified position
func (a *TestGameServerAdapter) SpawnUnit(unitType string, team int, x, y int) uint32 {
	entityID := a.server.nextId
	a.server.nextId++

	entity := &Entity{
		Id:           entityID,
		OwnerId:      uint32(team), // Use team as owner ID
		Type:         unitType,
		TileX:        x,
		TileY:        y,
		TargetTileX:  x,
		TargetTileY:  y,
		MoveProgress: 0.0,
		Health:       100,
		MaxHealth:    100,
	}

	a.server.entities[entityID] = entity
	a.entityIDMap[entityID] = entity

	return entityID
}

// SpawnBuilding creates a building at the specified position
func (a *TestGameServerAdapter) SpawnBuilding(buildingType string, team int, x, y int) uint32 {
	entityID := a.server.nextId
	a.server.nextId++

	footprintWidth := 2
	footprintHeight := 2

	entity := &Entity{
		Id:              entityID,
		OwnerId:         uint32(team),
		Type:            buildingType,
		TileX:           x,
		TileY:           y,
		TargetTileX:     x,
		TargetTileY:     y,
		MoveProgress:    0.0,
		Health:          100,
		MaxHealth:       100,
		FootprintWidth:  footprintWidth,
		FootprintHeight: footprintHeight,
	}

	a.server.entities[entityID] = entity
	a.entityIDMap[entityID] = entity

	return entityID
}

// Tick advances the game simulation by one tick
func (a *TestGameServerAdapter) Tick() {
	// Process movement for all entities
	for _, entity := range a.server.entities {
		if entity.Type == "worker" || entity.Type == "unit" {
			a.server.updateEntityMovement(entity, a.deltaTime)
		}
	}

	a.server.tick++
}

// GetEntityPosition returns the current position of an entity
func (a *TestGameServerAdapter) GetEntityPosition(entityID uint32) *[2]int {
	entity, exists := a.server.entities[entityID]
	if !exists {
		return nil
	}

	return &[2]int{entity.TileX, entity.TileY}
}

// GetEntityTeam returns the team/owner of an entity
func (a *TestGameServerAdapter) GetEntityTeam(entityID uint32) int {
	entity, exists := a.server.entities[entityID]
	if !exists {
		return -1
	}

	return int(entity.OwnerId)
}

// IsEntityMoving returns true if the entity is currently moving
func (a *TestGameServerAdapter) IsEntityMoving(entityID uint32) bool {
	entity, exists := a.server.entities[entityID]
	if !exists {
		return false
	}

	// Entity is moving if it has a path with remaining waypoints
	return len(entity.Path) > 0 && entity.PathIndex < len(entity.Path)
}

// EntityExists returns true if the entity exists in the game
func (a *TestGameServerAdapter) EntityExists(entityID uint32) bool {
	_, exists := a.server.entities[entityID]
	return exists
}

// MoveUnits commands units to move to a target position in formation
func (a *TestGameServerAdapter) MoveUnits(entityIDs []uint32, targetX, targetY int, formation string) error {
	// Create a mock client for the move command
	mockClient := &Client{
		Id:    0, // Test client
		Name:  "TestClient",
		Money: 1000,
	}

	// Convert unit IDs to interface{} array (as JSON parsing would do)
	unitIdsInterface := make([]interface{}, len(entityIDs))
	for i, id := range entityIDs {
		unitIdsInterface[i] = float64(id) // JSON numbers are float64
	}

	// Create move command data as map (simulating JSON parsing)
	moveData := map[string]interface{}{
		"unitIds":     unitIdsInterface,
		"targetTileX": float64(targetX),
		"targetTileY": float64(targetY),
		"formation":   formation,
	}

	// Convert to Command struct
	cmd := Command{
		Type: "move",
		Data: moveData,
	}

	// Process the command
	a.server.handleMoveCommand(cmd, mockClient)

	return nil
}

// AttackTarget commands units to attack a target
func (a *TestGameServerAdapter) AttackTarget(entityIDs []uint32, targetID uint32) error {
	// Create mock client
	mockClient := &Client{
		Id:    0,
		Name:  "TestClient",
		Money: 1000,
	}

	// Create attack command data as map (simulating JSON parsing)
	attackData := map[string]interface{}{
		"targetId": float64(targetID),
	}

	cmd := Command{
		Type: "attack",
		Data: attackData,
	}

	// Process the command (once for all units)
	a.server.handleAttackCommand(cmd, mockClient)

	return nil
}
