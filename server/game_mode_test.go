package main

import (
	"testing"
)

func TestLoadGameMode(t *testing.T) {
	// Test loading resource_race game mode
	def, err := LoadGameMode("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to load game mode: %v", err)
	}

	if def.ID != "resource_race" {
		t.Errorf("Expected ID 'resource_race', got '%s'", def.ID)
	}

	if def.Name != "Resource Race" {
		t.Errorf("Expected name 'Resource Race', got '%s'", def.Name)
	}

	if len(def.WinConditions) == 0 {
		t.Error("Expected at least one win condition")
	}

	// Check parameters
	resourceGoal, ok := def.Parameters["resource_goal"].(float64)
	if !ok {
		t.Error("Expected resource_goal parameter to be a number")
	}
	if resourceGoal != 500.0 {
		t.Errorf("Expected resource_goal 500, got %f", resourceGoal)
	}
}

func TestNewJSONGameMode(t *testing.T) {
	// Load and create game mode
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	if mode.GetID() != "resource_race" {
		t.Errorf("Expected ID 'resource_race', got '%s'", mode.GetID())
	}

	if mode.GetName() != "Resource Race" {
		t.Errorf("Expected name 'Resource Race', got '%s'", mode.GetName())
	}

	// Check that rules were compiled
	if len(mode.compiledWinRules) == 0 {
		t.Error("Expected compiled win rules")
	}
}

func TestGameModeWinCondition(t *testing.T) {
	// Create game mode
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	// Create mock server and stats
	server := NewGameServer()
	stats := server.stats

	// Set up a player who hasn't won yet
	stats.Set(0, StatMoneyCurrent, 400.0)

	ctx := &MatchContext{
		Server:   server,
		Stats:    stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     100,
	}

	// Check win conditions - should not have a winner
	result := mode.CheckWinConditions(ctx)
	if result.HasWinner {
		t.Error("Expected no winner when money < goal")
	}

	// Set player money to winning amount
	stats.Set(0, StatMoneyCurrent, 500.0)

	// Check win conditions - should have a winner
	result = mode.CheckWinConditions(ctx)
	if !result.HasWinner {
		t.Error("Expected a winner when money >= goal")
	}

	if result.ConditionID != "resource_victory" {
		t.Errorf("Expected condition ID 'resource_victory', got '%s'", result.ConditionID)
	}
}

func TestGameModeObjectiveText(t *testing.T) {
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	ctx := &MatchContext{
		Server:   server,
		Stats:    server.stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     0,
	}

	objective := mode.GetObjectiveText(ctx, 1)
	expected := "Reach $500 before your opponent"
	if objective != expected {
		t.Errorf("Expected objective '%s', got '%s'", expected, objective)
	}
}

func TestGameModeStatistics(t *testing.T) {
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	stats := server.stats

	// Add some test data
	stats.Set(1, StatMoneyCurrent, 250.0)
	stats.Set(1, StatBuildingsPlaced, 3.0)
	stats.SetGlobal(StatGlobalTotalBuildings, 5.0)

	// Create a mock client
	server.mu.Lock()
	server.clients[1] = &Client{
		Id:   1,
		Name: "TestPlayer",
	}
	server.mu.Unlock()

	ctx := &MatchContext{
		Server:   server,
		Stats:    stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     100,
	}

	statistics := mode.GetStatistics(ctx)

	// Check that statistics were returned
	if statistics == nil {
		t.Fatal("Expected statistics to be returned")
	}

	// Check global stats
	globalStats, ok := statistics["global"].(map[string]any)
	if !ok {
		t.Fatal("Expected global stats to be present")
	}

	if globalStats["total_buildings"].(float64) != 5.0 {
		t.Errorf("Expected total buildings 5, got %v", globalStats["total_buildings"])
	}
}

func TestSurvivalGameMode(t *testing.T) {
	// Test loading survival mode
	mode, err := LoadGameModeAndCreate("game_modes/survival.json")
	if err != nil {
		t.Fatalf("Failed to load survival mode: %v", err)
	}

	if mode.GetID() != "survival" {
		t.Errorf("Expected ID 'survival', got '%s'", mode.GetID())
	}

	server := NewGameServer()
	stats := server.stats

	// Simulate a scenario where one player has destroyed all enemy buildings
	stats.Set(0, StatBuildingsPlaced, 5.0)
	stats.SetGlobal(StatGlobalTotalBuildings, 5.0) // Only player 0's buildings remain

	ctx := &MatchContext{
		Server:   server,
		Stats:    stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     500,
	}

	result := mode.CheckWinConditions(ctx)
	if !result.HasWinner {
		t.Error("Expected a winner in survival mode when only one player has buildings")
	}
}

func TestTimedRaceGameMode(t *testing.T) {
	// Test loading timed race mode
	mode, err := LoadGameModeAndCreate("game_modes/timed_race.json")
	if err != nil {
		t.Fatalf("Failed to load timed race mode: %v", err)
	}

	if mode.GetID() != "timed_race" {
		t.Errorf("Expected ID 'timed_race', got '%s'", mode.GetID())
	}

	server := NewGameServer()
	stats := server.stats

	// Time hasn't run out yet
	stats.SetGlobal(StatGlobalMatchDurationTicks, 10000.0)

	ctx := &MatchContext{
		Server:   server,
		Stats:    stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     10000,
	}

	result := mode.CheckWinConditions(ctx)
	if result.HasWinner {
		t.Error("Expected no winner before time limit reached")
	}

	// Time has run out
	stats.SetGlobal(StatGlobalMatchDurationTicks, 12000.0)

	result = mode.CheckWinConditions(ctx)
	if !result.HasWinner {
		t.Error("Expected a winner when time limit reached")
	}
}

func TestGameModePeriodicChecks(t *testing.T) {
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	ctx := &MatchContext{
		Server:   server,
		Stats:    server.stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     20, // Should trigger periodic check with interval 20
	}

	// Call OnTick - this should execute periodic checks
	// For now, this just verifies no panic
	mode.OnTick(ctx)

	// At tick 20, periodic check should run
	ctx.Tick = 40
	mode.OnTick(ctx)

	// At tick 25, periodic check should NOT run
	ctx.Tick = 25
	mode.OnTick(ctx)
}

func TestGameModeLifecycleHooks(t *testing.T) {
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	ctx := &MatchContext{
		Server:   server,
		Stats:    server.stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     0,
	}

	// Call all lifecycle hooks - verify they don't panic
	mode.OnMatchCreated(ctx)
	mode.OnPlayerJoined(ctx, 1)
	mode.OnMatchStarting(ctx)
	mode.OnMatchStarted(ctx)
	mode.OnMatchEnding(ctx, "test")
	mode.OnMatchEnded(ctx)
	mode.OnPlayerLeft(ctx, 1)
}
