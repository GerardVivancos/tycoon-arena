package main

import (
	"sync"
)

// Standard statistics tracked per player
const (
	// Current state
	StatMoneyCurrent = "money_current"

	// Cumulative totals
	StatMoneyEarnedTotal = "money_earned_total"
	StatMoneySpentTotal  = "money_spent_total"

	// Building stats
	StatBuildingsPlaced    = "buildings_placed"
	StatBuildingsDestroyed = "buildings_destroyed" // Buildings player destroyed
	StatBuildingsLost      = "buildings_lost"      // Own buildings destroyed

	// Combat stats
	StatUnitsKilled     = "units_killed"
	StatUnitsLost       = "units_lost"
	StatDamageDealt     = "damage_dealt"
	StatDamageTaken     = "damage_taken"
	StatAttacksMade     = "attacks_made"
	StatAttacksReceived = "attacks_received"

	// Resource generation
	StatResourcesGenerated = "resources_generated"
)

// Global statistics (not per-player)
const (
	StatGlobalMatchDurationTicks = "match_duration_ticks"
	StatGlobalTotalBuildings     = "total_buildings"
	StatGlobalTotalUnits         = "total_units"
)

// TimePoint represents a value at a specific tick (for time series)
type TimePoint struct {
	Tick  uint64
	Value float64
}

// StatisticsTracker tracks arbitrary metrics over time
type StatisticsTracker struct {
	// Per-player counters
	counters map[uint32]map[string]float64

	// Per-player time series (optional - for graphs/history)
	timeSeries map[uint32]map[string][]TimePoint

	// Global match metrics
	globalCounters map[string]float64

	mu sync.RWMutex
}

// NewStatisticsTracker creates a new statistics tracker
func NewStatisticsTracker() *StatisticsTracker {
	return &StatisticsTracker{
		counters:       make(map[uint32]map[string]float64),
		timeSeries:     make(map[uint32]map[string][]TimePoint),
		globalCounters: make(map[string]float64),
	}
}

// Increment adds to a player's stat
func (st *StatisticsTracker) Increment(playerID uint32, stat string, delta float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if _, ok := st.counters[playerID]; !ok {
		st.counters[playerID] = make(map[string]float64)
	}
	st.counters[playerID][stat] += delta
}

// Set sets a player's stat to a specific value
func (st *StatisticsTracker) Set(playerID uint32, stat string, value float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if _, ok := st.counters[playerID]; !ok {
		st.counters[playerID] = make(map[string]float64)
	}
	st.counters[playerID][stat] = value
}

// Get retrieves a player's stat value (returns 0.0 if not found)
func (st *StatisticsTracker) Get(playerID uint32, stat string) float64 {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if counters, ok := st.counters[playerID]; ok {
		return counters[stat]
	}
	return 0.0
}

// GetAll retrieves all stats for a player
func (st *StatisticsTracker) GetAll(playerID uint32) map[string]float64 {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if counters, ok := st.counters[playerID]; ok {
		// Return a copy to avoid race conditions
		result := make(map[string]float64, len(counters))
		for k, v := range counters {
			result[k] = v
		}
		return result
	}
	return make(map[string]float64)
}

// IncrementGlobal adds to a global stat
func (st *StatisticsTracker) IncrementGlobal(stat string, delta float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.globalCounters[stat] += delta
}

// SetGlobal sets a global stat to a specific value
func (st *StatisticsTracker) SetGlobal(stat string, value float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.globalCounters[stat] = value
}

// GetGlobal retrieves a global stat value
func (st *StatisticsTracker) GetGlobal(stat string) float64 {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return st.globalCounters[stat]
}

// RecordTimeSeries adds a time series data point for a player's stat
func (st *StatisticsTracker) RecordTimeSeries(playerID uint32, stat string, tick uint64, value float64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if _, ok := st.timeSeries[playerID]; !ok {
		st.timeSeries[playerID] = make(map[string][]TimePoint)
	}

	st.timeSeries[playerID][stat] = append(st.timeSeries[playerID][stat], TimePoint{
		Tick:  tick,
		Value: value,
	})
}

// GetTimeSeries retrieves time series data for a player's stat
func (st *StatisticsTracker) GetTimeSeries(playerID uint32, stat string) []TimePoint {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if series, ok := st.timeSeries[playerID]; ok {
		if points, ok := series[stat]; ok {
			// Return a copy
			result := make([]TimePoint, len(points))
			copy(result, points)
			return result
		}
	}
	return []TimePoint{}
}

