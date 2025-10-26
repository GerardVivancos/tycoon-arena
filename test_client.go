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

type TerrainTile struct {
	X      int     `json:"x"`
	Y      int     `json:"y"`
	Type   string  `json:"type"`
	Height float32 `json:"height"`
}

type TerrainData struct {
	DefaultType string        `json:"defaultType"`
	Tiles       []TerrainTile `json:"tiles"`
}

type WelcomeMessage struct {
	ClientId          uint32      `json:"clientId"`
	TickRate          int         `json:"tickRate"`
	HeartbeatInterval int         `json:"heartbeatInterval"`
	InputRedundancy   int         `json:"inputRedundancy"`
	TileSize          int         `json:"tileSize"`
	ArenaTilesWidth   int         `json:"arenaTilesWidth"`
	ArenaTilesHeight  int         `json:"arenaTilesHeight"`
	TerrainData       TerrainData `json:"terrainData"`
}

type Command struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type CommandFrame struct {
	Sequence uint32    `json:"sequence"`
	Tick     uint64    `json:"tick"`
	Commands []Command `json:"commands"`
}

type InputPayload struct {
	ClientId uint32         `json:"clientId"`
	Commands []CommandFrame `json:"commands"`
}

type SnapshotEntity struct {
	Id              uint32  `json:"id"`
	OwnerId         uint32  `json:"ownerId"`
	Type            string  `json:"type"`
	TileX           int     `json:"tileX"`
	TileY           int     `json:"tileY"`
	TargetTileX     int     `json:"targetTileX"`
	TargetTileY     int     `json:"targetTileY"`
	MoveProgress    float32 `json:"moveProgress"`
	FootprintWidth  int     `json:"footprintWidth"`
	FootprintHeight int     `json:"footprintHeight"`
	Health          int32   `json:"health"`
	MaxHealth       int32   `json:"maxHealth"`
}

type SnapshotMessage struct {
	Tick         uint64           `json:"tick"`
	BaselineTick uint64           `json:"baselineTick"`
	Entities     []SnapshotEntity `json:"entities"`
}

