package main

import (
	"testing"
	"time"
)

func TestStatisticsBasicOperations(t *testing.T) {
	stats := NewStatisticsTracker()

	playerID := uint32(1)

	// Test Set
	stats.Set(playerID, StatMoneyCurrent, 100.0)
	if got := stats.Get(playerID, StatMoneyCurrent); got != 100.0 {
		t.Errorf("Expected 100.0, got %f", got)
	}

	// Test Increment
	stats.Increment(playerID, StatMoneyCurrent, 50.0)
	if got := stats.Get(playerID, StatMoneyCurrent); got != 150.0 {
		t.Errorf("Expected 150.0, got %f", got)
	}

	// Test Increment negative
	stats.Increment(playerID, StatMoneyCurrent, -25.0)
	if got := stats.Get(playerID, StatMoneyCurrent); got != 125.0 {
		t.Errorf("Expected 125.0, got %f", got)
	}
}

func TestStatisticsMultiplePlayers(t *testing.T) {
	stats := NewStatisticsTracker()

	// Set stats for different players
	stats.Set(1, StatMoneyCurrent, 100.0)
	stats.Set(2, StatMoneyCurrent, 200.0)
	stats.Set(3, StatMoneyCurrent, 300.0)

	// Verify each player has correct value
	if got := stats.Get(1, StatMoneyCurrent); got != 100.0 {
		t.Errorf("Player 1: expected 100.0, got %f", got)
	}
	if got := stats.Get(2, StatMoneyCurrent); got != 200.0 {
		t.Errorf("Player 2: expected 200.0, got %f", got)
	}
	if got := stats.Get(3, StatMoneyCurrent); got != 300.0 {
		t.Errorf("Player 3: expected 300.0, got %f", got)
	}

	// Increment one player
	stats.Increment(2, StatMoneyCurrent, 50.0)

	// Verify only that player changed
	if got := stats.Get(1, StatMoneyCurrent); got != 100.0 {
		t.Errorf("Player 1: expected 100.0, got %f", got)
	}
	if got := stats.Get(2, StatMoneyCurrent); got != 250.0 {
		t.Errorf("Player 2: expected 250.0, got %f", got)
	}
	if got := stats.Get(3, StatMoneyCurrent); got != 300.0 {
		t.Errorf("Player 3: expected 300.0, got %f", got)
	}
}

func TestStatisticsGetAll(t *testing.T) {
	stats := NewStatisticsTracker()

	playerID := uint32(1)

	stats.Set(playerID, StatMoneyCurrent, 100.0)
	stats.Set(playerID, StatBuildingsPlaced, 5.0)
	stats.Set(playerID, StatDamageDealt, 125.0)

	all := stats.GetAll(playerID)

	if len(all) != 3 {
		t.Errorf("Expected 3 stats, got %d", len(all))
	}

	if all[StatMoneyCurrent] != 100.0 {
		t.Errorf("Expected money 100.0, got %f", all[StatMoneyCurrent])
	}
	if all[StatBuildingsPlaced] != 5.0 {
		t.Errorf("Expected buildings 5.0, got %f", all[StatBuildingsPlaced])
	}
	if all[StatDamageDealt] != 125.0 {
		t.Errorf("Expected damage 125.0, got %f", all[StatDamageDealt])
	}
}

func TestStatisticsGlobalCounters(t *testing.T) {
	stats := NewStatisticsTracker()

	// Test global stats
	stats.SetGlobal(StatGlobalMatchDurationTicks, 1000.0)
	if got := stats.GetGlobal(StatGlobalMatchDurationTicks); got != 1000.0 {
		t.Errorf("Expected 1000.0, got %f", got)
	}

	stats.IncrementGlobal(StatGlobalTotalBuildings, 5.0)
	if got := stats.GetGlobal(StatGlobalTotalBuildings); got != 5.0 {
		t.Errorf("Expected 5.0, got %f", got)
	}

	stats.IncrementGlobal(StatGlobalTotalBuildings, 3.0)
	if got := stats.GetGlobal(StatGlobalTotalBuildings); got != 8.0 {
		t.Errorf("Expected 8.0, got %f", got)
	}
}

func TestStatisticsTimeSeries(t *testing.T) {
	stats := NewStatisticsTracker()

	playerID := uint32(1)

	// Record time series data
	stats.RecordTimeSeries(playerID, StatMoneyCurrent, 100, 100.0)
	stats.RecordTimeSeries(playerID, StatMoneyCurrent, 200, 150.0)
	stats.RecordTimeSeries(playerID, StatMoneyCurrent, 300, 175.0)

	series := stats.GetTimeSeries(playerID, StatMoneyCurrent)

	if len(series) != 3 {
		t.Fatalf("Expected 3 time points, got %d", len(series))
	}

	if series[0].Tick != 100 || series[0].Value != 100.0 {
		t.Errorf("Point 0: expected (100, 100.0), got (%d, %f)", series[0].Tick, series[0].Value)
	}
	if series[1].Tick != 200 || series[1].Value != 150.0 {
		t.Errorf("Point 1: expected (200, 150.0), got (%d, %f)", series[1].Tick, series[1].Value)
	}
	if series[2].Tick != 300 || series[2].Value != 175.0 {
		t.Errorf("Point 2: expected (300, 175.0), got (%d, %f)", series[2].Tick, series[2].Value)
	}
}

