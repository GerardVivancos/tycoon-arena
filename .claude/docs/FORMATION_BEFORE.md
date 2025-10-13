# Formation Code Snapshot - BEFORE Refactor

**Date:** 2025-10-13
**Purpose:** Snapshot of original formation code before orientation refactor
**Source:** `server/main.go` lines 940-1096

---

## calculateFormation (Entry Point)

```go
func (s *GameServer) calculateFormation(formation string, centerX, centerY, numUnits int) []TilePosition {
	switch formation {
	case "box":
		return s.calculateBoxFormation(centerX, centerY, numUnits)
	case "line":
		return s.calculateLineFormation(centerX, centerY, numUnits)
	case "spread":
		return s.calculateSpiralFormation(centerX, centerY, numUnits)
	default:
		// Default to box formation
		return s.calculateBoxFormation(centerX, centerY, numUnits)
	}
}
```

---

## calculateBoxFormation

```go
// calculateBoxFormation creates a grid pattern (√n × √n arrangement)
func (s *GameServer) calculateBoxFormation(centerX, centerY, numUnits int) []TilePosition {
	positions := make([]TilePosition, 0, numUnits)

	// Calculate grid dimensions (roughly square)
	gridSize := int(math.Ceil(math.Sqrt(float64(numUnits))))

	// Center the grid around the target point
	startX := centerX - gridSize/2
	startY := centerY - gridSize/2

	for i := 0; i < numUnits; i++ {
		row := i / gridSize
		col := i % gridSize

		tileX := startX + col
		tileY := startY + row

		// Check if tile is passable (includes bounds, terrain, and buildings)
		if !s.isTilePassable(tileX, tileY) {
			continue
		}

		positions = append(positions, TilePosition{X: tileX, Y: tileY})
	}

	// If we couldn't find enough positions, find nearest passable tiles
	// This prevents unit stacking when formations are partially blocked
	spiralOffset := 0
	for len(positions) < numUnits {
		// Try positions in a spiral around center
		searchX := centerX + spiralOffset
		searchY := centerY + spiralOffset
		fallbackPos := s.findNearestPassableTile(searchX, searchY, 10)

		// Check if this position is already in the list
		isDuplicate := false
		for _, pos := range positions {
			if pos.X == fallbackPos.X && pos.Y == fallbackPos.Y {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			positions = append(positions, fallbackPos)
		}

		spiralOffset++
		if spiralOffset > 20 {
			// Give up and allow duplicates rather than infinite loop
			positions = append(positions, fallbackPos)
		}
	}

	return positions
}
```

---

## calculateLineFormation

```go
// calculateLineFormation creates a horizontal line
func (s *GameServer) calculateLineFormation(centerX, centerY, numUnits int) []TilePosition {
	positions := make([]TilePosition, 0, numUnits)

	// Center the line around the target point
	startX := centerX - numUnits/2

	for i := 0; i < numUnits; i++ {
		tileX := startX + i
		tileY := centerY

		// Check if tile is passable (includes bounds, terrain, and buildings)
		if !s.isTilePassable(tileX, tileY) {
			continue
		}

		positions = append(positions, TilePosition{X: tileX, Y: tileY})
	}

	// If we couldn't find enough positions, find nearest passable tiles
	spiralOffset := 0
	for len(positions) < numUnits {
		// Try positions around center
		searchX := centerX
		searchY := centerY + spiralOffset
		fallbackPos := s.findNearestPassableTile(searchX, searchY, 10)

		// Check if this position is already in the list
		isDuplicate := false
		for _, pos := range positions {
			if pos.X == fallbackPos.X && pos.Y == fallbackPos.Y {
				isDuplicate = true
				break
			}
		}

		if !isDuplicate {
			positions = append(positions, fallbackPos)
		}

		spiralOffset++
		if spiralOffset > 20 {
			// Give up and allow duplicates rather than infinite loop
			positions = append(positions, fallbackPos)
		}
	}

	return positions
}
```

---

## calculateSpiralFormation

```go
// calculateSpiralFormation creates a spiral pattern from center
func (s *GameServer) calculateSpiralFormation(centerX, centerY, numUnits int) []TilePosition {
	positions := make([]TilePosition, 0, numUnits)

	// Start with center if passable
	if s.isTilePassable(centerX, centerY) {
		positions = append(positions, TilePosition{X: centerX, Y: centerY})
	}

	// Spiral outward
	directions := []TilePosition{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} // Right, Down, Left, Up
	x, y := centerX, centerY
	steps := 1

	for len(positions) < numUnits {
		for _, dir := range directions {
			for step := 0; step < steps && len(positions) < numUnits; step++ {
				x += dir.X
				y += dir.Y

				// Check if tile is passable (includes bounds, terrain, and buildings)
				if s.isTilePassable(x, y) {
					positions = append(positions, TilePosition{X: x, Y: y})
				}
			}

			// Increase steps after every 2 directions (right+down, left+up)
			if dir.X == 0 {
				steps++
			}
		}
	}

	return positions
}
```

---

## Usage in handleMoveCommand

```go
// Line 1160-1161 in handleMoveCommand:
// Calculate formation positions
formationPositions := s.calculateFormation(formation, tileX, tileY, len(validUnitIds))
```

---

## Behavior Summary

**Box Formation:**
- √n × √n grid (e.g., 5 units = 3×3 grid with 4 filled)
- Grid is centered on click point
- Always aligned with map axes (no rotation)

**Line Formation:**
- Always horizontal
- Centered on click point horizontally
- No vertical or diagonal lines

**Spiral Formation:**
- Starts at center, spirals outward clockwise
- Centered on click point
- (This behavior is correct and doesn't need changing)

---

## Test Code Snapshot

From `server/game_test.go`:

```go
// TestFormationCalculation tests that formations are calculated correctly
func TestFormationCalculation(t *testing.T) {
	mapData := &MapData{
		Width:          20,
		Height:         20,
		TileSize:       32,
		DefaultTerrain: TerrainType{Type: "grass", Passable: true},
		Tiles:          map[TileCoord]TerrainType{},
		Features:       []Feature{},
		SpawnPoints:    []SpawnPoint{},
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Test box formation for 5 units
	positions := server.calculateFormation("box", 10, 10, 5)

	if len(positions) != 5 {
		t.Errorf("Expected 5 positions, got %d", len(positions))
	}

	// All positions should be unique (no duplicates)
	seen := make(map[string]bool)
	for _, pos := range positions {
		key := formatPos(pos.X, pos.Y)
		if seen[key] {
			t.Errorf("Duplicate position in formation: (%d,%d)", pos.X, pos.Y)
		}
		seen[key] = true
	}

	t.Logf("Box formation: %v", positions)
}
```

**Expected output for 5 units at (10, 10):**
```
Box formation: [{9 9} {10 9} {11 9} {9 10} {10 10}]
```

This shows the center-based positioning (click at 10,10 → center unit at 10,10).

---

## Restoration Instructions

If refactor needs to be reverted:

1. **Restore functions:** Copy the 3 formation functions above back to `server/main.go`
2. **Restore entry point:** Copy `calculateFormation` function
3. **Restore handleMoveCommand call:** Use simple call without direction calculation
4. **Restore test:** Copy `TestFormationCalculation` back to `game_test.go`

**Or simply:** `git revert <commit-hash>` of the refactor commit
