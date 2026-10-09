package main

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func serverGameOver(s *GameServer) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gameOver
}

// TestEmitOutsideLockNoDeadlock is the Phase 4.1 regression guard: an event
// listener that acquires s.mu (as a stats/mode hook reasonably might) must not
// deadlock, because events are queued under the lock and emitted after it is
// released. If emission ever moved back under s.mu, this test hangs and fails.
func TestEmitOutsideLockNoDeadlock(t *testing.T) {
	s := combatTestServer()

	s.eventBus.On(EventBuildingPlaced, func(e GameEvent) {
		s.mu.RLock()
		_ = s.tick
		s.mu.RUnlock()
	})

	done := make(chan struct{})
	go func() {
		// Simulate a critical section: mutate under the lock, queue events,
		// release the lock, then flush (as gameTick does).
		s.mu.Lock()
		s.queueEvent(GameEvent{Type: EventBuildingPlaced, Tick: s.tick, ActorID: 1})
		s.mu.Unlock()
		s.flushEvents()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(3 * time.Second):
		t.Fatal("flushEvents deadlocked: a listener acquiring s.mu blocked")
	}
}

// TestConcurrentTickAndMatchEnd exercises the race-prone path under -race:
// the tick loop, concurrent money grants, and concurrent win-condition checks
// all hammer a running match. It asserts a clean single end with no races.
func TestConcurrentTickAndMatchEnd(t *testing.T) {
	s := NewGameServer()
	s.mapData = &MapData{
		Width:          20,
		Height:         10,
		DefaultTerrain: TerrainType{Type: "grass", Passable: true},
		Tiles:          map[TileCoord]TerrainType{},
	}

	def := &GameModeDefinition{
		ID:         "concurrent",
		Name:       "Concurrent Match",
		Parameters: map[string]any{"time_limit_ticks": float64(12000)},
		WinConditions: []WinCondition{
			{
				ID:       "time_limit",
				Name:     "Time Limit",
				Priority: 1,
				Winner:   "highest_money",
				Rule: RuleJSON{
					Type:     "comparison",
					Left:     &ValueJSON{Stat: "match_duration_ticks", Global: true},
					Operator: ">=",
					Right:    &ValueJSON{Param: "time_limit_ticks"},
				},
			},
		},
	}
	mode, err := NewJSONGameMode(def)
	if err != nil {
		t.Fatalf("build mode: %v", err)
	}
	s.setGameMode(mode, mode.definition)

	addTestClient(t, s, 1)
	s.stats.Set(1, StatMoneyCurrent, 100)

	var matchEnded int32
	s.eventBus.On(EventMatchEnded, func(e GameEvent) {
		atomic.AddInt32(&matchEnded, 1)
	})

	// Trip the win condition once (global stat threshold). Any concurrent
	// checkMatchEnd that observes it must finalize the match exactly once.
	s.stats.SetGlobal(StatGlobalMatchDurationTicks, 12000)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Goroutine A: drive the authoritative tick loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.gameTick()
			}
		}
	}()

	// Goroutine B: concurrently grant money and evaluate match end.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				s.grantMoneyToPlayer(1, 1, "concurrent_test")
				s.checkMatchEnd()
			}
		}
	}()

	// Wait for the match to end (bounded).
	deadline := time.Now().Add(5 * time.Second)
	for !serverGameOver(s) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	close(stop)
	wg.Wait()

	if !serverGameOver(s) {
		t.Fatal("expected match to end under concurrent load")
	}
	if got := atomic.LoadInt32(&matchEnded); got != 1 {
		t.Fatalf("expected exactly 1 match.ended event, got %d", got)
	}
	if !serverGameOver(s) {
		t.Error("expected gameOver to be set")
	}
}
