package main

import (
	"testing"
)

// combatTestServer returns a server with a small fully-passable map so unit
// combat (which relies on pathfinding) can run deterministically.
func combatTestServer() *GameServer {
	s := NewGameServer()
	s.mapData = &MapData{
		Width:          20,
		Height:         10,
		DefaultTerrain: TerrainType{Type: "grass", Passable: true},
		Tiles:          map[TileCoord]TerrainType{},
	}
	return s
}

func putUnit(t *testing.T, s *GameServer, id, owner uint32, x, y int) *Entity {
	t.Helper()
	e := &Entity{
		Id: id, OwnerId: owner, Type: "worker",
		TileX: x, TileY: y, TargetTileX: x, TargetTileY: y,
		Health: 100, MaxHealth: 100,
	}
	s.mu.Lock()
	s.entities[id] = e
	s.mu.Unlock()
	return e
}

func putBuildingWithHealth(t *testing.T, s *GameServer, id, owner uint32, x, y int, hp int32) *Entity {
	t.Helper()
	e := &Entity{
		Id: id, OwnerId: owner, Type: "generator",
		TileX: x, TileY: y, TargetTileX: x, TargetTileY: y,
		Health: hp, MaxHealth: hp, FootprintWidth: 2, FootprintHeight: 2,
	}
	s.mu.Lock()
	s.entities[id] = e
	s.mu.Unlock()
	return e
}

func clientOf(t *testing.T, s *GameServer, id uint32) *Client {
	t.Helper()
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[id]
}

// orderAttack builds an attack command (targetId + optional unitIds) and runs
// handleAttackCommand as the processCommand path would.
func orderAttack(s *GameServer, client *Client, targetId uint32, unitIds ...uint32) {
	data := map[string]any{"targetId": float64(targetId)}
	if len(unitIds) > 0 {
		list := make([]any, len(unitIds))
		for i, id := range unitIds {
			list[i] = float64(id)
		}
		data["unitIds"] = list
	}
	cmd := Command{Type: "attack", Data: data}
	s.handleAttackCommand(cmd, client)
}

// combatTick advances the simulation n ticks, mirroring gameTick's relevant
// order (movement, then combat).
func combatTick(s *GameServer, n int) {
	dt := 1.0 / float32(TickRate)
	for i := 0; i < n; i++ {
		s.tick++
		s.mu.Lock()
		for _, e := range s.entities {
			if e.Type == "worker" {
				s.updateEntityMovement(e, dt)
			}
		}
		s.processCombat()
		s.mu.Unlock()
	}
}

func TestUnitKillsEnemyUnit(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	attacker := putUnit(t, s, 10, 1, 5, 5)
	putUnit(t, s, 20, 2, 6, 5) // adjacent -> in range 1

	orderAttack(s, clientOf(t, s, 1), 20, 10)
	combatTick(s, 80)

	if _, exists := s.entities[20]; exists {
		t.Fatalf("expected enemy unit 20 to be destroyed")
	}
	if attacker.AttackTargetId != 0 {
		t.Errorf("expected attacker to clear its target after kill, got %d", attacker.AttackTargetId)
	}

	// Combat stats: 4 hits * 25 dmg
	if got := s.stats.Get(1, StatDamageDealt); got != 100 {
		t.Errorf("expected damage_dealt 100 for player 1, got %v", got)
	}
	if got := s.stats.Get(2, StatDamageTaken); got != 100 {
		t.Errorf("expected damage_taken 100 for player 2, got %v", got)
	}
	if got := s.stats.Get(1, StatAttacksMade); got != 4 {
		t.Errorf("expected attacks_made 4 for player 1, got %v", got)
	}
	if got := s.stats.Get(2, StatAttacksReceived); got != 4 {
		t.Errorf("expected attacks_received 4 for player 2, got %v", got)
	}
	if got := s.stats.Get(1, StatUnitsKilled); got != 1 {
		t.Errorf("expected units_killed 1 for player 1, got %v", got)
	}
	if got := s.stats.Get(2, StatUnitsLost); got != 1 {
		t.Errorf("expected units_lost 1 for player 2, got %v", got)
	}
}

