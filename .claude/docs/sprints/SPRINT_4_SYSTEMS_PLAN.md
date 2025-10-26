# Sprint 4 - Game Mode System Architecture

**Status:** 🚧 In Progress
**Started:** 2025-10-26
**Goal:** Build generic, data-driven systems that support any gameplay without hardcoded rules
**Philosophy:** Separate engine systems from game logic - experiment with gameplay later

---

## Core Principle

> **"The engine doesn't know what 'winning' means - game modes do."**

The server provides:
- ✅ Event emission (building placed, unit destroyed, money changed)
- ✅ Statistics tracking (generic metrics)
- ✅ Rule evaluation (condition checking)
- ✅ Match lifecycle (start, tick, end)

Game designers provide:
- 📝 JSON game mode definitions
- 📝 Win/lose conditions as declarative rules
- 📝 UI overlays and messages
- 📝 Match parameters (duration, starting resources, etc.)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Game Engine (Go)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │  Event Bus   │  │  Statistics  │  │ Rule Engine  │   │
│  │              │  │   Tracker    │  │              │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
│         │                  │                  │         │
│         └──────────────────┴──────────────────┘         │
│                            │                            │
└────────────────────────────┼────────────────────────────┘
                             │
                    ┌────────▼────────┐
                    │   Game Mode     │
                    │  Definition     │
                    │   (JSON/Lua)    │
                    └─────────────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────▼────┐   ┌─────▼─────┐  ┌────▼────┐
         │  Rules  │   │  Events   │  │   UI    │
         │ (when)  │   │  (what)   │  │ (show)  │
         └─────────┘   └───────────┘  └─────────┘
```

---

## System 1: Event Bus

### Purpose
Emit structured events for everything that happens in the game. Systems can subscribe to events.

### Event Types

```go
type GameEvent struct {
    Type      string                 // "entity.created", "player.money_changed", etc.
    Timestamp time.Time
    Tick      uint64
    ActorID   uint32                 // Who caused this (player/unit ID)
    TargetID  uint32                 // Who was affected
    Data      map[string]interface{} // Event-specific payload
}
```

### Core Events

**Entity Events:**
- `entity.created` - Unit or building spawned
- `entity.destroyed` - Unit or building destroyed
- `entity.moved` - Unit changed position
- `entity.damaged` - Unit/building took damage

**Player Events:**
- `player.joined` - Client connected
- `player.left` - Client disconnected
- `player.money_changed` - Money increased/decreased
- `player.command_issued` - Player sent command

**Match Events:**
- `match.started` - Game simulation begins
- `match.tick` - Every tick (for periodic checks)
- `match.ended` - Game concluded

**Action Events:**
- `building.placed` - Building constructed
- `building.destroyed` - Building demolished
- `unit.attacked` - Unit performed attack
- `resource.generated` - Money generated from building

### Implementation

```go
type EventBus struct {
    listeners map[string][]EventListener
    mu        sync.RWMutex
}

type EventListener func(event GameEvent)

func (eb *EventBus) Emit(event GameEvent) {
    eb.mu.RLock()
    defer eb.mu.RUnlock()

    if listeners, ok := eb.listeners[event.Type]; ok {
        for _, listener := range listeners {
            listener(event)
        }
    }
}

func (eb *EventBus) On(eventType string, listener EventListener) {
    eb.mu.Lock()
    defer eb.mu.Unlock()

    eb.listeners[eventType] = append(eb.listeners[eventType], listener)
}
```

### Usage Example

```go
// In game tick
s.eventBus.Emit(GameEvent{
    Type:      "player.money_changed",
    Timestamp: time.Now(),
    Tick:      s.currentTick,
    ActorID:   clientID,
    Data: map[string]interface{}{
        "oldMoney": oldMoney,
        "newMoney": newMoney,
        "delta":    newMoney - oldMoney,
    },
})
```

---

## System 2: Statistics Tracker

### Purpose
Track arbitrary metrics over time. Game modes can query stats to make decisions.

### Data Structure

```go
type StatisticsTracker struct {
    // Per-player counters
    counters map[uint32]map[string]float64  // counters[playerID][statName] = value

    // Per-player time series (optional - for graphs)
    timeSeries map[uint32]map[string][]TimePoint

    // Global match metrics
    globalCounters map[string]float64
}

