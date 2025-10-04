package main

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

const (
	ServerPort = ":8080"
	TickRate   = 20 // 20 Hz
	MaxClients = 6
	ArenaWidth = 800
	ArenaHeight = 600
	PlayerSpeed = 200.0 // units per second
)

type MessageType string

const (
	MsgHello    MessageType = "hello"
	MsgWelcome  MessageType = "welcome"
	MsgInput    MessageType = "input"
	MsgSnapshot MessageType = "snapshot"
	MsgEvent    MessageType = "event"
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
	ClientId uint32 `json:"clientId"`
	TickRate int    `json:"tickRate"`
}

type InputMessage struct {
	Tick     uint64 `json:"tick"`
	ClientId uint32 `json:"clientId"`
	Sequence uint32 `json:"sequence"`
	Commands []Command `json:"commands"`
}

type Command struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type MoveCommand struct {
	DeltaX float32 `json:"deltaX"`
	DeltaY float32 `json:"deltaY"`
}

type SnapshotMessage struct {
	Tick     uint64   `json:"tick"`
	Entities []Entity `json:"entities"`
}

type Entity struct {
	Id        uint32  `json:"id"`
	OwnerId   uint32  `json:"ownerId"`
	Type      string  `json:"type"`
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Health    int32   `json:"health"`
	MaxHealth int32   `json:"maxHealth"`
}

type Client struct {
	Id       uint32
	Name     string
	Addr     *net.UDPAddr
	LastSeen time.Time
	Entity   *Entity
}

type GameServer struct {
	conn     *net.UDPConn
	clients  map[uint32]*Client
	entities map[uint32]*Entity
	tick     uint64
	nextId   uint32
	mu       sync.RWMutex
}

func NewGameServer() *GameServer {
	return &GameServer{
		clients:  make(map[uint32]*Client),
		entities: make(map[uint32]*Entity),
		tick:     0,
		nextId:   1,
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
	s.mu.Lock()
	s.tick++
	
	// Clean up disconnected clients (simple timeout)
	now := time.Now()
	for id, client := range s.clients {
		if now.Sub(client.LastSeen) > 30*time.Second {
			log.Printf("Client %d (%s) timed out", id, client.Name)
			delete(s.clients, id)
			if client.Entity != nil {
				delete(s.entities, client.Entity.Id)
			}
		}
	}
	
	// Create snapshot
	entities := make([]Entity, 0, len(s.entities))
	for _, entity := range s.entities {
		entities = append(entities, *entity)
	}
	
	snapshot := SnapshotMessage{
		Tick:     s.tick,
		Entities: entities,
	}
	s.mu.Unlock()

	// Send snapshot to all clients
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
	
	entity := &Entity{
		Id:        entityId,
		OwnerId:   clientId,
		Type:      "player",
		X:         float32(100 + len(s.clients) * 150), // Simple spawn positioning
		Y:         float32(ArenaHeight / 2),
		Health:    100,
		MaxHealth: 100,
	}

	client := &Client{
		Id:       clientId,
		Name:     hello.PlayerName,
		Addr:     clientAddr,
		LastSeen: time.Now(),
		Entity:   entity,
	}

	s.clients[clientId] = client
	s.entities[entityId] = entity

	log.Printf("Client %d (%s) connected from %s", clientId, hello.PlayerName, clientAddr.String())

	// Send welcome message
	welcome := WelcomeMessage{
		ClientId: clientId,
		TickRate: TickRate,
	}

	s.sendMessage(Message{
		Type: MsgWelcome,
		Data: s.marshalData(welcome),
	}, clientAddr)
}

func (s *GameServer) handleInput(input InputMessage, clientAddr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists := s.clients[input.ClientId]
	if !exists {
		return
	}

	// Update last seen
	client.LastSeen = time.Now()

	// Process commands
	for _, cmd := range input.Commands {
		switch cmd.Type {
		case "move":
			if moveData, ok := cmd.Data.(map[string]interface{}); ok {
				if client.Entity != nil {
					// Apply delta movement with speed limit
					deltaTime := 1.0 / float32(TickRate)
					maxDelta := PlayerSpeed * deltaTime

					if dx, ok := moveData["deltaX"].(float64); ok {
						deltaX := float32(dx)
						if deltaX > maxDelta {
							deltaX = maxDelta
						} else if deltaX < -maxDelta {
							deltaX = -maxDelta
						}
						client.Entity.X += deltaX
					}
					if dy, ok := moveData["deltaY"].(float64); ok {
						deltaY := float32(dy)
						if deltaY > maxDelta {
							deltaY = maxDelta
						} else if deltaY < -maxDelta {
							deltaY = -maxDelta
						}
						client.Entity.Y += deltaY
					}

					// Apply arena bounds
					if client.Entity.X < 0 {
						client.Entity.X = 0
					} else if client.Entity.X > ArenaWidth {
						client.Entity.X = ArenaWidth
					}
					if client.Entity.Y < 0 {
						client.Entity.Y = 0
					} else if client.Entity.Y > ArenaHeight {
						client.Entity.Y = ArenaHeight
					}
				}
			}
		}
	}
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