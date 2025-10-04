package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"
)

type MessageType string

const (
	MsgHello    MessageType = "hello"
	MsgWelcome  MessageType = "welcome"
	MsgInput    MessageType = "input"
	MsgSnapshot MessageType = "snapshot"
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
	ClientId uint32 `json:"clientId"`
	TickRate int    `json:"tickRate"`
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
		fmt.Printf("Received welcome! ClientId: %d, TickRate: %d\n", welcome.ClientId, welcome.TickRate)

		// Send a move command
		moveCmd := Command{
			Type: "move",
			Data: map[string]float32{"deltaX": 5, "deltaY": 2},
		}

		inputMsg := InputMessage{
			Tick:     1,
			ClientId: welcome.ClientId,
			Sequence: 1,
			Commands: []Command{moveCmd},
		}

		inputData, _ := json.Marshal(inputMsg)
		input := Message{
			Type: MsgInput,
			Data: json.RawMessage(inputData),
		}

		inputBytes, _ := json.Marshal(input)
		conn.Write(inputBytes)
		fmt.Println("Sent move command")

		// Listen for snapshots
		for i := 0; i < 3; i++ {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := conn.Read(buffer)
			if err != nil {
				log.Printf("Error reading snapshot: %v", err)
				continue
			}

			var snapMsg Message
			if err := json.Unmarshal(buffer[:n], &snapMsg); err != nil {
				log.Printf("Error unmarshaling snapshot: %v", err)
				continue
			}

			if snapMsg.Type == MsgSnapshot {
				var snapshot SnapshotMessage
				json.Unmarshal(snapMsg.Data, &snapshot)
				fmt.Printf("Snapshot tick %d: %d entities\n", snapshot.Tick, len(snapshot.Entities))
				for _, entity := range snapshot.Entities {
					fmt.Printf("  Entity %d: %s at (%.1f, %.1f)\n", entity.Id, entity.Type, entity.X, entity.Y)
				}
			}
		}
	}

	fmt.Println("Test complete!")
}