package main

import "testing"

// putUnitType creates a unit of an arbitrary configured type (using HP from the
// balance config) so tests can exercise the soldier and other data-driven units.
func putUnitType(t *testing.T, s *GameServer, id, owner uint32, typ string, x, y int) *Entity {
	t.Helper()
	e := s.newUnitEntity(typ, owner, id, x, y)
	s.mu.Lock()
	s.entities[e.Id] = e
	s.mu.Unlock()
	return e
}

func TestSoldierAttacksFromRange(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	soldier := putUnitType(t, s, 10, 1, "soldier", 0, 0)
	enemy := putUnitType(t, s, 20, 2, "worker", 3, 0) // distance 3 == soldier default range

	orderAttack(s, clientOf(t, s, 1), 20, 10)
	combatTick(s, 1)

	if soldier.TileX != 0 || soldier.TileY != 0 {
		t.Fatalf("soldier should hold at range 3, but moved to (%d,%d)", soldier.TileX, soldier.TileY)
	}
	if enemy.Health != 85 { // 100 - soldier damage (15)
		t.Fatalf("expected one ranged hit (health 85), got %d", enemy.Health)
	}
}

func TestSoldierKillsEnemyUnit(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	putUnitType(t, s, 10, 1, "soldier", 0, 0)
	putUnitType(t, s, 20, 2, "worker", 2, 0)

	orderAttack(s, clientOf(t, s, 1), 20, 10)
	combatTick(s, 200)

	if _, alive := s.entities[20]; alive {
		t.Fatalf("soldier should have killed the enemy worker; health=%d", s.entities[20].Health)
	}
	if got := s.stats.Get(1, StatUnitsKilled); got != 1 {
		t.Errorf("expected units_killed=1 for player 1, got %v", got)
	}
	if got := s.stats.Get(2, StatUnitsLost); got != 1 {
		t.Errorf("expected units_lost=1 for player 2, got %v", got)
	}
}

func TestSoldierCombatIsConfigDriven(t *testing.T) {
	s := combatTestServer()
	addTestClient(t, s, 1)
	addTestClient(t, s, 2)

	// Shrink soldier range to melee: it must close the distance before it can
	// hurt a target that is 3 tiles away (proves combat stats come from config).
	s.balance.Units["soldier"] = UnitBalance{Health: 120, Damage: 15, Range: 1, IntervalTicks: 15, Speed: 3.5}

	soldier := putUnitType(t, s, 10, 1, "soldier", 0, 0)
	enemy := putUnitType(t, s, 20, 2, "worker", 3, 0)

	orderAttack(s, clientOf(t, s, 1), 20, 10)
	combatTick(s, 1)

	if len(soldier.Path) == 0 {
		t.Error("expected soldier to path toward out-of-range target when Range=1")
	}
	if enemy.Health != 100 {
		t.Fatalf("out-of-range soldier should not deal damage yet, health=%d", enemy.Health)
	}
}