type TimePoint struct {
    Tick  uint64
    Value float64
}
```

### Built-in Statistics

**Per-Player:**
- `money_current` - Current money
- `money_earned_total` - All-time earnings
- `money_spent_total` - All-time spending
- `buildings_placed` - Buildings constructed
- `buildings_destroyed` - Buildings demolished
- `buildings_lost` - Own buildings destroyed
- `units_killed` - Enemy units killed
- `units_lost` - Own units killed
- `damage_dealt` - Total damage inflicted
- `damage_taken` - Total damage received
- `tiles_controlled` - Territory control (optional)

**Global:**
- `match_duration_ticks` - Ticks elapsed
- `total_buildings` - All buildings on map
- `total_units` - All units alive

### Implementation

```go
func (st *StatisticsTracker) Increment(playerID uint32, stat string, delta float64) {
    if _, ok := st.counters[playerID]; !ok {
        st.counters[playerID] = make(map[string]float64)
    }
    st.counters[playerID][stat] += delta
}

func (st *StatisticsTracker) Set(playerID uint32, stat string, value float64) {
    if _, ok := st.counters[playerID]; !ok {
        st.counters[playerID] = make(map[string]float64)
    }
    st.counters[playerID][stat] = value
}

func (st *StatisticsTracker) Get(playerID uint32, stat string) float64 {
    if counters, ok := st.counters[playerID]; ok {
        return counters[stat]
    }
    return 0.0
}
```

### Auto-tracking from Events

```go
func (s *GameServer) setupStatisticsTracking() {
    // Automatically update stats based on events
    s.eventBus.On("building.placed", func(e GameEvent) {
        s.stats.Increment(e.ActorID, "buildings_placed", 1)
    })

    s.eventBus.On("building.destroyed", func(e GameEvent) {
        destroyerID := e.ActorID
        ownerID := e.TargetID
        s.stats.Increment(destroyerID, "buildings_destroyed", 1)
        s.stats.Increment(ownerID, "buildings_lost", 1)
    })

    s.eventBus.On("player.money_changed", func(e GameEvent) {
        delta := e.Data["delta"].(float64)
        newMoney := e.Data["newMoney"].(float64)

        s.stats.Set(e.ActorID, "money_current", newMoney)

        if delta > 0 {
            s.stats.Increment(e.ActorID, "money_earned_total", delta)
        } else {
            s.stats.Increment(e.ActorID, "money_spent_total", -delta)
        }
    })

    // ... more auto-tracking ...
}
```

---

## System 3: Rule Engine

### Purpose
Evaluate declarative conditions without hardcoding logic. Rules are expressions that return true/false.

### Rule Definition (JSON)

```json
{
  "id": "player_rich",
  "type": "comparison",
  "left": {"stat": "money_current", "player": "any"},
  "operator": ">=",
  "right": {"value": 500}
}
```

```json
{
  "id": "all_enemy_buildings_destroyed",
  "type": "comparison",
  "left": {"stat": "buildings_placed", "player": "opponent"},
  "operator": "==",
  "right": {"stat": "buildings_lost", "player": "opponent"}
}
```

```json
{
  "id": "time_limit_reached",
  "type": "comparison",
  "left": {"stat": "match_duration_ticks", "global": true},
  "operator": ">=",
  "right": {"value": 12000}
}
```

### Rule Types

**1. Comparison Rule**
- Compare two values (stats, constants)
- Operators: `==`, `!=`, `<`, `>`, `<=`, `>=`

**2. Logical Rule**
- Combine rules with AND/OR/NOT
```json
{
  "type": "and",
  "rules": [
    {"ref": "player_rich"},
    {"ref": "time_limit_reached"}
  ]
}
```

**3. Count Rule**
- Count entities matching criteria
```json
{
  "type": "count",
  "entity": "generator",
  "owner": "any",
  "operator": ">=",
  "value": 5
}
```

### Implementation

```go
type Rule interface {
    Evaluate(ctx *RuleContext) bool
}

type RuleContext struct {
    Server    *GameServer
    Stats     *StatisticsTracker
    EventBus  *EventBus
    Tick      uint64
    Variables map[string]interface{} // For parameterized rules
}

type ComparisonRule struct {
    Left     ValueExpression
    Operator string
    Right    ValueExpression
}

func (r *ComparisonRule) Evaluate(ctx *RuleContext) bool {
    leftVal := r.Left.Resolve(ctx)
    rightVal := r.Right.Resolve(ctx)

    switch r.Operator {
    case ">=":
        return leftVal >= rightVal
    case ">":
        return leftVal > rightVal
    case "==":
        return leftVal == rightVal
    // ... other operators ...
    }
    return false
}

type ValueExpression interface {
    Resolve(ctx *RuleContext) float64
}

type StatExpression struct {
    Stat     string
    PlayerID uint32 // 0 = all, specific ID, or "any" logic
}