type tileTarget struct {
	X int
	Y int
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func buildTargets(width, height int) []tileTarget {
	if width <= 0 || height <= 0 {
		return []tileTarget{{X: 0, Y: 0}}
	}
	centerX := clamp(width/2, 0, width-1)
	centerY := clamp(height/2, 0, height-1)
	offsets := [][2]int{{3, 0}, {0, 3}, {-3, 0}, {0, -3}}
	targets := make([]tileTarget, 0, len(offsets))
	for _, offset := range offsets {
		targets = append(targets, tileTarget{
			X: clamp(centerX+offset[0], 0, width-1),
			Y: clamp(centerY+offset[1], 0, height-1),
		})
	}
	return targets
}

func main() {
	conn, err := net.Dial("udp", "localhost:8080")
	if err != nil {
		log.Fatalf("failed to dial server: %v", err)
	}
	defer conn.Close()

	hello := HelloMessage{
		ClientVersion: "1.0",
		PlayerName:    "TestClient",
	}
	helloBytes, _ := json.Marshal(hello)
	if err := sendMessage(conn, MsgHello, helloBytes); err != nil {
		log.Fatalf("failed to send hello: %v", err)
	}
	fmt.Println("Sent hello message")

	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		log.Fatalf("failed to read welcome: %v", err)
	}

	var envelope Message
	if err := json.Unmarshal(buffer[:n], &envelope); err != nil {
		log.Fatalf("failed to parse welcome envelope: %v", err)
	}
	if envelope.Type != MsgWelcome {
		log.Fatalf("expected welcome message, got %s", envelope.Type)
	}

	var welcome WelcomeMessage
	if err := json.Unmarshal(envelope.Data, &welcome); err != nil {
		log.Fatalf("failed to parse welcome payload: %v", err)
	}

	fmt.Printf("Received welcome! ClientId: %d, TickRate: %dHz, Heartbeat: %dms\n",
		welcome.ClientId, welcome.TickRate, welcome.HeartbeatInterval)
	fmt.Printf("Map: %dx%d tiles (size %d)\n", welcome.ArenaTilesWidth, welcome.ArenaTilesHeight, welcome.TileSize)
	fmt.Printf("Terrain tiles received: %d\n", len(welcome.TerrainData.Tiles))

	inputRedundancy := welcome.InputRedundancy
	if inputRedundancy <= 0 {
		inputRedundancy = 3
	}

	var (
		currentTick    uint64
		tickMu         sync.RWMutex
		ownedUnits     []uint32
		ownedUnitsMu   sync.RWMutex
		commandHistory []CommandFrame
		commandMu      sync.Mutex
	)

	stopHeartbeat := make(chan struct{})
	go func(interval int) {
		ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := sendMessage(conn, MsgPing, []byte(`{}`)); err != nil {
					log.Printf("failed to send ping: %v", err)
				} else {
					fmt.Println("Sent ping")
				}
			case <-stopHeartbeat:
				return
			}
		}
	}(welcome.HeartbeatInterval)

	targets := buildTargets(welcome.ArenaTilesWidth, welcome.ArenaTilesHeight)
	stopMovement := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		var sequence uint32
		targetIndex := 0

		for {
			select {
			case <-ticker.C:
				ownedUnitsMu.RLock()
				if len(ownedUnits) == 0 {
					ownedUnitsMu.RUnlock()
					continue
				}
				unitIDs := append([]uint32(nil), ownedUnits...)
				ownedUnitsMu.RUnlock()

				tickMu.RLock()
				tick := currentTick
				tickMu.RUnlock()

				target := targets[targetIndex]
				targetIndex = (targetIndex + 1) % len(targets)

				sequence++
				frame := CommandFrame{
					Sequence: sequence,
					Tick:     tick,
					Commands: []Command{
						{
							Type: "move",
							Data: map[string]interface{}{
								"unitIds":     unitIDs,
								"targetTileX": target.X,
								"targetTileY": target.Y,
								"formation":   "box",
							},
						},
					},
				}

				commandMu.Lock()
				commandHistory = append(commandHistory, frame)
				if len(commandHistory) > inputRedundancy {
					commandHistory = commandHistory[len(commandHistory)-inputRedundancy:]
				}
				payload := InputPayload{
					ClientId: welcome.ClientId,
					Commands: append([]CommandFrame(nil), commandHistory...),
				}
				commandMu.Unlock()

				payloadBytes, _ := json.Marshal(payload)
				if err := sendMessage(conn, MsgInput, payloadBytes); err != nil {
					log.Printf("failed to send move command: %v", err)
				} else {
					fmt.Printf("Sent move command to (%d,%d) for %d unit(s)\n", target.X, target.Y, len(unitIDs))
				}
			case <-stopMovement:
				return
			}
		}
	}()

	timeout := time.After(10 * time.Second)
	snapshotCount := 0
	pongCount := 0

	for {
		select {
		case <-timeout:
			close(stopMovement)
			close(stopHeartbeat)
			fmt.Printf("\nTest complete! Received %d snapshots and %d pongs\n", snapshotCount, pongCount)
			return
		default:
			conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
			n, err := conn.Read(buffer)
			if err != nil {
				continue
			}

			var msg Message
			if err := json.Unmarshal(buffer[:n], &msg); err != nil {
				continue
			}

			switch msg.Type {
			case MsgSnapshot:
				var snapshot SnapshotMessage
				if err := json.Unmarshal(msg.Data, &snapshot); err != nil {
					log.Printf("failed to parse snapshot: %v", err)
					continue
				}
				tickMu.Lock()
				currentTick = snapshot.Tick
				tickMu.Unlock()

				snapshotCount++
				fmt.Printf("Snapshot tick %d: %d entities\n", snapshot.Tick, len(snapshot.Entities))

				var unitCandidates []uint32
				for _, entity := range snapshot.Entities {
					if entity.OwnerId == welcome.ClientId && entity.Type == "worker" {
						unitCandidates = append(unitCandidates, entity.Id)
					}
					fmt.Printf("  Entity %d (%s) owner %d tile (%d,%d) → (%d,%d)\n",
						entity.Id, entity.Type, entity.OwnerId,
						entity.TileX, entity.TileY, entity.TargetTileX, entity.TargetTileY)
				}

				if len(unitCandidates) > 0 {
					sort.Slice(unitCandidates, func(i, j int) bool { return unitCandidates[i] < unitCandidates[j] })
					ownedUnitsMu.Lock()
					ownedUnits = unitCandidates
					ownedUnitsMu.Unlock()
				}
			case MsgPong:
				pongCount++
				fmt.Print(".")
			}
		}
	}
}

func sendMessage(conn net.Conn, msgType MessageType, payload []byte) error {
	envelope := Message{
		Type: msgType,
		Data: json.RawMessage(payload),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	_, err = conn.Write(data)
	return err
}
