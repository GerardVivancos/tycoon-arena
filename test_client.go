package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
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

type WelcomeMessage struct {
	ClientId          uint32 `json:"clientId"`
	TickRate          int    `json:"tickRate"`
	HeartbeatInterval int    `json:"heartbeatInterval"`
}

type InputMessage struct {
	Tick     uint64    `json:"tick"`
	ClientId uint32    `json:"clientId"`
	Sequence uint32    `json:"sequence"`
	Commands []Command `json:"commands"`
}

type Command struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type MoveCommand struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
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

func main() {
	// Connect to server
	conn, err := net.Dial("udp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Send hello message
	hello := HelloMessage{
		ClientVersion: "1.0",
		PlayerName:    "TestPlayer",
	}

	helloData, _ := json.Marshal(hello)
	helloMsg := Message{
		Type: MsgHello,
		Data: json.RawMessage(helloData),
	}

	msgBytes, _ := json.Marshal(helloMsg)
	conn.Write(msgBytes)

	fmt.Println("Sent hello message")

	// Listen for response
	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading response: %v", err)
		return
	}

	var response Message
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		log.Printf("Error unmarshaling response: %v", err)
		return
	}

	if response.Type == MsgWelcome {
		var welcome WelcomeMessage
		json.Unmarshal(response.Data, &welcome)
		fmt.Printf("Received welcome! ClientId: %d, TickRate: %d, Heartbeat: %dms\n",
			welcome.ClientId, welcome.TickRate, welcome.HeartbeatInterval)

		heartbeatInterval := time.Duration(welcome.HeartbeatInterval) * time.Millisecond

		// Start heartbeat goroutine
		stopHeartbeat := make(chan bool)
		go func() {
			ticker := time.NewTicker(heartbeatInterval)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					pingMsg := Message{
						Type: MsgPing,
						Data: json.RawMessage("{}"),
					}
					pingBytes, _ := json.Marshal(pingMsg)
					conn.Write(pingBytes)
					fmt.Println("Sent ping")
				case <-stopHeartbeat:
					return
				}
			}
		}()
		defer close(stopHeartbeat)

		// Start movement input goroutine
		var sequence uint32 = 0
		stopMovement := make(chan bool)
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			angle := 0.0

			for {
				select {
				case <-ticker.C:
					sequence++
					angle += 0.2

					// Move in a circular pattern
					deltaX := float32(3.0 * math.Cos(angle))
					deltaY := float32(3.0 * math.Sin(angle))

					moveCmd := Command{
						Type: "move",
						Data: map[string]float32{"deltaX": deltaX, "deltaY": deltaY},
					}

					inputMsg := InputMessage{
						Tick:     uint64(sequence),
						ClientId: welcome.ClientId,
						Sequence: sequence,
						Commands: []Command{moveCmd},
					}

					inputData, _ := json.Marshal(inputMsg)
					input := Message{
						Type: MsgInput,
						Data: json.RawMessage(inputData),
					}

					inputBytes, _ := json.Marshal(input)
					conn.Write(inputBytes)
				case <-stopMovement:
					return
				}
			}
		}()
		defer close(stopMovement)

		fmt.Println("Sending continuous movement commands (circular pattern)...")

		// Listen for snapshots and pongs
		timeout := time.After(10 * time.Second)
		snapshotCount := 0
		pongCount := 0

		for {
			select {
			case <-timeout:
				fmt.Printf("\nTest complete! Received %d snapshots and %d pongs\n", snapshotCount, pongCount)
				return
			default:
				conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				n, err := conn.Read(buffer)
				if err != nil {
					continue // Timeout is expected
				}

				var msg Message
				if err := json.Unmarshal(buffer[:n], &msg); err != nil {
					continue
				}

				switch msg.Type {
				case MsgSnapshot:
					var snapshot SnapshotMessage
					json.Unmarshal(msg.Data, &snapshot)
					snapshotCount++
					fmt.Printf("Snapshot tick %d: %d entities\n", snapshot.Tick, len(snapshot.Entities))
					for _, entity := range snapshot.Entities {
						fmt.Printf("  Entity %d: %s at (%.1f, %.1f)\n", entity.Id, entity.Type, entity.X, entity.Y)
					}
				case MsgPong:
					pongCount++
					fmt.Print(".")
				}
			}
		}
	}

	fmt.Println("Test complete!")
}
