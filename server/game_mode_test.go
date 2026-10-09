package main

import (
	"fmt"
	"testing"
	"time"
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

// addTestClient registers a client on a non-started server (deterministic).
func addTestClient(t *testing.T, server *GameServer, id uint32) {
	t.Helper()
	server.mu.Lock()
	server.clients[id] = &Client{Id: id, Name: fmt.Sprintf("Player%d", id), Money: 100, LastSeen: time.Now()}
	server.mu.Unlock()
}

// newTestMatchContext builds a MatchContext for unit tests.
func newTestMatchContext(server *GameServer, mode *JSONGameMode, tick uint64) *MatchContext {
	return &MatchContext{
		Server:   server,
		Stats:    server.stats,
		EventBus: server.eventBus,
		Mode:     mode.definition,
		Tick:     tick,
	}
}

func TestGameModeWinCondition(t *testing.T) {
	// resource_race win: money_current(self) >= resource_goal (500)
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	addTestClient(t, server, 1)
	addTestClient(t, server, 2)
	stats := server.stats
	stats.Set(1, StatMoneyCurrent, 400.0)
	stats.Set(2, StatMoneyCurrent, 400.0)

	ctx := newTestMatchContext(server, mode, 100)

	// Neither player is at the goal yet
	if result := mode.CheckWinConditions(ctx); result.HasWinner {
		t.Error("Expected no winner when both players below goal")
	}

	// Player 2 reaches the goal
	stats.Set(2, StatMoneyCurrent, 500.0)
	result := mode.CheckWinConditions(ctx)
	if !result.HasWinner {
		t.Fatal("Expected a winner when player 2 reaches goal")
	}
	if result.WinnerID != 2 {
		t.Errorf("Expected winner 2, got %d", result.WinnerID)
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
	// survival win: total_buildings(global) == buildings_placed(self)
	mode, err := LoadGameModeAndCreate("game_modes/survival.json")
	if err != nil {
		t.Fatalf("Failed to load survival mode: %v", err)
	}

	if mode.GetID() != "survival" {
		t.Errorf("Expected ID 'survival', got '%s'", mode.GetID())
	}

	server := NewGameServer()
	addTestClient(t, server, 1)
	addTestClient(t, server, 2)
	stats := server.stats

	ctx := newTestMatchContext(server, mode, 500)

	// Buildings split between two players -> no sole survivor yet
	stats.SetGlobal(StatGlobalTotalBuildings, 5.0)
	stats.Set(1, StatBuildingsPlaced, 3.0)
	stats.Set(2, StatBuildingsPlaced, 2.0)
	if r := mode.CheckWinConditions(ctx); r.HasWinner {
		t.Error("Expected no winner while buildings are split")
	}

	// Player 1 owns all remaining buildings -> sole survivor
	stats.Set(1, StatBuildingsPlaced, 5.0)
	stats.Set(2, StatBuildingsPlaced, 0.0)
	result := mode.CheckWinConditions(ctx)
	if !result.HasWinner {
		t.Fatal("Expected a winner when one player owns all buildings")
	}
	if result.WinnerID != 1 {
		t.Errorf("Expected winner 1, got %d", result.WinnerID)
	}
}

func TestTimedRaceGameMode(t *testing.T) {
	// timed_race win: match_duration_ticks(global) >= time_limit_ticks (12000),
	// winner = highest_money
	mode, err := LoadGameModeAndCreate("game_modes/timed_race.json")
	if err != nil {
		t.Fatalf("Failed to load timed race mode: %v", err)
	}

	if mode.GetID() != "timed_race" {
		t.Errorf("Expected ID 'timed_race', got '%s'", mode.GetID())
	}

	server := NewGameServer()
	addTestClient(t, server, 1)
	addTestClient(t, server, 2)
	stats := server.stats
	stats.Set(1, StatMoneyCurrent, 100.0)
	stats.Set(2, StatMoneyCurrent, 300.0) // richer

	ctx := newTestMatchContext(server, mode, 10000)

	// Time hasn't run out yet
	stats.SetGlobal(StatGlobalMatchDurationTicks, 10000.0)
	if r := mode.CheckWinConditions(ctx); r.HasWinner {
		t.Error("Expected no winner before time limit reached")
	}

	// Time has run out -> richest player (2) wins
	stats.SetGlobal(StatGlobalMatchDurationTicks, 12000.0)
	result := mode.CheckWinConditions(ctx)
	if !result.HasWinner {
		t.Fatal("Expected a winner when time limit reached")
	}
	if result.WinnerID != 2 {
		t.Errorf("Expected highest-money winner 2, got %d", result.WinnerID)
	}
	if result.ConditionID != "time_limit_reached" {
		t.Errorf("Expected condition ID 'time_limit_reached', got '%s'", result.ConditionID)
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

func TestLoseConditionLastPlayerStanding(t *testing.T) {
	// resource_race lose: money_current(self) <= 0 AND buildings_placed(self) == 0
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	server.setGameMode(mode, mode.definition)
	addTestClient(t, server, 1)
	addTestClient(t, server, 2)

	// Player 1 healthy (has money), player 2 broke with no buildings
	server.stats.Set(1, StatMoneyCurrent, 100.0)
	server.stats.Set(1, StatBuildingsPlaced, 1.0)
	// player 2 defaults: money 0, buildings 0 -> losing

	server.checkMatchEnd()

	if !server.gameOver {
		t.Fatal("Expected match to end with one player losing")
	}
	if server.winnerId != 1 {
		t.Fatalf("Expected surviving player 1 to win, got %d", server.winnerId)
	}
}

func TestLoseConditionFreshPlayerNotBroke(t *testing.T) {
	// A freshly-joined player has money_current seeded (not 0) so the
	// bankruptcy rule must not fire immediately.
	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("Failed to create game mode: %v", err)
	}

	server := NewGameServer()
	server.setGameMode(mode, mode.definition)
	addTestClient(t, server, 1)
	addTestClient(t, server, 2)
	server.stats.Set(1, StatMoneyCurrent, 100.0)
	server.stats.Set(2, StatMoneyCurrent, 100.0)

	server.checkMatchEnd()

	if server.gameOver {
		t.Fatal("Match must not end while both players are solvent")
	}
}

func TestPeriodicGrantResources(t *testing.T) {
	def := &GameModeDefinition{
		ID:         "grant_test",
		Name:       "Grant Test",
		Parameters: map[string]any{},
		PeriodicChecks: []PeriodicCheck{
			{
				Name:          "grant",
				IntervalTicks: 1,
				Action: map[string]any{
					"type":                "grant_resources",
					"amount_per_building": float64(10),
					"building_type":       "generator",
				},
			},
		},
	}
	mode, err := NewJSONGameMode(def)
	if err != nil {
		t.Fatalf("build mode: %v", err)
	}

	server := NewGameServer()
	addTestClient(t, server, 1) // has 2 generators
	addTestClient(t, server, 2) // has none
	server.mu.Lock()
	server.entities[100] = &Entity{Id: 100, OwnerId: 1, Type: "generator", FootprintWidth: 2, FootprintHeight: 2}
	server.entities[101] = &Entity{Id: 101, OwnerId: 1, Type: "generator", FootprintWidth: 2, FootprintHeight: 2}
	server.mu.Unlock()
	server.stats.Set(1, StatMoneyCurrent, 100.0)
	server.stats.Set(2, StatMoneyCurrent, 100.0)

	ctx := newTestMatchContext(server, mode, 1)
	mode.OnTick(ctx)

	// Player 1: 2 generators * $10 = $20
	if got := server.clients[1].Money; got != 120 {
		t.Errorf("expected player 1 money 120, got %v", got)
	}
	if got := server.stats.Get(1, StatMoneyCurrent); got != 120 {
		t.Errorf("expected money_current stat synced to 120, got %v", got)
	}
	if got := server.stats.Get(1, StatResourcesGenerated); got != 20 {
		t.Errorf("expected resources_generated 20, got %v", got)
	}
	// Player 2: no generators -> unchanged
	if got := server.clients[2].Money; got != 100 {
		t.Errorf("expected player 2 money unchanged 100, got %v", got)
	}
}

func TestPeriodicIncrementGlobalStat(t *testing.T) {
	def := &GameModeDefinition{
		ID:         "counter_test",
		Name:       "Counter Test",
		Parameters: map[string]any{},
		PeriodicChecks: []PeriodicCheck{
			{
				Name:          "tick_counter",
				IntervalTicks: 1,
				Action: map[string]any{
					"type":   "increment_global_stat",
					"stat":   StatGlobalMatchDurationTicks,
					"amount": float64(1),
				},
			},
		},
	}
	mode, err := NewJSONGameMode(def)
	if err != nil {
		t.Fatalf("build mode: %v", err)
	}

	server := NewGameServer()
	ctx := newTestMatchContext(server, mode, 1)

	for i := 0; i < 5; i++ {
		ctx.Tick = uint64(i + 1)
		mode.OnTick(ctx)
	}

	if got := server.stats.GetGlobal(StatGlobalMatchDurationTicks); got != 5 {
		t.Errorf("expected match_duration_ticks 5, got %v", got)
	}
}

func TestResourceRaceEndToEnd(t *testing.T) {
	server := NewGameServer()
	server.mapData = &MapData{Width: 20, Height: 10}

	mode, err := LoadGameModeAndCreate("game_modes/resource_race.json")
	if err != nil {
		t.Fatalf("load mode: %v", err)
	}
	server.setGameMode(mode, mode.definition)
	addTestClient(t, server, 1)
	addTestClient(t, server, 2)

	// Player 1 hits the resource goal; no generators so income won't overwrite
	server.stats.Set(1, StatMoneyCurrent, 500.0)
	server.stats.Set(2, StatMoneyCurrent, 100.0)

	matchEndEvents := 0
	server.eventBus.On(EventMatchEnded, func(e GameEvent) { matchEndEvents++ })

	// A single tick should evaluate the win condition and end the match
	server.gameTick()

	if !server.gameOver {
		t.Fatal("Expected match to end after goal reached")
	}
	if server.winnerId != 1 {
		t.Fatalf("Expected winner 1, got %d", server.winnerId)
	}
	if matchEndEvents != 1 {
		t.Fatalf("expected exactly 1 match.ended event, got %d", matchEndEvents)
	}
}