// Reset clears all statistics
func (st *StatisticsTracker) Reset() {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.counters = make(map[uint32]map[string]float64)
	st.timeSeries = make(map[uint32]map[string][]TimePoint)
	st.globalCounters = make(map[string]float64)
}

// setupStatisticsTracking wires the statistics tracker to the event bus
func (s *GameServer) setupStatisticsTracking() {
	if s.eventBus == nil || s.stats == nil {
		return
	}

	// Track building placement
	s.eventBus.On(EventBuildingPlaced, func(e GameEvent) {
		s.stats.Increment(e.ActorID, StatBuildingsPlaced, 1)
		s.stats.IncrementGlobal(StatGlobalTotalBuildings, 1)
	})

	// Track building destruction
	s.eventBus.On(EventBuildingDestroyed, func(e GameEvent) {
		destroyerID := e.ActorID
		ownerID, ok := e.Data["ownerId"].(uint32)
		if !ok {
			return
		}

		s.stats.Increment(destroyerID, StatBuildingsDestroyed, 1)
		s.stats.Increment(ownerID, StatBuildingsLost, 1)
		s.stats.IncrementGlobal(StatGlobalTotalBuildings, -1)
	})

	// Track money changes
	s.eventBus.On(EventPlayerMoneyChanged, func(e GameEvent) {
		delta, ok := e.Data["delta"].(float64)
		if !ok {
			return
		}
		newMoney, ok := e.Data["newMoney"].(float64)
		if !ok {
			return
		}

		s.stats.Set(e.ActorID, StatMoneyCurrent, newMoney)

		if delta > 0 {
			s.stats.Increment(e.ActorID, StatMoneyEarnedTotal, delta)
		} else {
			s.stats.Increment(e.ActorID, StatMoneySpentTotal, -delta)
		}
	})

	// Track resource generation
	s.eventBus.On(EventResourceGenerated, func(e GameEvent) {
		amount, ok := e.Data["amount"].(float64)
		if !ok {
			return
		}
		s.stats.Increment(e.ActorID, StatResourcesGenerated, amount)
	})

	// Track attacks
	s.eventBus.On(EventUnitAttacked, func(e GameEvent) {
		s.stats.Increment(e.ActorID, StatAttacksMade, 1)
		// attacks_received lands on the target's owner
		if owner, ok := e.Data["targetOwnerId"].(uint32); ok {
			s.stats.Increment(owner, StatAttacksReceived, 1)
		}
	})

	// Track damage dealt and taken
	s.eventBus.On(EventEntityDamaged, func(e GameEvent) {
		damage, ok := e.Data["damage"].(int32)
		if !ok {
			// Try float64
			damageFl, ok2 := e.Data["damage"].(float64)
			if ok2 {
				damage = int32(damageFl)
			} else {
				return
			}
		}

		s.stats.Increment(e.ActorID, StatDamageDealt, float64(damage))

		// damage_taken lands on the target's owner (carried in the event so the
		// handler does not need to look up a possibly-deleted entity)
		if owner, ok := e.Data["targetOwnerId"].(uint32); ok {
			s.stats.Increment(owner, StatDamageTaken, float64(damage))
		}
	})

	// Track entity creation
	s.eventBus.On(EventEntityCreated, func(e GameEvent) {
		entityType, ok := e.Data["entityType"].(string)
		if !ok {
			return
		}
		// Distinguish units from buildings
		if entityType != "generator" {
			s.stats.IncrementGlobal(StatGlobalTotalUnits, 1)
		}
	})

	// Track entity destruction
	s.eventBus.On(EventEntityDestroyed, func(e GameEvent) {
		entityType, ok := e.Data["entityType"].(string)
		if !ok {
			return
		}
		ownerID, ok := e.Data["ownerId"].(uint32)
		if !ok {
			return
		}

		// Distinguish units from buildings
		if entityType != "generator" {
			s.stats.IncrementGlobal(StatGlobalTotalUnits, -1)
			s.stats.Increment(e.ActorID, StatUnitsKilled, 1)
			s.stats.Increment(ownerID, StatUnitsLost, 1)
		}
	})
}
