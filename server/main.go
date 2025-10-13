package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"sort"
	"sync"
	"time"
)

const (
	ServerPort        = ":8080"
	TickRate          = 20 // 20 Hz
	MaxClients        = 6
	TileSize          = 32                          // World units per tile
	ArenaTilesWidth   = 25                          // 800 / 32
	ArenaTilesHeight  = 18                          // 576 / 32 (adjusted for clean division)
	ArenaWidth        = ArenaTilesWidth * TileSize  // 800
	ArenaHeight       = ArenaTilesHeight * TileSize // 576
	MovementSpeed     = 4.0                         // tiles per second
	ClientTimeout     = 10 * time.Second            // Timeout if no ping/input
	HeartbeatInterval = 2 * time.Second             // How often clients should ping

	// Game economy
	StartingMoney = 100
	BuildingCost  = 50

	// Resource generation (money per second per building)
	GeneratorIncome = 10.0
)

type MessageType string

const (
	MsgHello    MessageType = "hello"
	MsgWelcome  MessageType = "welcome"
	MsgInput    MessageType = "input"
	MsgSnapshot MessageType = "snapshot"
	MsgPing     MessageType = "ping"
	MsgPong     MessageType = "pong"
)

type Message struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data"`
}

type HelloMessage struct {
	ClientVersion string `json:"clientVersion"`
	PlayerName    string `json:"playerName"`
}

type WelcomeMessage struct {
	ClientId          uint32      `json:"clientId"`
	TickRate          int         `json:"tickRate"`
	HeartbeatInterval int         `json:"heartbeatInterval"` // milliseconds
	InputRedundancy   int         `json:"inputRedundancy"`   // How many commands to send per input
	TileSize          int         `json:"tileSize"`          // World units per tile
	ArenaTilesWidth   int         `json:"arenaTilesWidth"`
	ArenaTilesHeight  int         `json:"arenaTilesHeight"`
	TerrainData       TerrainData `json:"terrainData"` // Terrain information for rendering
}

type TerrainData struct {
	DefaultType string        `json:"defaultType"` // Default terrain type (e.g. "grass")
	Tiles       []TerrainTile `json:"tiles"`       // Non-default terrain tiles
}

type TerrainTile struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Type   string  `json:"type"`
	Height float32 `json:"height"`
}

type InputMessage struct {
	ClientId uint32         `json:"clientId"`
	Commands []CommandFrame `json:"commands"`
}

type CommandFrame struct {
	Sequence uint32    `json:"sequence"`
	Tick     uint64    `json:"tick"`
	Commands []Command `json:"commands"`
}

type Command struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type MoveCommand struct {
	UnitIds     []uint32 `json:"unitIds"` // Which units to move
	TargetTileX int      `json:"targetTileX"`
	TargetTileY int      `json:"targetTileY"`
	Formation   string   `json:"formation"` // Formation type: "box", "line", "staggered", "spread"
}

type BuildCommand struct {
	BuildingType string `json:"buildingType"`
	TileX        int    `json:"tileX"`
	TileY        int    `json:"tileY"`
}

type AttackCommand struct {
	TargetId uint32 `json:"targetId"`
}

type SnapshotMessage struct {
	Tick         uint64            `json:"tick"`
	BaselineTick uint64            `json:"baselineTick"` // For delta compression (0 = full snapshot)
	Entities     []Entity          `json:"entities"`
	Players      map[string]Player `json:"players"`
}

type Player struct {
	Id    uint32  `json:"id"`
	Name  string  `json:"name"`
	Money float32 `json:"money"`
}

type Entity struct {
	Id              uint32  `json:"id"`
	OwnerId         uint32  `json:"ownerId"`
	Type            string  `json:"type"`
	TileX           int     `json:"tileX"`
	TileY           int     `json:"tileY"`
	TargetTileX     int     `json:"targetTileX"`
	TargetTileY     int     `json:"targetTileY"`
	MoveProgress    float32 `json:"moveProgress"` // 0.0 to 1.0
	Health          int32   `json:"health"`
	MaxHealth       int32   `json:"maxHealth"`
	FootprintWidth  int     `json:"footprintWidth,omitempty"`  // In tiles (0 for units)
	FootprintHeight int     `json:"footprintHeight,omitempty"` // In tiles (0 for units)

	// Pathfinding
	Path        []TilePosition `json:"-"` // Full path to goal (not sent to client)
	PathIndex   int            `json:"-"` // Current waypoint index
	BlockedTime float32        `json:"-"` // Time spent blocked (for rerouting)
}

type Client struct {
	Id               uint32
	Name             string
	Addr             *net.UDPAddr
	LastSeen         time.Time
	OwnedUnits       []uint32 // Entity IDs of units owned by this player
	Money            float32
	LastProcessedSeq uint32
	LastAckTick      uint64 // For delta compression (not implemented)
}

// Map system types
type TileCoord struct {
	X int
	Y int
}

type TerrainType struct {
	Type     string  `json:"type"`
	Passable bool    `json:"passable"`
	Height   float32 `json:"height"`
	Visual   string  `json:"visual"`
}

type Feature struct {
	Type         string  `json:"type"`
	X            int     `json:"x"`
	Y            int     `json:"y"`
	Width        int     `json:"width"`
	Height       int     `json:"height"`
	Passable     bool    `json:"passable"`
	VisualHeight float32 `json:"visualHeight"`
}

type SpawnPoint struct {
	Team   int `json:"team"`
	X      int `json:"x"`
	Y      int `json:"y"`
	Radius int `json:"radius"`
}

