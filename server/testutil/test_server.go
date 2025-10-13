package testutil

import (
	"sync"
)

// Import parent package types - will need to be adjusted based on actual structure
// For now, we'll reference the main package

// TestServer wraps GameServer for in-process testing
type TestServer struct {
	server      any // *GameServer from main package
	tick        uint64
	nextId      uint32
	clients     map[uint32]*TestClient
	mu          sync.Mutex
}

// NewTestServer creates a test server with specified map
func NewTestServer(mapFile string) *TestServer {
	// This will need to create a GameServer instance
	// For now, return placeholder
	ts := &TestServer{
		tick:    0,
		nextId:  100, // Start test IDs at 100
		clients: make(map[uint32]*TestClient),
	}
	return ts
}

// AddTestUnit adds a unit directly to the game state at specified position
// Returns the entity ID
func (ts *TestServer) AddTestUnit(x, y int, ownerId uint32) uint32 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	entityId := ts.nextId
	ts.nextId++

	// TODO: Actually add entity to game server
	// For now, just return ID

	return entityId
}

// AddTestClient creates a test client that can send commands
func (ts *TestServer) AddTestClient(name string) *TestClient {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	clientId := ts.nextId
	ts.nextId++

	client := &TestClient{
		id:     clientId,
		name:   name,
		server: ts,
	}

	ts.clients[clientId] = client

	return client
}

// StepTicks advances the game simulation by N ticks
func (ts *TestServer) StepTicks(n int) {
	for i := 0; i < n; i++ {
		ts.stepOneTick()
	}
}

// StepUntilStopped advances simulation until unit stops moving (or timeout)
func (ts *TestServer) StepUntilStopped(unitId uint32, maxTicks int) bool {
	for i := 0; i < maxTicks; i++ {
		ts.stepOneTick()

		// Check if unit has stopped
		// TODO: Implement actual check
		// For now, just step all ticks
	}
	return true
}

// stepOneTick advances simulation by one tick
func (ts *TestServer) stepOneTick() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.tick++

	// TODO: Call GameServer.gameTick()
	// This requires refactoring GameServer to be testable
}

// GetEntity returns entity by ID for inspection
func (ts *TestServer) GetEntity(id uint32) any {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// TODO: Return actual entity from game server
	return nil
}

// GetEntityAt returns entity at specified tile position
func (ts *TestServer) GetEntityAt(x, y int) any {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// TODO: Iterate entities and find one at (x,y)
	return nil
}

// GetAllEntities returns all entities for inspection
func (ts *TestServer) GetAllEntities() []any {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// TODO: Return all entities
	return nil
}

// SendMoveCommand sends a move command from a client
func (ts *TestServer) SendMoveCommand(clientId uint32, unitIds []uint32, targetX, targetY int, formation string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// TODO: Call handleMoveCommand with command data
}

// SendBuildCommand sends a build command from a client
func (ts *TestServer) SendBuildCommand(clientId uint32, buildingType string, x, y int) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// TODO: Call handleBuildCommand
}

// TestClient represents a simulated client for testing
type TestClient struct {
	id     uint32
	name   string
	server *TestServer
}

// MoveUnits sends a move command for units
func (tc *TestClient) MoveUnits(unitIds []uint32, x, y int, formation string) {
	tc.server.SendMoveCommand(tc.id, unitIds, x, y, formation)
}

// Build sends a build command
func (tc *TestClient) Build(buildingType string, x, y int) {
	tc.server.SendBuildCommand(tc.id, buildingType, x, y)
}

// GetID returns the client ID
func (tc *TestClient) GetID() uint32 {
	return tc.id
}

// Placeholder types (will be replaced with imports from main package)
type Entity struct {
	Id      uint32
	TileX   int
	TileY   int
	Path    []TilePosition
	PathIndex int
	MoveProgress float32
}

type TilePosition struct {
	X, Y int
}

type GameServer struct {
	// Placeholder
}

type Command struct {
	Type string
	Data any
}