func (e *StatExpression) Resolve(ctx *RuleContext) float64 {
    return ctx.Stats.Get(e.PlayerID, e.Stat)
}
```

---

## System 4: Game Mode Definition

### Purpose
JSON files that define complete game modes without touching code.

### Game Mode Schema

```json
{
  "id": "resource_race",
  "name": "Resource Race",
  "description": "First to $500 wins",

  "parameters": {
    "starting_money": 100,
    "starting_units": 5,
    "time_limit_ticks": 12000,
    "resource_goal": 500
  },

  "win_conditions": [
    {
      "id": "resource_victory",
      "name": "Resource Victory",
      "priority": 1,
      "rule": {
        "type": "comparison",
        "left": {"stat": "money_current", "player": "any"},
        "operator": ">=",
        "right": {"param": "resource_goal"}
      },
      "winner": "player_who_triggered"
    }
  ],

  "lose_conditions": [
    {
      "id": "bankruptcy",
      "name": "Bankrupt",
      "rule": {
        "type": "and",
        "rules": [
          {
            "type": "comparison",
            "left": {"stat": "money_current", "player": "self"},
            "operator": "<=",
            "right": {"value": 0}
          },
          {
            "type": "comparison",
            "left": {"stat": "buildings_placed", "player": "self"},
            "operator": "==",
            "right": {"value": 0}
          }
        ]
      }
    }
  ],

  "periodic_checks": [
    {
      "name": "resource_generation",
      "interval_ticks": 20,
      "action": {
        "type": "grant_resources",
        "amount_per_building": 10,
        "building_type": "generator"
      }
    }
  ],

  "events": [
    {
      "trigger": "player.money_changed",
      "condition": {
        "type": "comparison",
        "left": {"event_data": "newMoney"},
        "operator": ">=",
        "right": {"param": "resource_goal"}
      },
      "action": {
        "type": "check_win_conditions"
      }
    }
  ],

  "ui": {
    "objective_text": "Reach $500 before your opponent",
    "show_timer": true,
    "show_money": true,
    "victory_message": "{winner} achieved Resource Victory!",
    "defeat_message": "Defeated by {winner}'s economy"
  }
}
```

### Alternative Game Mode Examples

**Example: King of the Hill**
```json
{
  "id": "king_of_hill",
  "name": "King of the Hill",
  "description": "Control the center for 60 seconds",

  "parameters": {
    "control_radius": 5,
    "center_x": 20,
    "center_y": 15,
    "control_duration_ticks": 1200
  },

  "win_conditions": [
    {
      "id": "controlled_center",
      "rule": {
        "type": "comparison",
        "left": {"stat": "center_control_ticks", "player": "any"},
        "operator": ">=",
        "right": {"param": "control_duration_ticks"}
      }
    }
  ],

  "periodic_checks": [
    {
      "interval_ticks": 1,
      "action": {
        "type": "update_territory_control",
        "zone": "center",
        "radius": 5
      }
    }
  ]
}
```

**Example: Survival**
```json
{
  "id": "survival",
  "name": "Survival",
  "description": "Last player standing wins",

  "win_conditions": [
    {
      "id": "sole_survivor",
      "rule": {
        "type": "comparison",
        "left": {"count": "players_alive"},
        "operator": "==",
        "right": {"value": 1}
      }
    }
  ],

  "lose_conditions": [
    {
      "id": "eliminated",
      "rule": {
        "type": "and",
        "rules": [
          {
            "type": "comparison",
            "left": {"count_entities": {"type": "any", "owner": "self"}},
            "operator": "==",
            "right": {"value": 0}
          }
        ]
      }
    }
  ]
}
```

---

## System 5: Match Lifecycle Hooks

### Purpose
Allow game modes to inject logic at specific points without modifying core engine.

### Hook Points

```go
type GameMode interface {
    // Lifecycle hooks
    OnMatchCreated(ctx *MatchContext)     // Before players join
    OnPlayerJoined(ctx *MatchContext, playerID uint32)
    OnMatchStarting(ctx *MatchContext)    // Before countdown
    OnMatchStarted(ctx *MatchContext)     // Countdown finished
    OnTick(ctx *MatchContext)             // Every tick during play
    OnMatchEnding(ctx *MatchContext, reason string)
    OnMatchEnded(ctx *MatchContext)
    OnPlayerLeft(ctx *MatchContext, playerID uint32)

    // Query hooks
    CheckWinConditions(ctx *MatchContext) *WinResult
    CheckLoseConditions(ctx *MatchContext, playerID uint32) bool
    GetObjectiveText(ctx *MatchContext, playerID uint32) string
    GetStatistics(ctx *MatchContext) map[string]interface{}
}

