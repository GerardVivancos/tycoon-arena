package main

import (
	"encoding/json"
	"os"
)

// UnitBalance holds tunable per-unit-type attributes (Phase 5 data-driven
// balance). A unit type with no entry is non-combatant (see unitBalance).
type UnitBalance struct {
	Health        int32   `json:"health"`
	Damage        int32   `json:"damage"`
	Range         int     `json:"range"`         // attack range in tiles (Manhattan to footprint)
	IntervalTicks uint64  `json:"intervalTicks"` // ticks between attacks
	Speed         float32 `json:"speed"`         // tiles/second; 0 → fall back to BalanceConfig.MovementSpeed
}

// BuildingBalance holds tunable per-building-type attributes.
type BuildingBalance struct {
	Cost   int32   `json:"cost"`
	Health int32   `json:"health"`
	Income float32 `json:"income"` // money per second; 0 → non-generating
	Width  int     `json:"width"`  // footprint in tiles
	Height int     `json:"height"` // footprint in tiles
}

// BalanceConfig groups all data-driven gameplay tuning. Loaded from
// maps/balance.json when present; otherwise DefaultBalanceConfig() is used so
// behavior and the existing test suite are unchanged when the file is absent.
type BalanceConfig struct {
	StartingMoney    float32                    `json:"startingMoney"`
	StartingUnits    int                        `json:"startingUnits"`
	StartingSoldiers int                        `json:"startingSoldiers"`
	MovementSpeed    float32                    `json:"movementSpeed"` // default unit speed (tiles/sec)
	Units            map[string]UnitBalance     `json:"units"`
	Buildings        map[string]BuildingBalance `json:"buildings"`
}

// DefaultBalanceConfig reproduces the historical hardcoded constants so that
// running without maps/balance.json behaves exactly as before Phase 5. The
// "soldier" unit and StartingSoldiers are new in Phase 5 (0 by default, so the
// default match is unchanged; maps/balance.json enables them).
func DefaultBalanceConfig() BalanceConfig {
	return BalanceConfig{
		StartingMoney:    StartingMoney,
		StartingUnits:    5,
		StartingSoldiers: 0,
		MovementSpeed:    MovementSpeed,
		Units: map[string]UnitBalance{
			"worker":  {Health: 100, Damage: 25, Range: 1, IntervalTicks: 20, Speed: MovementSpeed},
			"soldier": {Health: 120, Damage: 15, Range: 3, IntervalTicks: 15, Speed: 3.5},
		},
		Buildings: map[string]BuildingBalance{
			"generator": {Cost: BuildingCost, Health: 100, Income: GeneratorIncome, Width: 2, Height: 2},
		},
	}
}

// LoadBalanceConfig reads a balance JSON file. A missing file is not an error
// (defaults are returned). On malformed JSON an error is returned alongside
// fresh defaults so callers can log and continue. Maps are always non-nil.
func LoadBalanceConfig(path string) (BalanceConfig, error) {
	cfg := DefaultBalanceConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultBalanceConfig(), err
	}
	if cfg.Units == nil {
		cfg.Units = DefaultBalanceConfig().Units
	}
	if cfg.Buildings == nil {
		cfg.Buildings = DefaultBalanceConfig().Buildings
	}
	return cfg, nil
}

// unitBalance returns the combat/health/spawn attributes for a unit type and
// whether it exists (a missing type is non-combatant and non-spawnable).
func (s *GameServer) unitBalance(entityType string) (UnitBalance, bool) {
	u, ok := s.balance.Units[entityType]
	return u, ok
}

// isUnitType reports whether an entity of the given type is a moveable unit
// (as opposed to a building): any configured unit type plus the legacy
// worker/unit/player labels used by scenarios and tests.
func (s *GameServer) isUnitType(entityType string) bool {
	switch entityType {
	case "worker", "unit", "player":
		return true
	}
	_, ok := s.unitBalance(entityType)
	return ok
}

// newUnitEntity builds a unit entity using HP from the balance config.
func (s *GameServer) newUnitEntity(entityType string, owner, id uint32, x, y int) *Entity {
	hp := int32(100)
	if u, ok := s.unitBalance(entityType); ok {
		hp = u.Health
	}
	return &Entity{
		Id: id, OwnerId: owner, Type: entityType,
		TileX: x, TileY: y, TargetTileX: x, TargetTileY: y,
		MoveProgress: 0.0, Health: hp, MaxHealth: hp,
	}
}

// buildingBalance returns attributes for a building type and whether it exists.
func (s *GameServer) buildingBalance(buildingType string) (BuildingBalance, bool) {
	b, ok := s.balance.Buildings[buildingType]
	return b, ok
}

// unitSpeed returns a unit type's movement speed, falling back to the global
// default and finally the historical const when unset (e.g. a hand-built
// GameServer with a zero-value balance in tests).
func (s *GameServer) unitSpeed(entityType string) float32 {
	if u, ok := s.balance.Units[entityType]; ok && u.Speed > 0 {
		return u.Speed
	}
	if s.balance.MovementSpeed > 0 {
		return s.balance.MovementSpeed
	}
	return MovementSpeed
}

// buildBalanceInfo projects the server's balance into the client-facing
// BalanceInfo sent in the welcome message (Phase 5).
func (s *GameServer) buildBalanceInfo() BalanceInfo {
	info := BalanceInfo{
		StartingMoney:  s.balance.StartingMoney,
		BuildingCost:   BuildingCost,
		UnitHealth:     make(map[string]int32, len(s.balance.Units)),
		BuildingHealth: make(map[string]int32, len(s.balance.Buildings)),
	}
	if g, ok := s.balance.Buildings["generator"]; ok {
		info.BuildingCost = g.Cost
	}
	for t, u := range s.balance.Units {
		info.UnitHealth[t] = u.Health
	}
	for t, b := range s.balance.Buildings {
		info.BuildingHealth[t] = b.Health
	}
	return info
}
