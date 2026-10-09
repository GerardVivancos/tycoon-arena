package main

import (
	"testing"
)

// newPlumbingTestMode builds a minimal game mode whose win condition fires once
// match_duration_ticks >= threshold. It only uses global stats so it does not
// depend on per-player rule resolution (Phase 2 work).
func newPlumbingTestMode(t *testing.T, threshold float64) *JSONGameMode {
	t.Helper()

	def := &GameModeDefinition{
		ID:          "plumbing_test",
		Name:        "Plumbing Test",
		Description: "Win when match_duration_ticks reaches the threshold",
		WinConditions: []WinCondition{
			{
				ID:       "tick_victory",
				Name:     "Tick Victory",
				Priority: 1,
				Winner:   "player_who_triggered",
				Rule: RuleJSON{
					Type:     "comparison",
					Left:     &ValueJSON{Stat: StatGlobalMatchDurationTicks, Global: true},
					Operator: ">=",
					Right:    &ValueJSON{Value: &threshold},
				},
			},
		},
		UI: UIConfig{
			ObjectiveText:  "Test objective",
			VictoryMessage: "Test victory",
			DefeatMessage:  "Test defeat",
		},
	}

	mode, err := NewJSONGameMode(def)
	if err != nil {
		t.Fatalf("failed to build test mode: %v", err)
	}
	return mode
}

func TestGameServerGameModePlumbing(t *testing.T) {
	server := NewGameServer()
	mode := newPlumbingTestMode(t, 100)
	server.setGameMode(mode, mode.definition)

	// Two players exist in the match
	server.mu.Lock()
	server.clients[1] = &Client{Id: 1, Name: "Alice"}
	server.clients[2] = &Client{Id: 2, Name: "Bob"}
	server.mu.Unlock()

	// Below threshold: match must not end
	server.checkMatchEnd()
	if server.gameOver {
		t.Fatal("match ended before win condition was met")
	}

	// Meet the threshold: match ends with one of the players as winner
	server.stats.SetGlobal(StatGlobalMatchDurationTicks, 100)
	server.checkMatchEnd()

	if !server.gameOver {
		t.Fatal("match did not end after win condition was met")
	}
	if server.winnerId != 1 && server.winnerId != 2 {
		t.Fatalf("unexpected winner id %d (want 1 or 2)", server.winnerId)
	}
}

func TestBuildGameOverMessage(t *testing.T) {
	server := NewGameServer()
	mode := newPlumbingTestMode(t, 100)
	server.setGameMode(mode, mode.definition)

	msg := server.buildGameOverMessage(&WinResult{
		HasWinner:     true,
		WinnerID:      7,
		ConditionID:   "tick_victory",
		ConditionName: "Tick Victory",
	})

	if msg.WinnerId != 7 {
		t.Errorf("expected winner id 7, got %d", msg.WinnerId)
	}
	if msg.ConditionId != "tick_victory" {
		t.Errorf("expected condition id 'tick_victory', got %q", msg.ConditionId)
	}
	if msg.ConditionName != "Tick Victory" {
		t.Errorf("expected condition name 'Tick Victory', got %q", msg.ConditionName)
	}
	if msg.ObjectiveText != "Test objective" {
		t.Errorf("expected objective 'Test objective', got %q", msg.ObjectiveText)
	}
	if msg.VictoryMessage != "Test victory" || msg.DefeatMessage != "Test defeat" {
		t.Errorf("unexpected victory/defeat messages: %q / %q", msg.VictoryMessage, msg.DefeatMessage)
	}
}

func TestCheckMatchEndNoMode(t *testing.T) {
	server := NewGameServer()
	server.stats.SetGlobal(StatGlobalMatchDurationTicks, 999999)

	// Must not panic or end the match when no mode is set
	server.checkMatchEnd()

	if server.gameOver {
		t.Fatal("match ended with no game mode set")
	}
}

func TestGameOverGuardsCommands(t *testing.T) {
	server := NewGameServer()
	server.gameOver = true

	client := &Client{Id: 1, Name: "A", Money: 1000}
	cmd := Command{Type: "build", Data: map[string]any{
		"buildingType": "generator",
		"tileX":        float64(2),
		"tileY":        float64(2),
	}}

	// processCommand is normally called with s.mu held by gameTick; reading
	// gameOver directly is safe here because this test is single-threaded.
	server.processCommand(cmd, client)

	if len(server.entities) != 0 {
		t.Fatalf("commands were processed after game over: %d entities", len(server.entities))
	}
}

func TestEndMatchIdempotent(t *testing.T) {
	server := NewGameServer()
	mode := newPlumbingTestMode(t, 100)
	server.setGameMode(mode, mode.definition)

	server.mu.Lock()
	server.clients[1] = &Client{Id: 1, Name: "A"}
	server.mu.Unlock()

	matchEndEvents := 0
	server.eventBus.On(EventMatchEnded, func(e GameEvent) {
		matchEndEvents++
	})

	result := &WinResult{
		HasWinner:     true,
		WinnerID:      1,
		ConditionID:   "tick_victory",
		ConditionName: "Tick Victory",
	}

	server.endMatch(result)
	server.endMatch(result) // second call must be a no-op

	if !server.gameOver {
		t.Fatal("gameOver not set after endMatch")
	}
	if server.winnerId != 1 {
		t.Fatalf("expected winner 1, got %d", server.winnerId)
	}
	if matchEndEvents != 1 {
		t.Fatalf("expected exactly 1 match.ended event, got %d", matchEndEvents)
	}
}