type MatchContext struct {
    Server   *GameServer
    Stats    *StatisticsTracker
    EventBus *EventBus
    Mode     *GameModeDefinition
    Tick     uint64
}

type WinResult struct {
    HasWinner    bool
    WinnerID     uint32
    ConditionID  string
    ConditionName string
}
```

### JSON-Driven Game Mode Implementation

```go
type JSONGameMode struct {
    definition *GameModeDefinition
    ruleEngine *RuleEngine
}

func (m *JSONGameMode) OnTick(ctx *MatchContext) {
    // Execute periodic checks
    for _, check := range m.definition.PeriodicChecks {
        if ctx.Tick%check.IntervalTicks == 0 {
            m.executeAction(ctx, check.Action)
        }
    }
}

func (m *JSONGameMode) CheckWinConditions(ctx *MatchContext) *WinResult {
    // Evaluate all win conditions in priority order
    sort.Slice(m.definition.WinConditions, func(i, j int) bool {
        return m.definition.WinConditions[i].Priority < m.definition.WinConditions[j].Priority
    })

    for _, condition := range m.definition.WinConditions {
        ruleCtx := &RuleContext{
            Server:   ctx.Server,
            Stats:    ctx.Stats,
            EventBus: ctx.EventBus,
            Tick:     ctx.Tick,
        }

        if condition.Rule.Evaluate(ruleCtx) {
            winnerID := m.determineWinner(ctx, condition)
            return &WinResult{
                HasWinner:     true,
                WinnerID:      winnerID,
                ConditionID:   condition.ID,
                ConditionName: condition.Name,
            }
        }
    }

    return &WinResult{HasWinner: false}
}
```

---

## System 6: Generic UI Overlay System

### Purpose
Client can render UI based on game mode metadata without hardcoded screens.

### UI Message Types

**ObjectiveUpdate**
```json
{
  "type": "ui_objective",
  "data": {
    "text": "Reach $500 before your opponent",
    "progress": 0.65,
    "progressMax": 1.0
  }
}
```

**StatDisplay**
```json
{
  "type": "ui_stats",
  "data": {
    "stats": [
      {"label": "Money", "value": "$325", "icon": "coin"},
      {"label": "Buildings", "value": "7", "icon": "building"},
      {"label": "Units", "value": "12", "icon": "worker"}
    ]
  }
}
```

**MatchNotification**
```json
{
  "type": "ui_notification",
  "data": {
    "message": "You achieved Resource Victory!",
    "style": "victory",
    "duration": 5.0
  }
}
```

### Client UI Components

**Generic Overlay Manager (GDScript)**
```gdscript
class_name OverlayManager

var active_overlays: Array[Control] = []

func show_objective(text: String, progress: float, max_val: float):
    var overlay = preload("res://ui/ObjectiveOverlay.tscn").instantiate()
    overlay.set_text(text)
    overlay.set_progress(progress, max_val)
    add_child(overlay)

func show_notification(message: String, style: String, duration: float):
    var notification = preload("res://ui/Notification.tscn").instantiate()
    notification.set_message(message)
    notification.set_style(style)  # "victory", "defeat", "info", "warning"
    notification.auto_hide(duration)
    add_child(notification)

func show_stats_panel(stats: Array):
    # Generic stat display from array
    pass