func TestUnitChasesAndKillsTarget(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	attacker := putUnit(t, s, 10, 1, 2, 5)
	putUnit(t, s, 20, 2, 8, 5) // far away -> must path into range

	orderAttack(s, clientOf(t, s, 1), 20, 10)

	// After one tick it should be chasing (has a path) but not yet in range
	combatTick(s, 1)
	if len(attacker.Path) == 0 {
		t.Error("expected attacker to start pathing toward target")
	}
	if _, exists := s.entities[20]; !exists {
		t.Fatal("target should not die immediately")
	}

	combatTick(s, 250)

	if _, exists := s.entities[20]; exists {
		t.Fatalf("expected enemy unit 20 to be destroyed after chase")
	}
}

func TestAttackRespectsCooldown(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	putUnit(t, s, 10, 1, 5, 5)
	victim := putUnit(t, s, 20, 2, 6, 5)
	victim.Health = 1000 // tanky so it survives the test window

	orderAttack(s, clientOf(t, s, 1), 20, 10)

	combatTick(s, 20) // interval is 20 ticks -> exactly one hit

	if got := s.stats.Get(1, StatAttacksMade); got != 1 {
		t.Errorf("expected exactly 1 attack in first 20 ticks, got %v", got)
	}
	if victim.Health != 975 {
		t.Errorf("expected victim hp 975 after one 25-dmg hit, got %d", victim.Health)
	}

	combatTick(s, 20) // second hit at tick ~21
	if got := s.stats.Get(1, StatAttacksMade); got != 2 {
		t.Errorf("expected 2 attacks by tick 40, got %v", got)
	}
}

func TestAttackOutOfRangeDoesNotDamageYet(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	attacker := putUnit(t, s, 10, 1, 2, 5)
	bld := putBuildingWithHealth(t, s, 20, 2, 18, 5, 100) // far away

	orderAttack(s, clientOf(t, s, 1), 20, 10)
	combatTick(s, 1)

	if bld.Health != 100 {
		t.Errorf("expected no damage while out of range, got hp %d", bld.Health)
	}
	if len(attacker.Path) == 0 {
		t.Error("expected attacker to path toward the distant target")
	}
}

func TestBuildingsDoNotAttack(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	gen := putBuildingWithHealth(t, s, 10, 1, 4, 4, 100) // "attacker" building
	victim := putUnit(t, s, 20, 2, 5, 4)

	// Even if we name a building as an attacker, it has no combat entry.
	orderAttack(s, clientOf(t, s, 1), 20, 10)
	combatTick(s, 5)

	if gen.AttackTargetId != 0 {
		t.Errorf("expected building to be non-combatant, got AttackTargetId %d", gen.AttackTargetId)
	}
	if victim.Health != 100 {
		t.Errorf("expected victim untouched by non-combatant building, got hp %d", victim.Health)
	}
}

func TestCannotAttackOwnUnit(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)

	a := putUnit(t, s, 10, 1, 5, 5)
	own := putUnit(t, s, 11, 1, 6, 5)

	orderAttack(s, clientOf(t, s, 1), 11, 10) // attack own unit
	combatTick(s, 5)

	if a.AttackTargetId != 0 {
		t.Errorf("expected no self-attack assignment, got %d", a.AttackTargetId)
	}
	if own.Health != 100 {
		t.Errorf("own unit should be untouched, got hp %d", own.Health)
	}
}

func TestUnitDestroysEnemyBuilding(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	attacker := putUnit(t, s, 10, 1, 3, 2)
	putBuildingWithHealth(t, s, 20, 2, 5, 2, 100) // generator at (5,2)

	orderAttack(s, clientOf(t, s, 1), 20, 10)
	if attacker.AttackTargetId != 20 {
		t.Fatalf("expected attack assignment to building 20, got %d", attacker.AttackTargetId)
	}

	combatTick(s, 250)

	if _, exists := s.entities[20]; exists {
		t.Fatalf("expected enemy building 20 to be destroyed")
	}
	if got := s.stats.Get(1, StatBuildingsDestroyed); got != 1 {
		t.Errorf("expected buildings_destroyed 1 for player 1, got %v", got)
	}
	if got := s.stats.Get(2, StatBuildingsLost); got != 1 {
		t.Errorf("expected buildings_lost 1 for player 2, got %v", got)
	}
}
