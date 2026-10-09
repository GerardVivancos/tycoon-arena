package main

import (
	"sync"
	"testing"
	"time"
)

func TestEventBusEmitAndReceive(t *testing.T) {
	bus := NewEventBus()

	received := false
	var receivedEvent GameEvent

	// Register listener
	bus.On(EventPlayerMoneyChanged, func(event GameEvent) {
		received = true
		receivedEvent = event
	})

	// Emit event
	testEvent := GameEvent{
		Type:      EventPlayerMoneyChanged,
		Timestamp: time.Now(),
		Tick:      100,
		ActorID:   1,
		Data: map[string]any{
			"oldMoney": 100.0,
			"newMoney": 150.0,
			"delta":    50.0,
		},
	}
	bus.Emit(testEvent)

	// Verify listener received event
	if !received {
		t.Fatal("Event listener was not called")
	}

	if receivedEvent.Type != EventPlayerMoneyChanged {
		t.Errorf("Expected event type %s, got %s", EventPlayerMoneyChanged, receivedEvent.Type)
	}

	if receivedEvent.Tick != 100 {
		t.Errorf("Expected tick 100, got %d", receivedEvent.Tick)
	}

	if receivedEvent.ActorID != 1 {
		t.Errorf("Expected actor ID 1, got %d", receivedEvent.ActorID)
	}

	delta, ok := receivedEvent.Data["delta"].(float64)
	if !ok || delta != 50.0 {
		t.Errorf("Expected delta 50.0, got %v", receivedEvent.Data["delta"])
	}
}

func TestEventBusMultipleListeners(t *testing.T) {
	bus := NewEventBus()

	count := 0
	var mu sync.Mutex

	// Register multiple listeners for same event
	bus.On(EventBuildingPlaced, func(event GameEvent) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	bus.On(EventBuildingPlaced, func(event GameEvent) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	bus.On(EventBuildingPlaced, func(event GameEvent) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	// Emit event
	bus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      50,
		ActorID:   2,
	})

	// Verify all listeners were called
	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("Expected 3 listeners to be called, got %d", count)
	}
}

func TestEventBusNoListeners(t *testing.T) {
	bus := NewEventBus()

	// Emit event with no listeners (should not panic)
	bus.Emit(GameEvent{
		Type:      EventEntityDestroyed,
		Timestamp: time.Now(),
		Tick:      75,
		ActorID:   3,
	})

	// If we got here, test passed
}

func TestEventBusDifferentEventTypes(t *testing.T) {
	bus := NewEventBus()

	moneyEventReceived := false
	buildEventReceived := false

	bus.On(EventPlayerMoneyChanged, func(event GameEvent) {
		moneyEventReceived = true
	})

	bus.On(EventBuildingPlaced, func(event GameEvent) {
		buildEventReceived = true
	})

	// Emit money event
	bus.Emit(GameEvent{
		Type:      EventPlayerMoneyChanged,
		Timestamp: time.Now(),
		Tick:      10,
		ActorID:   1,
	})

	if !moneyEventReceived {
		t.Error("Money event listener was not called")
	}
	if buildEventReceived {
		t.Error("Build event listener should not have been called")
	}

	// Reset and emit build event
	moneyEventReceived = false
	buildEventReceived = false

	bus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      20,
		ActorID:   1,
	})

	if moneyEventReceived {
		t.Error("Money event listener should not have been called")
	}
	if !buildEventReceived {
		t.Error("Build event listener was not called")
	}
}

func TestEventBusClear(t *testing.T) {
	bus := NewEventBus()

	count := 0

	bus.On(EventBuildingPlaced, func(event GameEvent) {
		count++
	})

	// Emit event - should be received
	bus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      1,
	})

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Clear all listeners
	bus.Clear()

	// Emit event again - should not be received
	bus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      2,
	})

	if count != 1 {
		t.Errorf("Expected count to remain 1 after clear, got %d", count)
	}
}

func TestEventBusOff(t *testing.T) {
	bus := NewEventBus()

	count := 0

	bus.On(EventBuildingPlaced, func(event GameEvent) {
		count++
	})

	// Emit event - should be received
	bus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      1,
	})

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}

	// Remove listeners for this event type
	bus.Off(EventBuildingPlaced)

	// Emit event again - should not be received
	bus.Emit(GameEvent{
		Type:      EventBuildingPlaced,
		Timestamp: time.Now(),
		Tick:      2,
	})

	if count != 1 {
		t.Errorf("Expected count to remain 1 after Off, got %d", count)
	}
}