```

---

## Implementation Plan

### Phase 1: Event Bus (Day 1)
1. ✅ Create `EventBus` struct with Emit/On methods
2. ✅ Define core event types as constants
3. ✅ Integrate event emission into existing systems:
   - Building placement → `building.placed`
   - Building destruction → `building.destroyed`
   - Money changes → `player.money_changed`
   - Unit movement → `entity.moved`
   - Combat → `unit.attacked`, `entity.damaged`
4. ✅ Add `eventBus` to `GameServer` struct
5. ✅ Write unit test: emit event, verify listener receives it

### Phase 2: Statistics Tracker (Day 1-2)
6. ✅ Create `StatisticsTracker` struct with Get/Set/Increment
7. ✅ Define standard statistics as constants
8. ✅ Auto-wire stats to event bus (`setupStatisticsTracking()`)
9. ✅ Add stats query endpoints (for debugging/testing)
10. ✅ Write unit test: trigger events, verify stats update

### Phase 3: Rule Engine (Day 2-3)
11. ✅ Define `Rule` interface and `RuleContext`
12. ✅ Implement `ComparisonRule` with value expressions
13. ✅ Implement `LogicalRule` (AND/OR/NOT)
14. ✅ Implement JSON rule parser
15. ✅ Write unit tests: evaluate rules against mock stats

### Phase 4: Game Mode Definition (Day 3-4)
16. ✅ Define game mode JSON schema
17. ✅ Create game mode loader (read JSON → struct)
18. ✅ Implement `JSONGameMode` that uses rules
19. ✅ Create 3 example modes:
    - `resource_race.json` - First to $X
    - `survival.json` - Last player standing
    - `king_of_hill.json` - Territory control
20. ✅ Load game mode from file on server start

### Phase 5: Match Lifecycle (Day 4-5)
21. ✅ Add `GameMode` interface with hook methods
22. ✅ Add match state to server (LOBBY, PLAYING, ENDED)
23. ✅ Call mode hooks at appropriate points in game loop
24. ✅ Implement win condition checking via mode
25. ✅ Broadcast match events to clients

### Phase 6: Client UI System (Day 5-6)
26. ✅ Create generic overlay scenes (Objective, Notification, Stats)
27. ✅ Implement `OverlayManager` in client
28. ✅ Handle UI messages from server
29. ✅ Test with different game modes (swap JSON, verify UI updates)

### Phase 7: Documentation & Testing (Day 6-7)
30. ✅ Document game mode JSON schema
31. ✅ Create game mode authoring guide
32. ✅ Write integration test: load mode, play match, verify winner
33. ✅ Write scenario test: custom mode with specific win condition
34. ✅ Update architecture documentation

---

## File Structure

```
server/
├── main.go                    # Existing game server
├── event_bus.go               # NEW: Event emission system
├── statistics.go              # NEW: Stat tracking
├── rule_engine.go             # NEW: Rule evaluation
├── game_mode.go               # NEW: GameMode interface
├── json_game_mode.go          # NEW: JSON-driven mode implementation
└── game_modes/                # NEW: Mode definitions
    ├── resource_race.json
    ├── survival.json
    └── king_of_hill.json

client/
├── ui/
│   ├── OverlayManager.gd      # NEW: Generic UI system
│   ├── ObjectiveOverlay.tscn  # NEW: Objective display
│   ├── Notification.tscn      # NEW: Notification popup
│   └── StatsPanel.tscn        # NEW: Statistics display
└── GameController.gd          # Modified: Handle UI messages
```

---

## Benefits of This Approach

### 1. **Experimentation Without Recompilation**
Change `resource_race.json` from $500 to $1000 → reload → test. No code changes.

### 2. **Multiple Game Modes**
Ship with 5 different modes. Let players choose. No engine changes needed.

### 3. **Theme-Agnostic**
Today: "Money" and "Generators"
Tomorrow: "Magic" and "Ley Lines"
→ Just swap JSON strings and art assets

### 4. **Community Mods**
Players can create custom game modes by writing JSON (no programming required).

### 5. **A/B Testing**
Run two modes simultaneously, see which is more fun. Data-driven design iteration.

### 6. **Future: Visual Editor**
Game mode editor can generate JSON (your stated goal).

---

## Example: Switching Gameplay Entirely

**Today's game mode** (`resource_race.json`):
```json
{
  "parameters": {"resource_goal": 500},
  "win_conditions": [{"rule": "money >= 500"}],
  "ui": {"objective_text": "Reach $500"}
}
```

**Tomorrow's game mode** (`tower_defense.json`):
```json
{
  "parameters": {"waves": 10},
  "win_conditions": [{"rule": "waves_survived >= 10"}],
  "lose_conditions": [{"rule": "base_health <= 0"}],
  "ui": {"objective_text": "Survive 10 waves"}
}
```

**Same engine. Zero code changes. Completely different game.**

---

## Success Criteria

Sprint 4 is complete when:

1. ✅ Event bus emits events for all major game actions
2. ✅ Statistics tracker auto-updates from events
3. ✅ Rule engine can evaluate JSON rules against stats
4. ✅ Game mode JSON files define win/lose conditions
5. ✅ Server loads game mode and checks conditions
6. ✅ Client displays objectives and notifications from mode
7. ✅ Three example game modes work end-to-end
8. ✅ Swapping JSON file changes gameplay (verified by playtest)
9. ✅ Documentation explains how to create custom modes

---

## Next Steps (Confirm with User)

Does this systems-oriented approach work for you? Key questions:

1. **JSON vs. Lua?** JSON is simple but limited. Lua scripts would be more powerful (custom logic). Preference?
2. **How much flexibility?** Should modes be able to spawn custom entities, modify map, etc.? Or just win conditions for now?
3. **Start with Phase 1** (Event Bus)? Or want to refine architecture first?

Let me know and I'll start building! 🚀
