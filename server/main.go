package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"
)

const (
	ServerPort        = ":8080"
	TickRate          = 20    // 20 Hz
	MaxClients        = 6
	TileSize          = 32    // World units per tile
	ArenaTilesWidth   = 25    // 800 / 32
	ArenaTilesHeight  = 18    // 576 / 32 (adjusted for clean division)
	ArenaWidth        = ArenaTilesWidth * TileSize  // 800
	ArenaHeight       = ArenaTilesHeight * TileSize // 576
	MovementSpeed     = 4.0   // tiles per second
	ClientTimeout     = 10 * time.Second // Timeout if no ping/input
	HeartbeatInterval = 2 * time.Second  // How often clients should ping

	// Game economy
	StartingMoney     = 100
	BuildingCost      = 50

	// Resource generation (money per second per building)
	GeneratorIncome   = 10.0
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
	Type MessageType `json:"type"`
	Data json.RawMessage `json:"data"`
}

type HelloMessage struct {
	ClientVersion string `json:"clientVersion"`
	PlayerName    string `json:"playerName"`
}

type WelcomeMessage struct {
	ClientId          uint32 `json:"clientId"`
	TickRate          int    `json:"tickRate"`
	HeartbeatInterval int    `json:"heartbeatInterval"` // milliseconds
	InputRedundancy   int    `json:"inputRedundancy"`   // How many commands to send per input
	TileSize          int    `json:"tileSize"`          // World units per tile
	ArenaTilesWidth   int    `json:"arenaTilesWidth"`
	ArenaTilesHeight  int    `json:"arenaTilesHeight"`
}

type InputMessage struct {
	ClientId uint32          `json:"clientId"`
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
	TargetTileX int `json:"targetTileX"`
	TargetTileY int `json:"targetTileY"`
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
}

type Client struct {
	Id                  uint32
	Name                string
	Addr                *net.UDPAddr
	LastSeen            time.Time
	Entity              *Entity
	Money               float32
	LastProcessedSeq    uint32
	LastAckTick         uint64 // For delta compression (not implemented)
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
			delete(s.clients, id)
			if client.Entity != nil {
				delete(s.entities, client.Entity.Id)
			}
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
		if entity.Type == "player" {
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

	// Create player entity
	entityId := s.nextId
	s.nextId++

	// Spawn at different positions based on number of existing clients
	spawnTileX := 3 + len(s.clients)*5
	spawnTileY := ArenaTilesHeight / 2

	entity := &Entity{
		Id:           entityId,
		OwnerId:      clientId,
		Type:         "player",
		TileX:        spawnTileX,
		TileY:        spawnTileY,
		TargetTileX:  spawnTileX,
		TargetTileY:  spawnTileY,
		MoveProgress: 0.0,
		Health:       100,
		MaxHealth:    100,
	}

	client := &Client{
		Id:       clientId,
		Name:     hello.PlayerName,
		Addr:     clientAddr,
		LastSeen: time.Now(),
		Entity:   entity,
		Money:    StartingMoney,
	}

	s.clients[clientId] = client
	s.entities[entityId] = entity

	log.Printf("Client %d (%s) connected from %s", clientId, hello.PlayerName, clientAddr.String())

	// Send welcome message
	welcome := WelcomeMessage{
		ClientId:          clientId,
		TickRate:          TickRate,
		HeartbeatInterval: int(HeartbeatInterval.Milliseconds()),
		InputRedundancy:   3, // Client should send last 3 commands
		TileSize:          TileSize,
		ArenaTilesWidth:   ArenaTilesWidth,
		ArenaTilesHeight:  ArenaTilesHeight,
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
	// Check if entity is moving
	if entity.TileX == entity.TargetTileX && entity.TileY == entity.TargetTileY {
		entity.MoveProgress = 0.0
		return
	}

	// Calculate movement progress increment
	// MovementSpeed is tiles/second, so progress per tick = (tiles/sec) * deltaTime / 1 tile
	progressIncrement := MovementSpeed * deltaTime

	entity.MoveProgress += progressIncrement

	// Check if reached target
	if entity.MoveProgress >= 1.0 {
		entity.TileX = entity.TargetTileX
		entity.TileY = entity.TargetTileY
		entity.MoveProgress = 0.0
	}
}

func (s *GameServer) handleMoveCommand(cmd Command, client *Client) {
	moveData, ok := cmd.Data.(map[string]interface{})
	if !ok || client.Entity == nil {
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
	if tileX < 0 || tileX >= ArenaTilesWidth || tileY < 0 || tileY >= ArenaTilesHeight {
		return
	}

	// Check if target tile has a building (can't move into buildings)
	if s.isTileOccupiedByBuilding(tileX, tileY) {
		return
	}

	// Set target (allow stacking of units)
	client.Entity.TargetTileX = tileX
	client.Entity.TargetTileY = tileY

	// If we're setting a new target while already moving, reset progress
	if client.Entity.TileX != tileX || client.Entity.TileY != tileY {
		client.Entity.MoveProgress = 0.0
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
	if tileX < 0 || tileX+footprintWidth > ArenaTilesWidth ||
		tileY < 0 || tileY+footprintHeight > ArenaTilesHeight {
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
	server := NewGameServer()
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}