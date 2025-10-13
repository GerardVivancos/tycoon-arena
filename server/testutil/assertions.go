package testutil

import (
	"fmt"
	"math"
	"testing"
)

// AssertUnitAt verifies a unit is at the specified tile position
func AssertUnitAt(t *testing.T, entity *Entity, x, y int) {
	t.Helper()
	if entity.TileX != x || entity.TileY != y {
		t.Errorf("Expected unit at (%d,%d), got (%d,%d)", x, y, entity.TileX, entity.TileY)
	}
}

// AssertUnitMoving verifies a unit is currently moving (has a path)
func AssertUnitMoving(t *testing.T, entity *Entity) {
	t.Helper()
	if len(entity.Path) == 0 {
		t.Errorf("Expected unit to be moving (have a path), but path is empty")
	}
}

// AssertUnitStopped verifies a unit has stopped (no path)
func AssertUnitStopped(t *testing.T, entity *Entity) {
	t.Helper()
	if len(entity.Path) > 0 {
		t.Errorf("Expected unit to be stopped (no path), but has path with %d waypoints", len(entity.Path))
	}
	if entity.MoveProgress > 0.01 {
		t.Errorf("Expected unit to be stopped (progress=0), but progress=%f", entity.MoveProgress)
	}
}

// AssertPathLength verifies the path has expected number of waypoints
func AssertPathLength(t *testing.T, entity *Entity, expectedLen int) {
	t.Helper()
	actualLen := len(entity.Path)
	if actualLen != expectedLen {
		t.Errorf("Expected path length %d, got %d", expectedLen, actualLen)
	}
}

// AssertPathAvoids verifies path doesn't go through specified tile
func AssertPathAvoids(t *testing.T, entity *Entity, x, y int) {
	t.Helper()
	for i, waypoint := range entity.Path {
		if waypoint.X == x && waypoint.Y == y {
			t.Errorf("Path waypoint %d goes through forbidden tile (%d,%d)", i, x, y)
		}
	}
}

// AssertPathReaches verifies path ends at specified destination
func AssertPathReaches(t *testing.T, entity *Entity, x, y int) {
	t.Helper()
	if len(entity.Path) == 0 {
		t.Errorf("Path is empty, cannot reach (%d,%d)", x, y)
		return
	}
	lastWaypoint := entity.Path[len(entity.Path)-1]
	if lastWaypoint.X != x || lastWaypoint.Y != y {
		t.Errorf("Path reaches (%d,%d), expected (%d,%d)", lastWaypoint.X, lastWaypoint.Y, x, y)
	}
}

// AssertNoUnitsStacked verifies no two units occupy the same tile
func AssertNoUnitsStacked(t *testing.T, entities []*Entity) {
	t.Helper()
	positions := make(map[string][]uint32) // "x,y" -> list of entity IDs

	for _, entity := range entities {
		key := formatPos(entity.TileX, entity.TileY)
		positions[key] = append(positions[key], entity.Id)
	}

	for pos, ids := range positions {
		if len(ids) > 1 {
			t.Errorf("Multiple units stacked at %s: %v", pos, ids)
		}
	}
}

// AssertFormationShape verifies units are in roughly expected formation
func AssertFormationShape(t *testing.T, units []*Entity, formation string) {
	t.Helper()

	if len(units) == 0 {
		t.Errorf("Cannot check formation of 0 units")
		return
	}

	switch formation {
	case "box":
		AssertBoxFormation(t, units)
	case "line":
		AssertLineFormation(t, units)
	case "spread":
		AssertSpreadFormation(t, units)
	default:
		t.Errorf("Unknown formation type: %s", formation)
	}
}

// AssertBoxFormation verifies units are in a roughly square/box arrangement
func AssertBoxFormation(t *testing.T, units []*Entity) {
	t.Helper()

	// Calculate bounding box
	minX, maxX := units[0].TileX, units[0].TileX
	minY, maxY := units[0].TileY, units[0].TileY

	for _, unit := range units {
		if unit.TileX < minX {
			minX = unit.TileX
		}
		if unit.TileX > maxX {
			maxX = unit.TileX
		}
		if unit.TileY < minY {
			minY = unit.TileY
		}
		if unit.TileY > maxY {
			maxY = unit.TileY
		}
	}

	width := maxX - minX + 1
	height := maxY - minY + 1

	// Box formation should be roughly square (width ≈ height)
	aspectRatio := float64(width) / float64(height)
	if aspectRatio < 0.5 || aspectRatio > 2.0 {
		t.Errorf("Box formation aspect ratio %f is not square-like (width=%d, height=%d)", aspectRatio, width, height)
	}
}

// AssertLineFormation verifies units are in a roughly horizontal line
func AssertLineFormation(t *testing.T, units []*Entity) {
	t.Helper()

	if len(units) < 2 {
		return // Can't form a line with < 2 units
	}

	// All units should have similar Y coordinates
	firstY := units[0].TileY
	tolerance := 2 // Allow 2 tile deviation

	for i, unit := range units {
		if abs(unit.TileY-firstY) > tolerance {
			t.Errorf("Line formation: unit %d at Y=%d deviates from first unit Y=%d by more than %d", i, unit.TileY, firstY, tolerance)
		}
	}
}

// AssertSpreadFormation verifies units are reasonably spread out
func AssertSpreadFormation(t *testing.T, units []*Entity) {
	t.Helper()

	// Calculate average distance between units
	if len(units) < 2 {
		return
	}

	var totalDist float64
	count := 0

	for i, unit1 := range units {
		for j, unit2 := range units {
			if i < j {
				dist := distance(unit1.TileX, unit1.TileY, unit2.TileX, unit2.TileY)
				totalDist += dist
				count++
			}
		}
	}

	avgDist := totalDist / float64(count)

	// Spread formation should have average distance > 1 tile
	if avgDist < 1.5 {
		t.Errorf("Spread formation average distance %.2f is too small (units too clustered)", avgDist)
	}
}

// AssertWithinDistance verifies unit is within N tiles of target
func AssertWithinDistance(t *testing.T, entity *Entity, targetX, targetY, maxDist int) {
	t.Helper()
	dist := distance(entity.TileX, entity.TileY, targetX, targetY)
	if dist > float64(maxDist) {
		t.Errorf("Unit at (%d,%d) is %.1f tiles from (%d,%d), expected within %d tiles",
			entity.TileX, entity.TileY, dist, targetX, targetY, maxDist)
	}
}

// Helper functions

func formatPos(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func distance(x1, y1, x2, y2 int) float64 {
	dx := float64(x2 - x1)
	dy := float64(y2 - y1)
	return math.Sqrt(dx*dx + dy*dy)
}
