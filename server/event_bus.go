package main

import (
	"sync"
	"time"
)

// Event type constants for all game events
const (
	// Entity events
	EventEntityCreated   = "entity.created"
	EventEntityDestroyed = "entity.destroyed"
	EventEntityMoved     = "entity.moved"
	EventEntityDamaged   = "entity.damaged"

	// Player events
	EventPlayerJoined        = "player.joined"
	EventPlayerLeft          = "player.left"
	EventPlayerMoneyChanged  = "player.money_changed"
	EventPlayerCommandIssued = "player.command_issued"

	// Match events
	EventMatchStarted = "match.started"
	EventMatchTick    = "match.tick"
	EventMatchEnded   = "match.ended"

	// Action events
	EventBuildingPlaced    = "building.placed"
	EventBuildingDestroyed = "building.destroyed"
	EventUnitAttacked      = "unit.attacked"
	EventResourceGenerated = "resource.generated"
)

// GameEvent represents a structured event in the game system
type GameEvent struct {
	Type      string         // Event type (use constants above)
	Timestamp time.Time      // When the event occurred
	Tick      uint64         // Game tick when event occurred
	ActorID   uint32         // Who caused this event (player/unit ID)
	TargetID  uint32         // Who was affected by this event
	Data      map[string]any // Event-specific payload
}

// EventListener is a function that handles game events
type EventListener func(event GameEvent)

// EventBus manages event emission and subscription
type EventBus struct {
	listeners map[string][]EventListener
	mu        sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		listeners: make(map[string][]EventListener),
	}
}

// Emit broadcasts an event to all registered listeners
func (eb *EventBus) Emit(event GameEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if listeners, ok := eb.listeners[event.Type]; ok {
		for _, listener := range listeners {
			// Call listener synchronously for now
			// Could be made async with goroutines if needed
			listener(event)
		}
	}
}

// On registers a listener for a specific event type
func (eb *EventBus) On(eventType string, listener EventListener) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.listeners[eventType] = append(eb.listeners[eventType], listener)
}

// Off removes all listeners for a specific event type
func (eb *EventBus) Off(eventType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	delete(eb.listeners, eventType)
}

// Clear removes all event listeners
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.listeners = make(map[string][]EventListener)
}
