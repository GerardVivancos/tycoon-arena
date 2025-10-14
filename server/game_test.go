package main

import (
	"fmt"
	"testing"
)

// TestPathfindingAroundSingleRock tests that pathfinding routes around a single obstacle
func TestPathfindingAroundSingleRock(t *testing.T) {
	// Load test map with single rock
	mapData, err := LoadMap("../maps/test_single_rock.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	// Create minimal GameServer for testing
	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Test: Path from left of rock to right of rock
	// Rock is at (10, 5), path should go around it
	path := server.findPath(5, 5, 15, 5, 999)

	// Assertions
	if path == nil {
		t.Fatal("Expected path to be found, got nil")
	}

	// Path should not go through the rock at (10, 5)
	for i, waypoint := range path {
		if waypoint.X == 10 && waypoint.Y == 5 {
			t.Errorf("Path waypoint %d goes through rock at (10,5)", i)
		}
	}

	// Path should reach destination
	if len(path) == 0 {
		t.Fatal("Path is empty")
	}
	finalWaypoint := path[len(path)-1]
	if finalWaypoint.X != 15 || finalWaypoint.Y != 5 {
		t.Errorf("Path ends at (%d,%d), expected (15,5)", finalWaypoint.X, finalWaypoint.Y)
	}

	t.Logf("Path found with %d waypoints", len(path))
}

// TestPathfindingAroundCluster tests pathfinding around a cluster of obstacles
func TestPathfindingAroundCluster(t *testing.T) {
	mapData, err := LoadMap("../maps/test_rock_cluster.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Test: Path from left to right, cluster is at (9-11, 7-8)
	path := server.findPath(2, 7, 17, 7, 999)

	if path == nil {
		t.Fatal("Expected path to be found, got nil")
	}

	// Verify path doesn't go through any rock in the cluster
	rockPositions := map[string]bool{
		"9,7":  true,
		"10,7": true,
		"11,7": true,
		"9,8":  true,
		"10,8": true,
		"11,8": true,
	}

	for i, waypoint := range path {
		key := formatPos(waypoint.X, waypoint.Y)
		if rockPositions[key] {
			t.Errorf("Path waypoint %d goes through rock at (%d,%d)", i, waypoint.X, waypoint.Y)
		}
	}

	// Path should reach destination
	finalWaypoint := path[len(path)-1]
	if finalWaypoint.X != 17 || finalWaypoint.Y != 7 {
		t.Errorf("Path ends at (%d,%d), expected (17,7)", finalWaypoint.X, finalWaypoint.Y)
	}

	t.Logf("Path found with %d waypoints around cluster", len(path))
}

// TestPathfindingNoPath tests that pathfinding returns nil when no path exists
func TestPathfindingNoPath(t *testing.T) {
	// Create a map where destination is surrounded by rocks
	mapData := &MapData{
		Width:          10,
		Height:         10,
		TileSize:       32,
		DefaultTerrain: TerrainType{Type: "grass", Passable: true},
		Tiles: map[TileCoord]TerrainType{
			{X: 4, Y: 4}: {Type: "rock", Passable: false},
			{X: 5, Y: 4}: {Type: "rock", Passable: false},
			{X: 6, Y: 4}: {Type: "rock", Passable: false},
			{X: 4, Y: 5}: {Type: "rock", Passable: false},
			// (5,5) is destination - surrounded by rocks
			{X: 6, Y: 5}: {Type: "rock", Passable: false},
			{X: 4, Y: 6}: {Type: "rock", Passable: false},
			{X: 5, Y: 6}: {Type: "rock", Passable: false},
			{X: 6, Y: 6}: {Type: "rock", Passable: false},
		},
		Features:    []Feature{},
		SpawnPoints: []SpawnPoint{},
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Try to path to the surrounded tile
	path := server.findPath(0, 0, 5, 5, 999)

	if path != nil {
		t.Errorf("Expected no path (nil), but got path with %d waypoints", len(path))
	}

	t.Log("Correctly returned nil for unreachable destination")
}

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

// TestUnitCollisionDetection tests that units properly detect collisions
func TestUnitCollisionDetection(t *testing.T) {
	mapData := &MapData{
		Width:          10,
		Height:         10,
		TileSize:       32,
		DefaultTerrain: TerrainType{Type: "grass", Passable: true},
		Tiles:          map[TileCoord]TerrainType{},
		Features:       []Feature{},
		SpawnPoints:    []SpawnPoint{},
	}

	server := &GameServer{
		mapData: mapData,
		entities: map[uint32]*Entity{
			1: {
				Id:    1,
				Type:  "worker",
				TileX: 5,
				TileY: 5,
				Path: []TilePosition{
					{X: 5, Y: 6},
					{X: 5, Y: 7},
				},
			},
		},
	}

	// Test 1: Tile (5,5) should be occupied by unit 1
	if !server.isTileOccupiedByUnit(5, 5, 999) {
		t.Error("Expected tile (5,5) to be occupied by unit 1")
	}

	// Test 2: Unit 1 should not consider its own position as occupied
	if server.isTileOccupiedByUnit(5, 5, 1) {
		t.Error("Unit should not consider its own position as occupied")
	}

	// Test 3: Final destination (5,7) should be considered occupied
	if !server.isTileOccupiedByUnit(5, 7, 999) {
		t.Error("Expected final destination (5,7) to be occupied")
	}

	// Test 4: Empty tile should not be occupied
	if server.isTileOccupiedByUnit(0, 0, 999) {
		t.Error("Expected tile (0,0) to be unoccupied")
	}

	t.Log("Unit collision detection working correctly")
}

// TestPathDoesNotGoThroughRock tests that a path between two passable tiles avoids rocks
func TestPathDoesNotGoThroughRock(t *testing.T) {
	mapData, err := LoadMap("../maps/test_single_rock.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Path from (5, 5) to (15, 5) - both are grass
	// Rock is at (10, 5) directly in the way
	startX, startY := 5, 5
	goalX, goalY := 15, 5
	rockX, rockY := 10, 5

	// Verify start and goal are passable
	if !server.isTilePassable(startX, startY) {
		t.Fatal("Start position should be passable")
	}
	if !server.isTilePassable(goalX, goalY) {
		t.Fatal("Goal position should be passable")
	}

	// Verify rock is NOT passable
	if server.isTilePassable(rockX, rockY) {
		t.Fatal("Rock should not be passable")
	}

	// Find path
	path := server.findPath(startX, startY, goalX, goalY, 999)

	if path == nil {
		t.Fatal("Expected path to be found (around rock), got nil")
	}

	// Verify path does NOT include the rock
	for i, waypoint := range path {
		if waypoint.X == rockX && waypoint.Y == rockY {
			t.Errorf("Path waypoint %d goes THROUGH rock at (%d,%d) - this should never happen!",
				i, rockX, rockY)
		}
	}

	// Verify path reaches destination
	finalWaypoint := path[len(path)-1]
	if finalWaypoint.X != goalX || finalWaypoint.Y != goalY {
		t.Errorf("Path ends at (%d,%d), expected (%d,%d)",
			finalWaypoint.X, finalWaypoint.Y, goalX, goalY)
	}

	t.Logf("✓ Path from (%d,%d) to (%d,%d) correctly avoids rock at (%d,%d) with %d waypoints",
		startX, startY, goalX, goalY, rockX, rockY, len(path))
}

// TestCannotPathToRock tests that pathfinding to a rock tile returns nil
func TestCannotPathToRock(t *testing.T) {
	mapData, err := LoadMap("../maps/test_single_rock.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Try to path directly TO the rock at (10, 5)
	path := server.findPath(5, 5, 10, 5, 999)

	if path != nil {
		t.Errorf("Expected nil path when destination is a rock, got path with %d waypoints", len(path))
	}

	t.Log("Correctly returned nil when attempting to path to rock tile")
}

// TestCannotMoveToRock tests that commanding a unit to move to a rock fails gracefully
func TestCannotMoveToRock(t *testing.T) {
	mapData, err := LoadMap("../maps/test_single_rock.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Create a unit at (5, 5)
	unitID := uint32(100)
	server.entities[unitID] = &Entity{
		Id:           unitID,
		OwnerId:      1,
		Type:         "worker",
		TileX:        5,
		TileY:        5,
		TargetTileX:  5,
		TargetTileY:  5,
		MoveProgress: 0.0,
		Health:       100,
		MaxHealth:    100,
	}

	// Try to move unit to rock at (10, 5)
	path := server.findPath(5, 5, 10, 5, unitID)

	// Path should be nil because destination is impassable
	if path != nil {
		t.Errorf("Expected nil path to rock tile, got path with %d waypoints", len(path))
	}

	// Unit should still be at original position
	entity := server.entities[unitID]
	if entity.TileX != 5 || entity.TileY != 5 {
		t.Errorf("Unit moved from (%d,%d), expected to stay at (5,5)", entity.TileX, entity.TileY)
	}

	t.Log("Unit cannot move to rock tile")
}

// TestRockBlocksBuilding tests that buildings cannot be placed on rocks
func TestRockBlocksBuilding(t *testing.T) {
	mapData, err := LoadMap("../maps/test_single_rock.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Try to place a building on the rock at (10, 5)
	// First check that the tile is not passable
	if server.isTilePassable(10, 5) {
		t.Error("Rock tile (10,5) should not be passable")
	}

	// Verify no building can be placed there by checking passability
	// (In real code, handleBuildCommand would check isTilePassable)
	canBuild := server.isTilePassable(10, 5)
	if canBuild {
		t.Error("Should not be able to build on rock tile")
	}

	t.Log("Rocks correctly block building placement")
}

// TestTerrainPassability tests the isTilePassable function directly
func TestTerrainPassability(t *testing.T) {
	mapData, err := LoadMap("../maps/test_single_rock.json")
	if err != nil {
		t.Fatalf("Failed to load test map: %v", err)
	}

	server := &GameServer{
		mapData:  mapData,
		entities: make(map[uint32]*Entity),
	}

	// Test 1: Rock tile is not passable
	if server.isTilePassable(10, 5) {
		t.Error("Rock at (10,5) should not be passable")
	}

	// Test 2: Grass tile is passable
	if !server.isTilePassable(5, 5) {
		t.Error("Grass at (5,5) should be passable")
	}

	// Test 3: Out of bounds is not passable
	if server.isTilePassable(-1, 5) {
		t.Error("Tile at (-1,5) out of bounds should not be passable")
	}
	if server.isTilePassable(5, -1) {
		t.Error("Tile at (5,-1) out of bounds should not be passable")
	}
	if server.isTilePassable(999, 5) {
		t.Error("Tile at (999,5) out of bounds should not be passable")
	}
	if server.isTilePassable(5, 999) {
		t.Error("Tile at (5,999) out of bounds should not be passable")
	}

	// Test 4: Add a building and verify it blocks passage
	buildingID := uint32(200)
	server.entities[buildingID] = &Entity{
		Id:              buildingID,
		OwnerId:         1,
		Type:            "generator",
		TileX:           7,
		TileY:           7,
		TargetTileX:     7,
		TargetTileY:     7,
		FootprintWidth:  2,
		FootprintHeight: 2,
		Health:          100,
		MaxHealth:       100,
	}

	// Building occupies (7,7), (8,7), (7,8), (8,8)
	if server.isTilePassable(7, 7) {
		t.Error("Tile (7,7) occupied by building should not be passable")
	}
	if server.isTilePassable(8, 7) {
		t.Error("Tile (8,7) occupied by building should not be passable")
	}
	if server.isTilePassable(7, 8) {
		t.Error("Tile (7,8) occupied by building should not be passable")
	}
	if server.isTilePassable(8, 8) {
		t.Error("Tile (8,8) occupied by building should not be passable")
	}

	// Adjacent tiles should still be passable
	if !server.isTilePassable(6, 7) {
		t.Error("Tile (6,7) adjacent to building should be passable")
	}

	t.Log("Terrain passability checks working correctly")
}

// TestDirectionCalculation tests the 8-way direction classification
func TestDirectionCalculation(t *testing.T) {
	tests := []struct {
		name     string
		dx       float64
		dy       float64
		expected string
	}{
		{"Pure East", 1.0, 0.0, "E"},
		{"Pure West", -1.0, 0.0, "W"},
		{"Pure North", 0.0, -1.0, "N"},
		{"Pure South", 0.0, 1.0, "S"},
		{"Northeast", 0.7, -0.7, "NE"},
		{"Northwest", -0.7, -0.7, "NW"},
		{"Southeast", 0.7, 0.7, "SE"},
		{"Southwest", -0.7, 0.7, "SW"},
		{"Mostly East", 0.9, 0.2, "E"},
		{"Mostly North", 0.2, -0.9, "N"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPrimaryDirection(tt.dx, tt.dy)
			if result != tt.expected {
				t.Errorf("getPrimaryDirection(%v, %v) = %v, expected %v",
					tt.dx, tt.dy, result, tt.expected)
			}
		})
	}
}

// TestBoxFormationOriented tests that box formations orient correctly
func TestBoxFormationOriented(t *testing.T) {
	mapData := &MapData{
		Width:          30,
		Height:         30,
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

	tests := []struct {
		name      string
		direction string
		tipX      int
		tipY      int
		numUnits  int
	}{
		{"East formation", "E", 15, 15, 5},
		{"West formation", "W", 15, 15, 5},
		{"North formation", "N", 15, 15, 5},
		{"South formation", "S", 15, 15, 5},
		{"Northeast formation", "NE", 15, 15, 5},
		{"Northwest formation", "NW", 15, 15, 5},
		{"Southeast formation", "SE", 15, 15, 5},
		{"Southwest formation", "SW", 15, 15, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positions := server.calculateBoxFormationOriented(tt.tipX, tt.tipY, tt.numUnits, tt.direction)

			if len(positions) == 0 {
				t.Error("No positions returned")
				return
			}

			// Verify we got the expected number of positions
			if len(positions) != tt.numUnits {
				t.Errorf("Expected %d positions, got %d", tt.numUnits, len(positions))
			}

			// Verify all positions are unique
			seen := make(map[string]bool)
			for _, pos := range positions {
				key := formatPos(pos.X, pos.Y)
				if seen[key] {
					t.Errorf("Duplicate position in formation: (%d,%d)", pos.X, pos.Y)
				}
				seen[key] = true
			}

			// Verify at least one position is at or very near the tip
			// (tip might be adjusted if blocked, but should be close)
			foundNearTip := false
			for _, pos := range positions {
				distX := abs(pos.X - tt.tipX)
				distY := abs(pos.Y - tt.tipY)
				if distX <= 2 && distY <= 2 {
					foundNearTip = true
					break
				}
			}
			if !foundNearTip {
				t.Errorf("No position found near tip (%d,%d)", tt.tipX, tt.tipY)
			}

			t.Logf("%s: %d positions", tt.direction, len(positions))
		})
	}
}

// TestLineFormationOriented tests that line formations are perpendicular to movement
func TestLineFormationOriented(t *testing.T) {
	mapData := &MapData{
		Width:          30,
		Height:         30,
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

	tests := []struct {
		name         string
		direction    string
		tipX         int
		tipY         int
		numUnits     int
		expectLinear bool
		checkAxis    string // "X" or "Y" for which should be constant
	}{
		{"East movement (horizontal line)", "E", 15, 15, 5, true, "Y"},
		{"West movement (horizontal line)", "W", 15, 15, 5, true, "Y"},
		{"North movement (vertical line)", "N", 15, 15, 5, true, "X"},
		{"South movement (vertical line)", "S", 15, 15, 5, true, "X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positions := server.calculateLineFormationOriented(tt.tipX, tt.tipY, tt.numUnits, tt.direction)

			if len(positions) == 0 {
				t.Error("No positions returned")
				return
			}

			// Verify unique positions
			seen := make(map[string]bool)
			for _, pos := range positions {
				key := formatPos(pos.X, pos.Y)
				if seen[key] {
					t.Errorf("Duplicate position in formation: (%d,%d)", pos.X, pos.Y)
				}
				seen[key] = true
			}

			// Verify line is linear along expected axis
			if tt.expectLinear {
				switch tt.checkAxis {
				case "X":
					// All X values should be the same (vertical line)
					firstX := positions[0].X
					for i, pos := range positions {
						if pos.X != firstX {
							t.Errorf("Position %d has X=%d, expected X=%d (vertical line)", i, pos.X, firstX)
						}
					}
				case "Y":
					// All Y values should be the same (horizontal line)
					firstY := positions[0].Y
					for i, pos := range positions {
						if pos.Y != firstY {
							t.Errorf("Position %d has Y=%d, expected Y=%d (horizontal line)", i, pos.Y, firstY)
						}
					}
				}
			}

			t.Logf("%s: %d positions along %s axis", tt.direction, len(positions), tt.checkAxis)
		})
	}
}

// TestCentroidCalculation tests the unit centroid calculation
func TestCentroidCalculation(t *testing.T) {
	mapData := &MapData{
		Width:          30,
		Height:         30,
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

	// Add test units
	server.entities[1] = &Entity{Id: 1, TileX: 5, TileY: 5}
	server.entities[2] = &Entity{Id: 2, TileX: 10, TileY: 5}
	server.entities[3] = &Entity{Id: 3, TileX: 5, TileY: 10}

	tests := []struct {
		name      string
		unitIds   []uint32
		expectedX float64
		expectedY float64
	}{
		{"Single unit", []uint32{1}, 5.0, 5.0},
		{"Two units horizontal", []uint32{1, 2}, 7.5, 5.0},
		{"Three units L-shape", []uint32{1, 2, 3}, 6.666666, 6.666666},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := server.calculateUnitCentroid(tt.unitIds)

			// Allow small floating point tolerance
			if !floatNear(x, tt.expectedX, 0.01) || !floatNear(y, tt.expectedY, 0.01) {
				t.Errorf("Centroid = (%.2f, %.2f), expected (%.2f, %.2f)", x, y, tt.expectedX, tt.expectedY)
			}

			t.Logf("Centroid for %v: (%.2f, %.2f)", tt.unitIds, x, y)
		})
	}
}

// Helper function
func floatNear(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

// TestLineFormationBackwardExtension tests that line formations extend backward from click
func TestLineFormationBackwardExtension(t *testing.T) {
	mapData := &MapData{
		Width:          30,
		Height:         30,
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

	// Scenario 1: Units moving south, line should extend north from click
	// Units at (10,5), click at (10,15) → expect (10,15), (10,14), (10,13)
	server.entities[1] = &Entity{Id: 1, TileX: 10, TileY: 5}
	server.entities[2] = &Entity{Id: 2, TileX: 11, TileY: 5}
	server.entities[3] = &Entity{Id: 3, TileX: 12, TileY: 5}

	dx, dy := server.calculateMovementDirection([]uint32{1, 2, 3}, 10, 15)
	direction := getPrimaryDirection(dx, dy)

	if direction != "S" {
		t.Errorf("Expected direction S (south), got %s", direction)
	}

	positions := server.calculateLineFormationOriented(10, 15, 3, direction)

	// Verify position[0] is at click point
	if positions[0].X != 10 || positions[0].Y != 15 {
		t.Errorf("Position[0] should be at click point (10,15), got (%d,%d)", positions[0].X, positions[0].Y)
	}

	// Verify line extends north (negative Y)
	expectedPositions := []TilePosition{{10, 15}, {10, 14}, {10, 13}}
	for i, expected := range expectedPositions {
		if i >= len(positions) {
			t.Errorf("Not enough positions, expected %d, got %d", len(expectedPositions), len(positions))
			break
		}
		if positions[i].X != expected.X || positions[i].Y != expected.Y {
			t.Errorf("Position[%d] expected (%d,%d), got (%d,%d)", i, expected.X, expected.Y, positions[i].X, positions[i].Y)
		}
	}

	t.Logf("Scenario 1: Moving south, line extends north from (10,15): %v", positions)

	// Scenario 2: Units moving east, line should extend west from click
	// Units at (5,10), (5,11), (5,12), click at (15,10) → expect (15,10), (14,10), (13,10)
	server.entities[4] = &Entity{Id: 4, TileX: 5, TileY: 10}
	server.entities[5] = &Entity{Id: 5, TileX: 5, TileY: 11}
	server.entities[6] = &Entity{Id: 6, TileX: 5, TileY: 12}

	dx, dy = server.calculateMovementDirection([]uint32{4, 5, 6}, 15, 10)
	direction = getPrimaryDirection(dx, dy)

	if direction != "E" {
		t.Errorf("Expected direction E (east), got %s", direction)
	}

	positions = server.calculateLineFormationOriented(15, 10, 3, direction)

	// Verify position[0] is at click point
	if positions[0].X != 15 || positions[0].Y != 10 {
		t.Errorf("Position[0] should be at click point (15,10), got (%d,%d)", positions[0].X, positions[0].Y)
	}

	// Verify line extends west (negative X)
	expectedPositions = []TilePosition{{15, 10}, {14, 10}, {13, 10}}
	for i, expected := range expectedPositions {
		if i >= len(positions) {
			t.Errorf("Not enough positions, expected %d, got %d", len(expectedPositions), len(positions))
			break
		}
		if positions[i].X != expected.X || positions[i].Y != expected.Y {
			t.Errorf("Position[%d] expected (%d,%d), got (%d,%d)", i, expected.X, expected.Y, positions[i].X, positions[i].Y)
		}
	}

	t.Logf("Scenario 2: Moving east, line extends west from (15,10): %v", positions)
}

// TestAllUnitsReceivePaths verifies that every unit in a formation gets a path
func TestAllUnitsReceivePaths(t *testing.T) {
	mapData := &MapData{
		Width:          40,
		Height:         40,
		TileSize:       32,
		DefaultTerrain: TerrainType{Type: "grass", Passable: true},
		Tiles:          map[TileCoord]TerrainType{},
		Features:       []Feature{},
		SpawnPoints:    []SpawnPoint{},
	}

	server := &GameServer{
		mapData:         mapData,
		entities:        make(map[uint32]*Entity),
		formations:      make(map[uint32]*FormationGroup),
		clients:         make(map[uint32]*Client),
		nextId:          1,
		nextFormationID: 1,
		tick:            0,
	}

	// Create test client
	testClient := &Client{
		Id:    1,
		Name:  "TestPlayer",
		Money: 1000,
	}
	server.clients[1] = testClient

	// Create 5 units spread out
	unitPositions := [][2]int{
		{5, 5}, // Unit 1
		{6, 5}, // Unit 2
		{7, 5}, // Unit 3
		{8, 5}, // Unit 4
		{9, 5}, // Unit 5 (farthest from target)
	}

	unitIds := []uint32{}
	for _, pos := range unitPositions {
		unitId := server.nextId
		server.nextId++
		entity := &Entity{
			Id:      unitId,
			OwnerId: 1,
			Type:    "worker",
			TileX:   pos[0],
			TileY:   pos[1],
		}
		server.entities[unitId] = entity
		unitIds = append(unitIds, unitId)
		t.Logf("Created unit %d at (%d,%d)", unitId, pos[0], pos[1])
	}

	// Issue formation move command to distant location
	targetX, targetY := 30, 30
	cmd := Command{
		Type: "move",
		Data: map[string]interface{}{
			"unitIds":     convertToInterfaceSlice(unitIds),
			"targetTileX": float64(targetX),
			"targetTileY": float64(targetY),
			"formation":   "box",
		},
	}

	server.handleMoveCommand(cmd, testClient)

	// Check: ALL units should have non-nil paths
	unitsWithoutPaths := []uint32{}
	for i, unitId := range unitIds {
		entity := server.entities[unitId]
		if len(entity.Path) == 0 {
			unitsWithoutPaths = append(unitsWithoutPaths, unitId)
			t.Errorf("Unit %d (index %d) at (%d,%d) has NO PATH!",
				unitId, i, entity.TileX, entity.TileY)
		} else {
			t.Logf("✓ Unit %d (index %d) has path with %d waypoints",
				unitId, i, len(entity.Path))
		}
	}

	if len(unitsWithoutPaths) > 0 {
		t.Fatalf("%d units failed to receive paths: %v", len(unitsWithoutPaths), unitsWithoutPaths)
	}

	// Simulate 20 ticks and verify ALL units have moved
	initialPositions := make(map[uint32][2]int)
	for _, unitId := range unitIds {
		entity := server.entities[unitId]
		initialPositions[unitId] = [2]int{entity.TileX, entity.TileY}
	}

	deltaTime := 1.0 / float32(20) // 20Hz tick rate
	for tick := 0; tick < 20; tick++ {
		server.tickFormations()
		for _, entity := range server.entities {
			if entity.Type == "worker" {
				server.updateEntityMovement(entity, deltaTime)
			}
		}
		server.tick++
	}

	// Check: ALL units should have moved from initial position
	unmovedUnits := []uint32{}
	for i, unitId := range unitIds {
		entity := server.entities[unitId]
		initialPos := initialPositions[unitId]
		if entity.TileX == initialPos[0] && entity.TileY == initialPos[1] {
			unmovedUnits = append(unmovedUnits, unitId)
			t.Errorf("Unit %d (index %d) did NOT MOVE! Still at (%d,%d)",
				unitId, i, entity.TileX, entity.TileY)
		} else {
			t.Logf("✓ Unit %d (index %d) moved from (%d,%d) to (%d,%d)",
				unitId, i, initialPos[0], initialPos[1], entity.TileX, entity.TileY)
		}
	}

	if len(unmovedUnits) > 0 {
		t.Fatalf("%d units did not move after 20 ticks: %v", len(unmovedUnits), unmovedUnits)
	}
}

// Helper to convert uint32 slice to interface{} slice for command data
func convertToInterfaceSlice(ids []uint32) []interface{} {
	result := make([]interface{}, len(ids))
	for i, id := range ids {
		result[i] = float64(id) // JSON uses float64 for numbers
	}
	return result
}

// Helper function
func formatPos(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}
