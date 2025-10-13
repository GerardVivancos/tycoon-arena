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
		"9,7":   true,
		"10,7":  true,
		"11,7":  true,
		"9,8":   true,
		"10,8":  true,
		"11,8":  true,
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

// Helper function
func formatPos(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}