type MapData struct {
	Width          int
	Height         int
	TileSize       int
	DefaultTerrain TerrainType
	Tiles          map[TileCoord]TerrainType // Sparse map for non-default tiles
	Features       []Feature
	SpawnPoints    []SpawnPoint
}

// JSON format for map files (matches our JSON structure)
type MapFileFormat struct {
	Version  string `json:"version"`
	Name     string `json:"name"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	TileSize int    `json:"tileSize"`
	Terrain  struct {
		Default TerrainType `json:"default"`
		Tiles   []struct {
			X        int     `json:"x"`
			Y        int     `json:"y"`
			Type     string  `json:"type"`
			Passable bool    `json:"passable"`
			Height   float32 `json:"height"`
		} `json:"tiles"`
	} `json:"terrain"`
	Features    []Feature    `json:"features"`
	SpawnPoints []SpawnPoint `json:"spawnPoints"`
	Metadata    struct {
		Author      string `json:"author"`
		Created     string `json:"created"`
		Description string `json:"description"`
	} `json:"metadata"`
}

type QueuedInput struct {
	ClientId uint32
	Sequence uint32
	Tick     uint64
	Commands []Command
}

type GameServer struct {
	conn       *net.UDPConn
	clients    map[uint32]*Client
	entities   map[uint32]*Entity
	tick       uint64
	nextId     uint32
	mu         sync.RWMutex
	inputQueue []QueuedInput
	queueMu    sync.Mutex
	mapData    *MapData // Map configuration
}

func NewGameServer() *GameServer {
	return &GameServer{
		clients:    make(map[uint32]*Client),
		entities:   make(map[uint32]*Entity),
		tick:       0,
		nextId:     1,
		inputQueue: make([]QueuedInput, 0),
	}
}

// LoadMap loads a map from a JSON file and returns MapData
func LoadMap(filepath string) (*MapData, error) {
	// Read the file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read map file: %w", err)
	}

	// Parse JSON
	var mapFile MapFileFormat
	if err := json.Unmarshal(data, &mapFile); err != nil {
		return nil, fmt.Errorf("failed to parse map JSON: %w", err)
	}

	// Validate dimensions
	if mapFile.Width <= 0 || mapFile.Height <= 0 {
		return nil, fmt.Errorf("invalid map dimensions: %dx%d", mapFile.Width, mapFile.Height)
	}

	// Build MapData
	mapData := &MapData{
		Width:          mapFile.Width,
		Height:         mapFile.Height,
		TileSize:       mapFile.TileSize,
		DefaultTerrain: mapFile.Terrain.Default,
		Tiles:          make(map[TileCoord]TerrainType),
		Features:       mapFile.Features,
		SpawnPoints:    mapFile.SpawnPoints,
	}

	// Build sparse tile map (only store non-default tiles)
	for _, tile := range mapFile.Terrain.Tiles {
		coord := TileCoord{X: tile.X, Y: tile.Y}
		mapData.Tiles[coord] = TerrainType{
			Type:     tile.Type,
			Passable: tile.Passable,
			Height:   tile.Height,
			Visual:   tile.Type, // Use type as visual if not specified
		}
	}

	log.Printf("Loaded map '%s': %dx%d tiles, %d terrain tiles, %d features, %d spawn points",
		mapFile.Name, mapData.Width, mapData.Height, len(mapData.Tiles), len(mapData.Features), len(mapData.SpawnPoints))

	return mapData, nil
}

func (s *GameServer) Start() error {
	addr, err := net.ResolveUDPAddr("udp", ServerPort)
	if err != nil {
		return err
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}

	log.Printf("Game server listening on %s", ServerPort)

	// Start the game tick loop
	go s.tickLoop()

	// Handle incoming messages
	return s.handleMessages()
}

func (s *GameServer) tickLoop() {
	ticker := time.NewTicker(time.Duration(1000/TickRate) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		s.gameTick()
	}
}

func (s *GameServer) gameTick() {
	// Get and sort input queue by tick (process in time order)
	s.queueMu.Lock()
	inputs := s.inputQueue
	s.inputQueue = make([]QueuedInput, 0) // Clear queue
	s.queueMu.Unlock()

	// Sort by tick (earliest first) for fair processing
	sort.Slice(inputs, func(i, j int) bool {
		return inputs[i].Tick < inputs[j].Tick
	})

	// Now lock for game state modification (single-threaded processing)
	s.mu.Lock()
	s.tick++

	// Clean up disconnected clients (heartbeat timeout)
	now := time.Now()
	for id, client := range s.clients {
		if now.Sub(client.LastSeen) > ClientTimeout {
			log.Printf("Client %d (%s) timed out (no heartbeat/input for %v)", id, client.Name, ClientTimeout)
			// Delete all owned units
			for _, unitId := range client.OwnedUnits {
				delete(s.entities, unitId)
			}
			delete(s.clients, id)
		}
	}

	// Process all queued inputs in tick order
	for _, input := range inputs {
		client, exists := s.clients[input.ClientId]
		if !exists {
			continue
		}

		// Skip if already processed (redundancy deduplication)
		if input.Sequence <= client.LastProcessedSeq {
			continue
		}

		// Mark as processed
		client.LastProcessedSeq = input.Sequence

		// Process commands
		for _, cmd := range input.Commands {
			s.processCommand(cmd, client)
		}
	}

	// Update entity movement
	deltaTime := 1.0 / float32(TickRate)
	for _, entity := range s.entities {
		// Update movement for all unit types
		if entity.Type == "worker" {
			s.updateEntityMovement(entity, deltaTime)
		}
	}

	// Generate resources from buildings
	for _, entity := range s.entities {
		if entity.Type == "generator" {
			if client, ok := s.clients[entity.OwnerId]; ok {
				client.Money += GeneratorIncome * deltaTime
			}
		}
	}

	// Create snapshot
	entities := make([]Entity, 0, len(s.entities))
	for _, entity := range s.entities {
		entities = append(entities, *entity)
	}

	// Create player data
	players := make(map[string]Player)
	for id, client := range s.clients {
		players[fmt.Sprintf("%d", id)] = Player{
			Id:    id,
			Name:  client.Name,
			Money: client.Money,
		}
	}

	snapshot := SnapshotMessage{
		Tick:         s.tick,
		BaselineTick: 0, // TODO: Delta compression - always full snapshot for now
		Entities:     entities,
		Players:      players,
	}
	s.mu.Unlock()

	// Send snapshot to all clients (without holding lock)
	s.broadcastMessage(Message{
		Type: MsgSnapshot,
		Data: s.marshalData(snapshot),
	})
}

func (s *GameServer) handleMessages() error {
	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP message: %v", err)
			continue
		}

		var msg Message
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		s.handleMessage(msg, clientAddr)
	}
}

func (s *GameServer) handleMessage(msg Message, clientAddr *net.UDPAddr) {
	switch msg.Type {
	case MsgHello:
		var hello HelloMessage
		if err := json.Unmarshal(msg.Data, &hello); err != nil {
			log.Printf("Error unmarshaling hello message: %v", err)
			return
		}
		s.handleHello(hello, clientAddr)

	case MsgInput:
		var input InputMessage
		if err := json.Unmarshal(msg.Data, &input); err != nil {
			log.Printf("Error unmarshaling input message: %v", err)
			return
		}
		s.handleInput(input, clientAddr)

	case MsgPing:
		s.handlePing(clientAddr)
	}
}

func (s *GameServer) handleHello(hello HelloMessage, clientAddr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.clients) >= MaxClients {
		log.Printf("Server full, rejecting client from %s", clientAddr.String())
		return
	}

	clientId := s.nextId
	s.nextId++

	// Spawn starting units for this player (5 workers)
	// Assign team based on client number (team 0 for first client, team 1 for second, etc.)
	teamId := len(s.clients)
	spawnBaseTileX, spawnBaseTileY := s.getSpawnPosition(teamId)

	ownedUnits := make([]uint32, 0, 5)
	for i := 0; i < 5; i++ {
		entityId := s.nextId
		s.nextId++

		// Spawn workers in horizontal line
		workerX := spawnBaseTileX + i
		workerY := spawnBaseTileY

		// Ensure spawn position is passable (fallback to base position if not)
		if !s.isTilePassable(workerX, workerY) {
			workerX = spawnBaseTileX
			workerY = spawnBaseTileY
		}

		worker := &Entity{
			Id:           entityId,
			OwnerId:      clientId,
			Type:         "worker",
			TileX:        workerX,
			TileY:        workerY,
			TargetTileX:  workerX,
			TargetTileY:  workerY,
			MoveProgress: 0.0,
			Health:       100,
			MaxHealth:    100,
		}

		s.entities[entityId] = worker
		ownedUnits = append(ownedUnits, entityId)
	}

	client := &Client{
		Id:         clientId,
		Name:       hello.PlayerName,
		Addr:       clientAddr,
		LastSeen:   time.Now(),
		OwnedUnits: ownedUnits,
		Money:      StartingMoney,
	}

	s.clients[clientId] = client

	log.Printf("Client %d (%s) connected from %s with %d workers", clientId, hello.PlayerName, clientAddr.String(), len(ownedUnits))

	// Build terrain data for client
	terrainTiles := make([]TerrainTile, 0, len(s.mapData.Tiles))
	for coord, terrain := range s.mapData.Tiles {
		terrainTiles = append(terrainTiles, TerrainTile{
			X:      coord.X,
			Y:      coord.Y,
			Type:   terrain.Type,
			Height: terrain.Height,
		})
	}

	// Send welcome message
	welcome := WelcomeMessage{
		ClientId:          clientId,
		TickRate:          TickRate,
		HeartbeatInterval: int(HeartbeatInterval.Milliseconds()),
		InputRedundancy:   3, // Client should send last 3 commands
		TileSize:          TileSize,
		ArenaTilesWidth:   s.mapData.Width,
		ArenaTilesHeight:  s.mapData.Height,
		TerrainData: TerrainData{
			DefaultType: s.mapData.DefaultTerrain.Type,
			Tiles:       terrainTiles,
		},
	}

	s.sendMessage(Message{
		Type: MsgWelcome,
		Data: s.marshalData(welcome),
	}, clientAddr)
}

func (s *GameServer) handlePing(clientAddr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find client by address
	var foundClient *Client
	for _, client := range s.clients {
		if client.Addr.String() == clientAddr.String() {
			foundClient = client
			break
		}
	}

	if foundClient != nil {
		// Update last seen time
		foundClient.LastSeen = time.Now()

		// Send pong response
		s.mu.Unlock() // Unlock before sending
		s.sendMessage(Message{
			Type: MsgPong,
			Data: json.RawMessage("{}"),
		}, clientAddr)
		s.mu.Lock() // Re-lock for defer
	}
}

func (s *GameServer) handleInput(input InputMessage, clientAddr *net.UDPAddr) {
	s.mu.RLock()
	client, exists := s.clients[input.ClientId]
	s.mu.RUnlock()

	if !exists {
		return
	}

	// Update last seen (quick lock)
	s.mu.Lock()
	client.LastSeen = time.Now()
	s.mu.Unlock()

	// Enqueue all command frames (with redundancy)
	s.queueMu.Lock()
	defer s.queueMu.Unlock()

	for _, cmdFrame := range input.Commands {
		// Skip already-processed commands (deduplication)
		if cmdFrame.Sequence <= client.LastProcessedSeq {
			continue
		}

		// Add to input queue
		s.inputQueue = append(s.inputQueue, QueuedInput{
			ClientId: input.ClientId,
			Sequence: cmdFrame.Sequence,
			Tick:     cmdFrame.Tick,
			Commands: cmdFrame.Commands,
		})
	}
}

func (s *GameServer) processCommand(cmd Command, client *Client) {
	switch cmd.Type {
	case "move":
		s.handleMoveCommand(cmd, client)
	case "build":
		s.handleBuildCommand(cmd, client)
	case "attack":
		s.handleAttackCommand(cmd, client)
	}
}

func (s *GameServer) updateEntityMovement(entity *Entity, deltaTime float32) {
	// Check if entity has a path to follow
	if len(entity.Path) == 0 {
		entity.MoveProgress = 0.0
		return
	}

	// Check if path is complete
	if entity.PathIndex >= len(entity.Path) {
		// Path complete, clear it
		entity.Path = nil
		entity.PathIndex = 0
		entity.MoveProgress = 0.0
		return
	}

	// Get next waypoint
	waypoint := entity.Path[entity.PathIndex]
	entity.TargetTileX = waypoint.X
	entity.TargetTileY = waypoint.Y

	// Dynamic collision avoidance: Check if next waypoint is currently occupied
	// If so, pause movement this tick (unit waits for other unit to pass)
	if entity.MoveProgress < 1.0 {
		// Check if waypoint is occupied by another unit's current position
		isBlocked := false
		for _, other := range s.entities {
			if other.Id == entity.Id {
				continue
			}
			if other.Type != "worker" && other.Type != "player" {
				continue
			}
			// Check if other unit is currently at this waypoint
			if other.TileX == waypoint.X && other.TileY == waypoint.Y {
				isBlocked = true
				break
			}
		}

		// If blocked, accumulate blocked time and consider rerouting
		if isBlocked {
			entity.BlockedTime += deltaTime

			// If blocked for more than 1 second, recalculate path to find alternate route
			const BlockedThreshold = 1.0 // seconds
			if entity.BlockedTime > BlockedThreshold && len(entity.Path) > 0 {
				// Get final destination
				finalGoal := entity.Path[len(entity.Path)-1]

				// Recalculate path from current position to goal
				newPath := s.findPath(entity.TileX, entity.TileY, finalGoal.X, finalGoal.Y, entity.Id)

				if newPath != nil && len(newPath) > 0 {
					// Found alternate route
					entity.Path = newPath
					entity.PathIndex = 0
					entity.MoveProgress = 0.0
					entity.BlockedTime = 0.0
					log.Printf("Unit %d rerouting around blockage", entity.Id)
				} else {
					// No alternate path found, reset blocked time and keep waiting
					entity.BlockedTime = 0.0
				}
			}

			return // Don't move this tick
		}

		// Not blocked, reset blocked time
		entity.BlockedTime = 0.0
	}

	// Calculate movement progress increment
	// MovementSpeed is tiles/second, so progress per tick = (tiles/sec) * deltaTime / 1 tile
	progressIncrement := MovementSpeed * deltaTime
	entity.MoveProgress += progressIncrement

	// Check if reached waypoint
	if entity.MoveProgress >= 1.0 {
		// Move to waypoint
		entity.TileX = waypoint.X
		entity.TileY = waypoint.Y
		entity.MoveProgress = 0.0

		// Advance to next waypoint
		entity.PathIndex++

		// Check if path complete
		if entity.PathIndex >= len(entity.Path) {
			entity.Path = nil
			entity.PathIndex = 0
		}
	}
}

type TilePosition struct {
	X, Y int
}

// Pathfinding structures for A* algorithm
type pathNode struct {
	x, y   int
	gCost  float32 // Cost from start
	hCost  float32 // Heuristic to goal
	fCost  float32 // gCost + hCost
	parent *pathNode
	index  int // Index in heap
}

// Priority queue for A* open set
type nodeHeap []*pathNode

func (h nodeHeap) Len() int { return len(h) }

func (h nodeHeap) Less(i, j int) bool {
	// Lower fCost has higher priority
	return h[i].fCost < h[j].fCost
}

func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *nodeHeap) Push(x any) {
	n := len(*h)
	node := x.(*pathNode)
	node.index = n
	*h = append(*h, node)
}

func (h *nodeHeap) Pop() any {
	old := *h
	n := len(old)
	node := old[n-1]
	old[n-1] = nil
	node.index = -1
	*h = old[0 : n-1]
	return node
}

// findNearestPassableTile searches in a spiral pattern for the nearest passable tile
// Returns the input position if already passable, or nearest alternative
func (s *GameServer) findNearestPassableTile(startX, startY, maxRadius int) TilePosition {
	// Check center first
	if s.isTilePassable(startX, startY) {
		return TilePosition{X: startX, Y: startY}
	}

	// Spiral outward looking for passable tile
	directions := []TilePosition{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} // Right, Down, Left, Up
	x, y := startX, startY
	steps := 1

	for radius := 1; radius <= maxRadius; radius++ {
		for _, dir := range directions {
			for step := 0; step < steps && radius <= maxRadius; step++ {
				x += dir.X
				y += dir.Y

				if s.isTilePassable(x, y) {
					return TilePosition{X: x, Y: y}
				}
			}

			// Increase steps after every 2 directions
			if dir.X == 0 {
				steps++
			}
		}
	}

	// Fallback: return original position (unit will stack, but at least won't crash)
	return TilePosition{X: startX, Y: startY}
}

// manhattanDistance calculates Manhattan distance heuristic for A*
func (s *GameServer) manhattanDistance(x1, y1, x2, y2 int) float32 {
	return float32(abs(x2-x1) + abs(y2-y1))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// reconstructPath builds path from goal node back to start by following parent pointers
func reconstructPath(node *pathNode) []TilePosition {
	path := []TilePosition{}
	for current := node; current != nil; current = current.parent {
		path = append(path, TilePosition{X: current.x, Y: current.y})
	}
	// Reverse path so it goes from start to goal
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

// findPath uses A* algorithm to find path from (startX, startY) to (goalX, goalY)
// Returns path as slice of tile positions, or nil if no path exists
func (s *GameServer) findPath(startX, startY, goalX, goalY int, unitId uint32) []TilePosition {
	// Early exit: already at goal
	if startX == goalX && startY == goalY {
		return []TilePosition{{X: startX, Y: startY}}
	}

	// Early exit: goal not passable
	if !s.isTileAvailableForUnit(goalX, goalY, unitId) {
		return nil
	}

	// Initialize open and closed sets
	openSet := &nodeHeap{}
	heap.Init(openSet)
	closedSet := make(map[int]bool) // Use single int key: y*width + x

	// Start node
	startNode := &pathNode{
		x:     startX,
		y:     startY,
		gCost: 0,
		hCost: s.manhattanDistance(startX, startY, goalX, goalY),
	}
	startNode.fCost = startNode.gCost + startNode.hCost
	heap.Push(openSet, startNode)

	// 4-directional movement
	directions := [][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} // N, E, S, W

	// A* main loop
	for openSet.Len() > 0 {
		// Pop node with lowest fCost
		current := heap.Pop(openSet).(*pathNode)

		// Goal reached!
		if current.x == goalX && current.y == goalY {
			return reconstructPath(current)
		}

		// Add to closed set
		closedKey := current.y*s.mapData.Width + current.x
		closedSet[closedKey] = true

		// Check all neighbors
		for _, dir := range directions {
			nx := current.x + dir[0]
			ny := current.y + dir[1]

			// Skip if out of bounds or impassable
			if !s.isTileAvailableForUnit(nx, ny, unitId) {
				continue
			}

			// Skip if already in closed set
			neighborKey := ny*s.mapData.Width + nx
			if closedSet[neighborKey] {
				continue
			}

			// Calculate costs
			tentativeGCost := current.gCost + 1.0 // Cost to move to adjacent tile

			// Check if neighbor already in open set
			var neighborNode *pathNode
			for i := 0; i < openSet.Len(); i++ {
				node := (*openSet)[i]
				if node.x == nx && node.y == ny {
					neighborNode = node
					break
				}
			}

			if neighborNode == nil {
				// New node, add to open set
				neighborNode = &pathNode{
					x:      nx,
					y:      ny,
					gCost:  tentativeGCost,
					hCost:  s.manhattanDistance(nx, ny, goalX, goalY),
					parent: current,
				}
				neighborNode.fCost = neighborNode.gCost + neighborNode.hCost
				heap.Push(openSet, neighborNode)
			} else if tentativeGCost < neighborNode.gCost {
				// Found better path to this node
				neighborNode.gCost = tentativeGCost
				neighborNode.fCost = neighborNode.gCost + neighborNode.hCost
				neighborNode.parent = current
				heap.Fix(openSet, neighborNode.index)
			}
		}
	}

	// No path found
	return nil
}

// calculateFormation returns tile positions for units in the specified formation
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

// calculateBoxFormationOriented creates a grid pattern with tip at target point
func (s *GameServer) calculateBoxFormationOriented(tipX, tipY, numUnits int, direction string) []TilePosition {
	positions := make([]TilePosition, 0, numUnits)

	// Calculate grid dimensions (roughly square)
	gridSize := int(math.Ceil(math.Sqrt(float64(numUnits))))

	// Position[0] (closest unit) should be at (tipX, tipY)
	// Grid extends backward from tip based on movement direction
	// Using reversed iteration or adjusted start position to ensure tip is position[0]

	var startX, startY int
	var colDir, rowDir int // Direction multipliers for grid expansion

	switch direction {
	case "E": // Moving east, tip on right, grid extends left/up-down
		startX = tipX
		startY = tipY - gridSize/2
		colDir = -1 // Columns go left (west)
		rowDir = 1  // Rows go down (south)
	case "W": // Moving west, tip on left, grid extends right/up-down
		startX = tipX
		startY = tipY - gridSize/2
		colDir = 1 // Columns go right (east)
		rowDir = 1 // Rows go down (south)
	case "N": // Moving north, tip on top, grid extends down/left-right
		startX = tipX - gridSize/2
		startY = tipY
		colDir = 1 // Columns go right (east)
		rowDir = 1 // Rows go down (south)
	case "S": // Moving south, tip on bottom, grid extends up/left-right
		startX = tipX - gridSize/2
		startY = tipY
		colDir = 1  // Columns go right (east)
		rowDir = -1 // Rows go up (north)
	case "NE": // Moving northeast, tip top-right, grid extends left/down
		startX = tipX
		startY = tipY
		colDir = -1 // Columns go left (west)
		rowDir = 1  // Rows go down (south)
	case "NW": // Moving northwest, tip top-left, grid extends right/down
		startX = tipX
		startY = tipY
		colDir = 1 // Columns go right (east)
		rowDir = 1 // Rows go down (south)
	case "SE": // Moving southeast, tip bottom-right, grid extends left/up
		startX = tipX
		startY = tipY
		colDir = -1 // Columns go left (west)
		rowDir = -1 // Rows go up (north)
	case "SW": // Moving southwest, tip bottom-left, grid extends right/up
		startX = tipX
		startY = tipY
		colDir = 1  // Columns go right (east)
		rowDir = -1 // Rows go up (north)
	default: // Fallback to centered
		startX = tipX - gridSize/2
		startY = tipY - gridSize/2
		colDir = 1
		rowDir = 1
	}

	for i := 0; i < numUnits; i++ {
		row := i / gridSize
		col := i % gridSize

		tileX := startX + (col * colDir)
		tileY := startY + (row * rowDir)

		// Check if tile is passable (includes bounds, terrain, and buildings)
		if !s.isTilePassable(tileX, tileY) {
			continue
		}

		positions = append(positions, TilePosition{X: tileX, Y: tileY})
	}

	// If we couldn't find enough positions, find nearest passable tiles
	spiralOffset := 0
	for len(positions) < numUnits {
		searchX := tipX + spiralOffset
		searchY := tipY + spiralOffset
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
			positions = append(positions, fallbackPos)
		}
	}

	return positions
}

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

// calculateLineFormationOriented creates a line parallel to movement direction
func (s *GameServer) calculateLineFormationOriented(tipX, tipY, numUnits int, direction string) []TilePosition {
	positions := make([]TilePosition, 0, numUnits)

	// Line extends backward from click point (opposite to movement direction)
	// Position[0] is at click point (tip), rest extend backward toward origin
	// For horizontal movement (E/W), create horizontal line extending opposite
	// For vertical movement (N/S), create vertical line extending opposite
	// For diagonal, create diagonal line extending opposite

	var dx, dy int // Direction to extend backward (opposite of movement)

	switch direction {
	case "E": // Moving east → line extends west (negative X)
		dx = -1
		dy = 0
	case "W": // Moving west → line extends east (positive X)
		dx = 1
		dy = 0
	case "N": // Moving north → line extends south (positive Y)
		dx = 0
		dy = 1
	case "S": // Moving south → line extends north (negative Y)
		dx = 0
		dy = -1
	case "NE": // Moving northeast → line extends southwest
		dx = -1
		dy = 1
	case "SW": // Moving southwest → line extends northeast
		dx = 1
		dy = -1
	case "NW": // Moving northwest → line extends southeast
		dx = 1
		dy = 1
	case "SE": // Moving southeast → line extends northwest
		dx = -1
		dy = -1
	default: // Fallback to extending west
		dx = -1
		dy = 0
	}

	// Start at click point (tip), extend backward
	for i := 0; i < numUnits; i++ {
		tileX := tipX + (dx * i)
		tileY := tipY + (dy * i)

		// Check if tile is passable
		if !s.isTilePassable(tileX, tileY) {
			continue
		}

		positions = append(positions, TilePosition{X: tileX, Y: tileY})
	}

	// If we couldn't find enough positions, find nearest passable tiles
	spiralOffset := 0
	for len(positions) < numUnits {
		searchX := tipX
		searchY := tipY + spiralOffset
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
			positions = append(positions, fallbackPos)
		}
	}

	return positions
}

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

// calculateUnitCentroid calculates the average position of selected units
func (s *GameServer) calculateUnitCentroid(unitIds []uint32) (float64, float64) {
	if len(unitIds) == 0 {
		return 0, 0
	}

	sumX, sumY := 0, 0
	for _, unitId := range unitIds {
		entity := s.entities[unitId]
		if entity == nil {
			continue
		}
		sumX += entity.TileX
		sumY += entity.TileY
	}

	return float64(sumX) / float64(len(unitIds)), float64(sumY) / float64(len(unitIds))
}

// calculateMovementDirection determines direction vector from units to target
func (s *GameServer) calculateMovementDirection(unitIds []uint32, targetX, targetY int) (dx, dy float64) {
	centroidX, centroidY := s.calculateUnitCentroid(unitIds)

	// Direction vector (normalized)
	dx = float64(targetX) - centroidX
	dy = float64(targetY) - centroidY
	length := math.Sqrt(dx*dx + dy*dy)

	if length > 0 {
		dx /= length
		dy /= length
	}

	return dx, dy
}

// getPrimaryDirection converts direction vector to 8-way cardinal/ordinal direction
func getPrimaryDirection(dx, dy float64) string {
	absDx := math.Abs(dx)
	absDy := math.Abs(dy)

	if absDx > absDy*2 {
		// Strongly horizontal
		if dx > 0 {
			return "E"
		}
		return "W"
	} else if absDy > absDx*2 {
		// Strongly vertical
		if dy > 0 {
			return "S"
		}
		return "N"
	} else {
		// Diagonal
		if dx > 0 && dy > 0 {
			return "SE"
		}
		if dx > 0 && dy < 0 {
			return "NE"
		}
		if dx < 0 && dy > 0 {
			return "SW"
		}
		return "NW"
	}
}

func (s *GameServer) handleMoveCommand(cmd Command, client *Client) {
	moveData, ok := cmd.Data.(map[string]interface{})
	if !ok {
		return
	}

	// Get unit IDs to move
	unitIdsInterface, ok := moveData["unitIds"].([]interface{})
	if !ok || len(unitIdsInterface) == 0 {
		return
	}

	targetTileX, okX := moveData["targetTileX"].(float64) // JSON numbers are float64
	targetTileY, okY := moveData["targetTileY"].(float64)
	if !okX || !okY {
		return
	}

	tileX := int(targetTileX)
	tileY := int(targetTileY)

	// Validate bounds
	if tileX < 0 || tileX >= s.mapData.Width || tileY < 0 || tileY >= s.mapData.Height {
		return
	}

	// Get formation type (default to "box")
	formation, _ := moveData["formation"].(string)
	if formation == "" {
		formation = "box"
	}

	// Collect valid unit IDs that belong to this player
	validUnitIds := make([]uint32, 0, len(unitIdsInterface))
	for _, unitIdInterface := range unitIdsInterface {
		unitIdFloat, ok := unitIdInterface.(float64)
		if !ok {
			continue
		}
		unitId := uint32(unitIdFloat)

		// Verify this unit exists and belongs to this player
		entity, exists := s.entities[unitId]
		if !exists || entity.OwnerId != client.Id {
			continue
		}

		// Only move units, not buildings
		if entity.Type == "worker" {
			validUnitIds = append(validUnitIds, unitId)
		}
	}

	if len(validUnitIds) == 0 {
		return
	}

	// Sort units by distance to target (closest first becomes tip)
	sort.Slice(validUnitIds, func(i, j int) bool {
		entity1 := s.entities[validUnitIds[i]]
		entity2 := s.entities[validUnitIds[j]]
		// Manhattan distance
		dist1 := abs(entity1.TileX-tileX) + abs(entity1.TileY-tileY)
		dist2 := abs(entity2.TileX-tileX) + abs(entity2.TileY-tileY)
		return dist1 < dist2
	})

	// Calculate movement direction for oriented formations
	dx, dy := s.calculateMovementDirection(validUnitIds, tileX, tileY)
	direction := getPrimaryDirection(dx, dy)

	// Calculate formation positions (oriented based on movement direction)
	var formationPositions []TilePosition
	switch formation {
	case "box":
		formationPositions = s.calculateBoxFormationOriented(tileX, tileY, len(validUnitIds), direction)
	case "line":
		formationPositions = s.calculateLineFormationOriented(tileX, tileY, len(validUnitIds), direction)
	case "spread":
		// Spread formation doesn't need orientation (radially symmetric)
		formationPositions = s.calculateSpiralFormation(tileX, tileY, len(validUnitIds))
	default:
		// Default to box formation
		formationPositions = s.calculateBoxFormationOriented(tileX, tileY, len(validUnitIds), direction)
	}

	// Assign each unit to its formation position using pathfinding
	for i, unitId := range validUnitIds {
		entity := s.entities[unitId]

		var targetX, targetY int
		if i < len(formationPositions) {
			targetX = formationPositions[i].X
			targetY = formationPositions[i].Y
		} else {
			// Fallback to center if not enough formation positions
			targetX = tileX
			targetY = tileY
		}

		// Calculate path to formation position
		path := s.findPath(entity.TileX, entity.TileY, targetX, targetY, entity.Id)

		if path != nil && len(path) > 0 {
			// Store path and start following it
			entity.Path = path
			entity.PathIndex = 0
			entity.MoveProgress = 0.0

			// Set target to first waypoint (will be updated as path is followed)
			if len(path) > 0 {
				entity.TargetTileX = path[0].X
				entity.TargetTileY = path[0].Y
			}
		} else {
			// No path found, unit stays at current position
			log.Printf("No path found for unit %d from (%d,%d) to (%d,%d)",
				entity.Id, entity.TileX, entity.TileY, targetX, targetY)
		}
	}
}

func (s *GameServer) isTileOccupiedByBuilding(tileX, tileY int) bool {
	for _, entity := range s.entities {
		if entity.Type == "generator" {
			// Check if (tileX, tileY) is within building's footprint
			if tileX >= entity.TileX && tileX < entity.TileX+entity.FootprintWidth &&
				tileY >= entity.TileY && tileY < entity.TileY+entity.FootprintHeight {
				return true
			}
		}
	}
	return false
}

// getSpawnPosition returns a spawn position for a given team
// It attempts to find a passable tile near the team's spawn point
func (s *GameServer) getSpawnPosition(teamId int) (int, int) {
	// Find spawn point for this team
	for _, spawn := range s.mapData.SpawnPoints {
		if spawn.Team == teamId {
			// Try to find a passable tile near the spawn point
			for attempt := 0; attempt < 100; attempt++ {
				offsetX := 0
				offsetY := 0
				if spawn.Radius > 0 {
					// Random offset within radius (simplified - not true circle)
					offsetX = (attempt % (spawn.Radius*2 + 1)) - spawn.Radius
					offsetY = (attempt / (spawn.Radius*2 + 1)) - spawn.Radius
				}

				x := spawn.X + offsetX
				y := spawn.Y + offsetY

				if s.isTilePassable(x, y) {
					return x, y
				}
			}
		}
	}

	// Fallback: use team-based default positions
	if teamId == 0 {
		return 5, s.mapData.Height / 2
	} else {
		return s.mapData.Width - 10, s.mapData.Height / 2
	}
}

// isTilePassable checks if a tile can be moved through or built on
func (s *GameServer) isTilePassable(tileX, tileY int) bool {
	// 1. Check bounds
	if tileX < 0 || tileX >= s.mapData.Width || tileY < 0 || tileY >= s.mapData.Height {
		return false
	}

	// 2. Check terrain (sparse map - if tile exists and is not passable)
	coord := TileCoord{X: tileX, Y: tileY}
	if terrain, exists := s.mapData.Tiles[coord]; exists {
		if !terrain.Passable {
			return false
		}
	}
	// If tile doesn't exist in sparse map, use default terrain passability
	if !s.mapData.DefaultTerrain.Passable {
		return false
	}

	// 3. Check multi-tile features
	for _, feature := range s.mapData.Features {
		if tileX >= feature.X && tileX < feature.X+feature.Width &&
			tileY >= feature.Y && tileY < feature.Y+feature.Height {
			if !feature.Passable {
				return false
			}
		}
	}

	// 4. Check buildings (existing logic)
	if s.isTileOccupiedByBuilding(tileX, tileY) {
		return false
	}

	return true
}

// isTileOccupiedByUnit checks if another unit is at this tile or will stop there
func (s *GameServer) isTileOccupiedByUnit(tileX, tileY int, excludeId uint32) bool {
	for _, entity := range s.entities {
		// Skip non-units (buildings)
		if entity.Type != "worker" && entity.Type != "player" {
			continue
		}

		// Skip the unit we're checking for
		if entity.Id == excludeId {
			continue
		}

		// Check current position (where unit is standing)
		if entity.TileX == tileX && entity.TileY == tileY {
			return true
		}

		// Check final destination (where unit will stop)
		// Allow paths to cross, but prevent units from having same destination
		if len(entity.Path) > 0 {
			finalDest := entity.Path[len(entity.Path)-1]
			if finalDest.X == tileX && finalDest.Y == tileY {
				return true
			}
		}
	}
	return false
}

// isTileAvailableForUnit checks if tile is passable and not occupied by other units
func (s *GameServer) isTileAvailableForUnit(tileX, tileY int, unitId uint32) bool {
	// Check terrain + buildings
	if !s.isTilePassable(tileX, tileY) {
		return false
	}

	// Check other units
	if s.isTileOccupiedByUnit(tileX, tileY, unitId) {
		return false
	}

	return true
}

func (s *GameServer) handleBuildCommand(cmd Command, client *Client) {
	buildData, ok := cmd.Data.(map[string]interface{})
	if !ok {
		return
	}

	buildingType, _ := buildData["buildingType"].(string)
	tileXFloat, _ := buildData["tileX"].(float64)
	tileYFloat, _ := buildData["tileY"].(float64)
	tileX := int(tileXFloat)
	tileY := int(tileYFloat)

	// Validate building type and get footprint
	var footprintWidth, footprintHeight int
	switch buildingType {
	case "generator":
		footprintWidth = 2
		footprintHeight = 2
	default:
		return // Unknown building type
	}

	// Check if player has enough money
	if client.Money < BuildingCost {
		return
	}

	// Check bounds
	if tileX < 0 || tileX+footprintWidth > s.mapData.Width ||
		tileY < 0 || tileY+footprintHeight > s.mapData.Height {
		return
	}

	// Check for collisions with existing buildings (all tiles in footprint must be free)
	for dx := 0; dx < footprintWidth; dx++ {
		for dy := 0; dy < footprintHeight; dy++ {
			if s.isTileOccupiedByBuilding(tileX+dx, tileY+dy) {
				return
			}
		}
	}

	// Deduct money and create building
	client.Money -= BuildingCost

	entityId := s.nextId
	s.nextId++

	building := &Entity{
		Id:              entityId,
		OwnerId:         client.Id,
		Type:            buildingType,
		TileX:           tileX,
		TileY:           tileY,
		TargetTileX:     tileX,
		TargetTileY:     tileY,
		MoveProgress:    0.0,
		Health:          100,
		MaxHealth:       100,
		FootprintWidth:  footprintWidth,
		FootprintHeight: footprintHeight,
	}

	s.entities[entityId] = building

	log.Printf("Client %d built %s at tile (%d, %d)", client.Id, buildingType, tileX, tileY)
}

func (s *GameServer) handleAttackCommand(cmd Command, client *Client) {
	attackData, ok := cmd.Data.(map[string]interface{})
	if !ok {
		return
	}

	targetIdFloat, ok := attackData["targetId"].(float64)
	if !ok {
		return
	}
	targetId := uint32(targetIdFloat)

	// Find target entity
	target, exists := s.entities[targetId]
	if !exists {
		return
	}

	// Can't attack own entities
	if target.OwnerId == client.Id {
		return
	}

	// Only allow attacking buildings for now
	if target.Type != "generator" {
		return
	}

	// Apply damage
	damage := int32(25)
	target.Health -= damage

	log.Printf("Client %d attacked entity %d for %d damage (HP: %d)", client.Id, targetId, damage, target.Health)

	// Check if destroyed
	if target.Health <= 0 {
		delete(s.entities, targetId)
		log.Printf("Entity %d destroyed", targetId)
	}
	// No events needed - client will see health change / entity removal in snapshot
}

func (s *GameServer) broadcastMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling broadcast message: %v", err)
		return
	}

	s.mu.RLock()
	for _, client := range s.clients {
		s.conn.WriteToUDP(data, client.Addr)
	}
	s.mu.RUnlock()
}

func (s *GameServer) sendMessage(msg Message, addr *net.UDPAddr) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	s.conn.WriteToUDP(data, addr)
}

func (s *GameServer) marshalData(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return nil
	}
	return json.RawMessage(bytes)
}

func main() {
	// Load map (relative to server directory)
	mapData, err := LoadMap("../maps/default.json")
	if err != nil {
		log.Fatalf("Failed to load map: %v", err)
	}

	// Create server and assign map
	server := NewGameServer()
	server.mapData = mapData

	// Start server
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
