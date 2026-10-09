package main

import (
	"testing"
)

func TestEntityVisibleChanged(t *testing.T) {
	base := Entity{Id: 1, OwnerId: 1, Type: "worker", TileX: 0, TileY: 0, Health: 100, MaxHealth: 100}

	sameExceptPath := base
	sameExceptPath.Path = []TilePosition{{X: 5, Y: 5}} // json:"-": must not count
	if entityVisibleChanged(base, sameExceptPath) {
		t.Fatal("path-only difference should not be visible to clients")
	}

	moved := base
	moved.TileX = 2
	if !entityVisibleChanged(base, moved) {
		t.Fatal("tile change should be visible")
	}

	hurt := base
	hurt.Health = 50
	if !entityVisibleChanged(base, hurt) {
		t.Fatal("health change should be visible")
	}
}

func TestSnapshotFullThenDelta(t *testing.T) {
	s := combatTestServer()
	s.clients[1] = &Client{Id: 1, Name: "p", Money: 100}
	s.entities[1] = &Entity{Id: 1, OwnerId: 1, Type: "worker", TileX: 0, TileY: 0, TargetTileX: 0, TargetTileY: 0, Health: 100, MaxHealth: 100}
	s.entities[2] = &Entity{Id: 2, OwnerId: 1, Type: "generator", TileX: 5, TileY: 5, Health: 100, MaxHealth: 100, FootprintWidth: 2, FootprintHeight: 2}

	// First build: full (no baseline yet).
	s.mu.Lock()
	s.tick = 1
	full := s.buildSnapshotLocked()
	s.mu.Unlock()
	if full.BaselineTick != 0 {
		t.Fatalf("first snapshot should be full, got baseline %d", full.BaselineTick)
	}
	if len(full.Entities) != 2 {
		t.Fatalf("full snapshot should contain all entities, got %d", len(full.Entities))
	}

	// No changes: delta is empty and marks itself as a delta.
	s.mu.Lock()
	s.tick = 3
	delta := s.buildSnapshotLocked()
	s.mu.Unlock()
	if delta.BaselineTick == 0 {
		t.Fatal("expected a delta snapshot (baselineTick != 0)")
	}
	if len(delta.Entities) != 0 {
		t.Fatalf("delta with no changes should carry no entities, got %d", len(delta.Entities))
	}
	if len(delta.Removed) != 0 {
		t.Fatalf("no removals expected, got %v", delta.Removed)
	}

	// Move the worker: only it appears in the delta.
	s.entities[1].TileX = 3
	s.mu.Lock()
	s.tick = 4
	delta2 := s.buildSnapshotLocked()
	s.mu.Unlock()
	if len(delta2.Entities) != 1 || delta2.Entities[0].Id != 1 {
		t.Fatalf("delta should contain only the moved worker, got %+v", delta2.Entities)
	}

	// Delete the generator: it is listed as removed.
	delete(s.entities, 2)
	s.mu.Lock()
	s.tick = 5
	delta3 := s.buildSnapshotLocked()
	s.mu.Unlock()
	if len(delta3.Removed) != 1 || delta3.Removed[0] != 2 {
		t.Fatalf("expected removed [2], got %v", delta3.Removed)
	}
}

func TestSnapshotPeriodicFull(t *testing.T) {
	s := combatTestServer()
	s.clients[1] = &Client{Id: 1}
	s.entities[1] = &Entity{Id: 1, Type: "worker", Health: 100, MaxHealth: 100}

	// Establish a baseline.
	s.mu.Lock()
	s.tick = 1
	s.buildSnapshotLocked()
	s.mu.Unlock()

	// At a multiple of the interval a full snapshot is forced.
	s.mu.Lock()
	s.tick = FullSnapshotInterval
	snap := s.buildSnapshotLocked()
	s.mu.Unlock()
	if snap.BaselineTick != 0 {
		t.Fatalf("expected periodic full at tick %d, got baseline %d", FullSnapshotInterval, snap.BaselineTick)
	}
	if len(snap.Entities) != 1 {
		t.Fatalf("periodic full should include all entities, got %d", len(snap.Entities))
	}
}

func TestSnapshotForceFullOnJoin(t *testing.T) {
	s := combatTestServer()
	s.clients[1] = &Client{Id: 1}
	s.entities[1] = &Entity{Id: 1, Type: "worker", Health: 100, MaxHealth: 100}

	// Establish a baseline first (so a delta would otherwise be produced).
	s.mu.Lock()
	s.tick = 1
	s.buildSnapshotLocked()
	s.mu.Unlock()

	// Simulate a join: force a full snapshot next build.
	s.mu.Lock()
	s.forceFullSnapshot = true
	s.tick = 2
	snap := s.buildSnapshotLocked()
	forced := s.forceFullSnapshot
	s.mu.Unlock()

	if snap.BaselineTick != 0 {
		t.Fatal("join should force a full snapshot")
	}
	if len(snap.Entities) != 1 {
		t.Fatalf("forced full should include all entities, got %d", len(snap.Entities))
	}
	if forced {
		t.Fatal("forceFullSnapshot should be cleared after a build")
	}
}