func TestStatisticsReset(t *testing.T) {
	stats := NewStatisticsTracker()

	// Set some data
	stats.Set(1, StatMoneyCurrent, 100.0)
	stats.Set(2, StatBuildingsPlaced, 5.0)
	stats.SetGlobal(StatGlobalTotalBuildings, 10.0)
	stats.RecordTimeSeries(1, StatMoneyCurrent, 100, 100.0)

	// Reset
	stats.Reset()

	// Verify everything is cleared
	if got := stats.Get(1, StatMoneyCurrent); got != 0.0 {
		t.Errorf("After reset, expected 0.0, got %f", got)
	}
	if got := stats.GetGlobal(StatGlobalTotalBuildings); got != 0.0 {
		t.Errorf("After reset, expected global 0.0, got %f", got)
	}
	if series := stats.GetTimeSeries(1, StatMoneyCurrent); len(series) != 0 {
		t.Errorf("After reset, expected empty time series, got %d points", len(series))
	}
}

func TestStatisticsEventIntegration(t *testing.T) {
	// Test that statistics auto-track from events
	server := NewGameServer()

	playerID := uint32(1)

	// Emit building placed event
	server.eventBus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      100,
		ActorID:   playerID,
		TargetID:  10,
		Data: map[string]any{
			"buildingType": "generator",
		},
	})

	// Check that stat was incremented
	if got := server.stats.Get(playerID, StatBuildingsPlaced); got != 1.0 {
		t.Errorf("Expected buildings placed 1.0, got %f", got)
	}
	if got := server.stats.GetGlobal(StatGlobalTotalBuildings); got != 1.0 {
		t.Errorf("Expected global buildings 1.0, got %f", got)
	}

	// Emit money changed event
	server.eventBus.Emit(GameEvent{
		Type:      EventPlayerMoneyChanged,
		Timestamp: time.Now(),
		Tick:      110,
		ActorID:   playerID,
		Data: map[string]any{
			"oldMoney": 100.0,
			"newMoney": 150.0,
			"delta":    50.0,
			"reason":   "resource_generation",
		},
	})

	// Check money stats
	if got := server.stats.Get(playerID, StatMoneyCurrent); got != 150.0 {
		t.Errorf("Expected current money 150.0, got %f", got)
	}
	if got := server.stats.Get(playerID, StatMoneyEarnedTotal); got != 50.0 {
		t.Errorf("Expected total earned 50.0, got %f", got)
	}

	// Emit money spent event
	server.eventBus.Emit(GameEvent{
		Type:      EventPlayerMoneyChanged,
		Timestamp: time.Now(),
		Tick:      120,
		ActorID:   playerID,
		Data: map[string]any{
			"oldMoney": 150.0,
			"newMoney": 100.0,
			"delta":    -50.0,
			"reason":   "building_cost",
		},
	})

	// Check spent stats
	if got := server.stats.Get(playerID, StatMoneyCurrent); got != 100.0 {
		t.Errorf("Expected current money 100.0, got %f", got)
	}
	if got := server.stats.Get(playerID, StatMoneySpentTotal); got != 50.0 {
		t.Errorf("Expected total spent 50.0, got %f", got)
	}

	// Emit building destroyed event
	ownerID := uint32(2)
	server.eventBus.Emit(GameEvent{
		Type:      EventBuildingDestroyed,
		Timestamp: time.Now(),
		Tick:      130,
		ActorID:   playerID, // Player 1 destroyed it
		TargetID:  10,
		Data: map[string]any{
			"buildingType": "generator",
			"ownerId":      ownerID, // Player 2 owned it
		},
	})

	// Check destruction stats
	if got := server.stats.Get(playerID, StatBuildingsDestroyed); got != 1.0 {
		t.Errorf("Player 1: expected buildings destroyed 1.0, got %f", got)
	}
	if got := server.stats.Get(ownerID, StatBuildingsLost); got != 1.0 {
		t.Errorf("Player 2: expected buildings lost 1.0, got %f", got)
	}
	if got := server.stats.GetGlobal(StatGlobalTotalBuildings); got != 0.0 {
		t.Errorf("Expected global buildings 0.0, got %f", got)
	}
}
