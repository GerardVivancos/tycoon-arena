package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultBalanceConfig(t *testing.T) {
	cfg := DefaultBalanceConfig()
	if cfg.StartingMoney != StartingMoney || cfg.StartingUnits != 5 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	w, ok := cfg.Units["worker"]
	if !ok || w.Damage != 25 || w.Range != 1 || w.IntervalTicks != 20 || w.Health != 100 {
		t.Fatalf("unexpected worker defaults: %+v ok=%v", w, ok)
	}
	g, ok := cfg.Buildings["generator"]
	if !ok || g.Cost != BuildingCost || g.Income != GeneratorIncome || g.Width != 2 || g.Height != 2 {
		t.Fatalf("unexpected generator defaults: %+v ok=%v", g, ok)
	}
}

func TestLoadBalanceConfigMissingFileUsesDefaults(t *testing.T) {
	cfg, err := LoadBalanceConfig(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg.StartingMoney != StartingMoney {
		t.Fatalf("expected defaults, got %+v", cfg)
	}
}

func TestLoadBalanceConfigOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "balance.json")
	data := `{
	  "startingMoney": 250,
	  "startingUnits": 3,
	  "units": {"worker": {"health": 80, "damage": 10, "range": 4, "intervalTicks": 12, "speed": 5}},
	  "buildings": {"generator": {"cost": 75, "health": 150, "income": 20, "width": 2, "height": 2}}
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBalanceConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.StartingMoney != 250 || cfg.StartingUnits != 3 {
		t.Fatalf("scalar overrides failed: %+v", cfg)
	}
	if u := cfg.Units["worker"]; u.Health != 80 || u.Damage != 10 || u.Range != 4 || u.IntervalTicks != 12 || u.Speed != 5 {
		t.Fatalf("worker override failed: %+v", u)
	}
	if g := cfg.Buildings["generator"]; g.Cost != 75 || g.Health != 150 || g.Income != 20 {
		t.Fatalf("generator override failed: %+v", g)
	}
}

func TestUnitSpeedFallback(t *testing.T) {
	s := &GameServer{} // zero-value balance (e.g. hand-built test server)
	if got := s.unitSpeed("worker"); got != MovementSpeed {
		t.Fatalf("zero balance speed = %v, want %v", got, MovementSpeed)
	}

	s.balance = DefaultBalanceConfig()
	if got := s.unitSpeed("worker"); got != s.balance.MovementSpeed {
		t.Fatalf("default speed = %v, want %v", got, s.balance.MovementSpeed)
	}

	s.balance.Units["sniper"] = UnitBalance{Speed: 1.5}
	if got := s.unitSpeed("sniper"); got != 1.5 {
		t.Fatalf("configured speed = %v, want 1.5", got)
	}
}

func TestBuildUsesBalanceCostAndHealth(t *testing.T) {
	// Unaffordable: cost above money → no building.
	s := combatTestServer()
	s.balance.Buildings["generator"] = BuildingBalance{Cost: 1000, Health: 300, Income: GeneratorIncome, Width: 2, Height: 2}
	s.mu.Lock()
	s.clients[1] = &Client{Id: 1, Money: 100}
	s.mu.Unlock()
	s.handleBuildCommand(buildCmd("generator", 3, 3), clientOf(t, s, 1))
	if len(s.entities) != 0 {
		t.Fatalf("expected no building (unaffordable), got %d entities", len(s.entities))
	}

	// Affordable: created with config HP, and cost deducted.
	s2 := combatTestServer()
	s2.balance.Buildings["generator"] = BuildingBalance{Cost: 40, Health: 300, Income: GeneratorIncome, Width: 2, Height: 2}
	s2.mu.Lock()
	s2.clients[1] = &Client{Id: 1, Money: 100}
	s2.mu.Unlock()
	s2.handleBuildCommand(buildCmd("generator", 3, 3), clientOf(t, s2, 1))

	var b *Entity
	for _, e := range s2.entities {
		if e.Type == "generator" {
			b = e
		}
	}
	if b == nil {
		t.Fatal("expected generator to be created")
	}
	if b.Health != 300 || b.MaxHealth != 300 {
		t.Fatalf("expected config HP 300, got %d/%d", b.Health, b.MaxHealth)
	}
	if c := clientOf(t, s2, 1); c.Money != 60 {
		t.Fatalf("expected money 100-40=60, got %v", c.Money)
	}
}

func TestBuildBalanceInfo(t *testing.T) {
	s := combatTestServer()
	s.balance.StartingMoney = 250
	s.balance.Buildings["generator"] = BuildingBalance{Cost: 40, Health: 300, Income: GeneratorIncome, Width: 2, Height: 2}
	info := s.buildBalanceInfo()
	if info.StartingMoney != 250 || info.BuildingCost != 40 {
		t.Fatalf("balance info scalars wrong: %+v", info)
	}
	if info.UnitHealth["worker"] != 100 {
		t.Fatalf("worker HP not surfaced: %+v", info.UnitHealth)
	}
	if info.BuildingHealth["generator"] != 300 {
		t.Fatalf("generator HP not surfaced: %+v", info.BuildingHealth)
	}
}

func buildCmd(buildingType string, x, y int) Command {
	return Command{Type: "build", Data: map[string]any{
		"buildingType": buildingType,
		"tileX":        float64(x),
		"tileY":        float64(y),
	}}
}
