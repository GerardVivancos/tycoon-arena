package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// GameMode defines the interface for game mode logic
type GameMode interface {
	// Lifecycle hooks
	OnMatchCreated(ctx *MatchContext)
	OnPlayerJoined(ctx *MatchContext, playerID uint32)
	OnMatchStarting(ctx *MatchContext)
	OnMatchStarted(ctx *MatchContext)
	OnTick(ctx *MatchContext)
	OnMatchEnding(ctx *MatchContext, reason string)
	OnMatchEnded(ctx *MatchContext)
	OnPlayerLeft(ctx *MatchContext, playerID uint32)

	// Query hooks
	CheckWinConditions(ctx *MatchContext) *WinResult
	CheckLoseConditions(ctx *MatchContext, playerID uint32) bool
	GetObjectiveText(ctx *MatchContext, playerID uint32) string
	GetStatistics(ctx *MatchContext) map[string]any

	// Metadata
	GetID() string
	GetName() string
	GetDescription() string
}

// MatchContext provides context for game mode hooks
type MatchContext struct {
	Server   *GameServer
	Stats    *StatisticsTracker
	EventBus *EventBus
	Mode     *GameModeDefinition
	Tick     uint64
}

// WinResult represents the result of checking win conditions
type WinResult struct {
	HasWinner     bool
	WinnerID      uint32
	ConditionID   string
	ConditionName string
}

// GameModeDefinition represents a game mode loaded from JSON
type GameModeDefinition struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`

	WinConditions  []WinCondition  `json:"win_conditions"`
	LoseConditions []LoseCondition `json:"lose_conditions"`
	PeriodicChecks []PeriodicCheck `json:"periodic_checks"`
	EventTriggers  []EventTrigger  `json:"event_triggers"`
	UI             UIConfig        `json:"ui"`
}

// WinCondition represents a condition that results in victory
type WinCondition struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Priority int      `json:"priority"` // Lower priority checked first
	Rule     RuleJSON `json:"rule"`
	Winner   string   `json:"winner"` // "player_who_triggered", "highest_stat", etc.
}

// LoseCondition represents a condition that results in defeat
type LoseCondition struct {
	ID   string   `json:"id"`
	Name string   `json:"name"`
	Rule RuleJSON `json:"rule"`
}

// PeriodicCheck represents an action that runs on an interval
type PeriodicCheck struct {
	Name          string         `json:"name"`
	IntervalTicks uint64         `json:"interval_ticks"`
	Action        map[string]any `json:"action"` // Action definition
}

// EventTrigger represents an action triggered by an event
type EventTrigger struct {
	Trigger   string         `json:"trigger"` // Event type
	Condition *RuleJSON      `json:"condition,omitempty"`
	Action    map[string]any `json:"action"`
}

// UIConfig defines UI elements for the game mode
type UIConfig struct {
	ObjectiveText  string `json:"objective_text"`
	ShowTimer      bool   `json:"show_timer"`
	ShowMoney      bool   `json:"show_money"`
	VictoryMessage string `json:"victory_message"`
	DefeatMessage  string `json:"defeat_message"`
}

// ===== JSONGameMode Implementation =====

// JSONGameMode implements GameMode using JSON definitions
type JSONGameMode struct {
	definition        *GameModeDefinition
	compiledWinRules  []compiledWinRule
	compiledLoseRules []compiledLoseRule
}

type compiledWinRule struct {
	condition *WinCondition
	rule      Rule
}

type compiledLoseRule struct {
	condition *LoseCondition
	rule      Rule
}

// NewJSONGameMode creates a game mode from a definition
func NewJSONGameMode(def *GameModeDefinition) (*JSONGameMode, error) {
	mode := &JSONGameMode{
		definition:        def,
		compiledWinRules:  make([]compiledWinRule, 0, len(def.WinConditions)),
		compiledLoseRules: make([]compiledLoseRule, 0, len(def.LoseConditions)),
	}

	// Compile win condition rules
	for i := range def.WinConditions {
		rule, err := ParseRule(def.WinConditions[i].Rule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse win condition %s: %w", def.WinConditions[i].ID, err)
		}
		mode.compiledWinRules = append(mode.compiledWinRules, compiledWinRule{
			condition: &def.WinConditions[i],
			rule:      rule,
		})
	}

	// Compile lose condition rules
	for i := range def.LoseConditions {
		rule, err := ParseRule(def.LoseConditions[i].Rule)
		if err != nil {
			return nil, fmt.Errorf("failed to parse lose condition %s: %w", def.LoseConditions[i].ID, err)
		}
		mode.compiledLoseRules = append(mode.compiledLoseRules, compiledLoseRule{
			condition: &def.LoseConditions[i],
			rule:      rule,
		})
	}

	return mode, nil
}

// Lifecycle hooks
func (m *JSONGameMode) OnMatchCreated(ctx *MatchContext)                  {}
func (m *JSONGameMode) OnPlayerJoined(ctx *MatchContext, playerID uint32) {}
func (m *JSONGameMode) OnMatchStarting(ctx *MatchContext)                 {}
func (m *JSONGameMode) OnMatchStarted(ctx *MatchContext)                  {}
func (m *JSONGameMode) OnMatchEnding(ctx *MatchContext, reason string)    {}
func (m *JSONGameMode) OnMatchEnded(ctx *MatchContext)                    {}
func (m *JSONGameMode) OnPlayerLeft(ctx *MatchContext, playerID uint32)   {}

func (m *JSONGameMode) OnTick(ctx *MatchContext) {
	// Execute periodic checks
	for _, check := range m.definition.PeriodicChecks {
		if ctx.Tick%check.IntervalTicks == 0 {
			m.executePeriodicCheck(ctx, &check)
		}
	}
}

// newRuleContext builds a RuleContext for evaluating rules for a given
// "current" player (0 for aggregate/global evaluation).
func (m *JSONGameMode) newRuleContext(ctx *MatchContext, players []uint32, currentPlayer uint32) *RuleContext {
	return &RuleContext{
		Server:        ctx.Server,
		Stats:         ctx.Stats,
		EventBus:      ctx.EventBus,
		Tick:          ctx.Tick,
		Variables:     m.definition.Parameters,
		CurrentPlayer: currentPlayer,
		Players:       players,
	}
}

// Query hooks
func (m *JSONGameMode) CheckWinConditions(ctx *MatchContext) *WinResult {
	// Sort by priority (lower first)
	sort.Slice(m.compiledWinRules, func(i, j int) bool {
		return m.compiledWinRules[i].condition.Priority < m.compiledWinRules[j].condition.Priority
	})

	// Sorted for deterministic iteration / winner selection
	players := ctx.Server.sortedClientIDs()

	for _, compiled := range m.compiledWinRules {
		switch compiled.condition.Winner {
		case "highest_money":
			// Global/aggregated condition; winner is the richest player
			if compiled.rule.Evaluate(m.newRuleContext(ctx, players, 0)) {
				if winner := m.playerWithHighestStat(ctx, StatMoneyCurrent, players); winner != 0 {
					return winResult(compiled.condition, winner)
				}
			}

		case "highest_stat":
			if compiled.rule.Evaluate(m.newRuleContext(ctx, players, 0)) {
				stat := ruleLeftStat(compiled.rule)
				if stat != "" {
					if winner := m.playerWithHighestStat(ctx, stat, players); winner != 0 {
						return winResult(compiled.condition, winner)
					}
				}
			}

		default: // "player_who_triggered"
			// Evaluate per-player; the first (lowest-id) player whose
			// self-based rule passes is the winner.
			for _, p := range players {
				if compiled.rule.Evaluate(m.newRuleContext(ctx, players, p)) {
					return winResult(compiled.condition, p)
				}
			}
		}
	}

	return &WinResult{HasWinner: false}
}

func (m *JSONGameMode) CheckLoseConditions(ctx *MatchContext, playerID uint32) bool {
	players := ctx.Server.sortedClientIDs()
	for _, compiled := range m.compiledLoseRules {
		if compiled.rule.Evaluate(m.newRuleContext(ctx, players, playerID)) {
			return true
		}
	}
	return false
}

// playerWithHighestStat returns the player with the maximum value of stat
// (ties broken by lowest id, given players is sorted ascending). 0 if none.
func (m *JSONGameMode) playerWithHighestStat(ctx *MatchContext, stat string, players []uint32) uint32 {
	var best uint32
	var bestVal float64
	found := false
	for _, p := range players {
		v := ctx.Stats.Get(p, stat)
		if !found || v > bestVal {
			bestVal = v
			best = p
			found = true
		}
	}
	return best
}

// winResult builds a WinResult from a condition and winner id.
func winResult(cond *WinCondition, winner uint32) *WinResult {
	return &WinResult{
		HasWinner:     true,
		WinnerID:      winner,
		ConditionID:   cond.ID,
		ConditionName: cond.Name,
	}
}

// ruleLeftStat returns the stat name of a comparison rule's left operand
// (used by the "highest_stat" winner strategy); "" if not a stat comparison.
func ruleLeftStat(rule Rule) string {
	if cr, ok := rule.(*ComparisonRule); ok {
		if sv, ok := cr.Left.(*StatValue); ok {
			return sv.Stat
		}
	}
	return ""
}

func (m *JSONGameMode) GetObjectiveText(ctx *MatchContext, playerID uint32) string {
	return m.definition.UI.ObjectiveText
}

func (m *JSONGameMode) GetStatistics(ctx *MatchContext) map[string]any {
	// Return all player statistics
	result := make(map[string]any)

	ctx.Server.mu.RLock()
	defer ctx.Server.mu.RUnlock()

	for clientID := range ctx.Server.clients {
		result[fmt.Sprintf("player_%d", clientID)] = ctx.Stats.GetAll(clientID)
	}

	// Add global stats
	result["global"] = map[string]any{
		"match_duration":  ctx.Stats.GetGlobal(StatGlobalMatchDurationTicks),
		"total_buildings": ctx.Stats.GetGlobal(StatGlobalTotalBuildings),
		"total_units":     ctx.Stats.GetGlobal(StatGlobalTotalUnits),
	}

	return result
}

// Metadata
func (m *JSONGameMode) GetID() string          { return m.definition.ID }
func (m *JSONGameMode) GetName() string        { return m.definition.Name }
func (m *JSONGameMode) GetDescription() string { return m.definition.Description }

// executePeriodicCheck runs a periodic action. Supports:
//   - grant_resources: {type, amount_per_building, building_type}
//   - increment_global_stat: {type, stat, amount}
func (m *JSONGameMode) executePeriodicCheck(ctx *MatchContext, check *PeriodicCheck) {
	if check.Action == nil {
		return
	}

	actionType, _ := check.Action["type"].(string)
	switch actionType {
	case "grant_resources":
		m.grantResources(ctx, check.Action)
	case "increment_global_stat":
		m.incrementGlobalStat(ctx, check.Action)
	}
}

func (m *JSONGameMode) grantResources(ctx *MatchContext, action map[string]any) {
	amountPerBuilding, ok := action["amount_per_building"].(float64)
	if !ok || amountPerBuilding <= 0 {
		return
	}
	buildingType, _ := action["building_type"].(string)

	for _, p := range ctx.Server.sortedClientIDs() {
		count := ctx.Server.countBuildings(p, buildingType)
		if count == 0 {
			continue
		}
		amount := amountPerBuilding * float64(count)
		// grantMoneyToPlayer updates client.Money and emits the money_changed
		// event, which keeps the money_current stat in sync.
		ctx.Server.grantMoneyToPlayer(p, float32(amount), "periodic_grant")
		ctx.Stats.Increment(p, StatResourcesGenerated, amount)
	}
}

func (m *JSONGameMode) incrementGlobalStat(ctx *MatchContext, action map[string]any) {
	stat, _ := action["stat"].(string)
	if stat == "" {
		return
	}
	amount, ok := action["amount"].(float64)
	if !ok {
		amount = 1
	}
	ctx.Stats.IncrementGlobal(stat, amount)
}

// ===== Loading Game Modes from Files =====

// LoadGameMode loads a game mode definition from a JSON file
func LoadGameMode(filepath string) (*GameModeDefinition, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var def GameModeDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return &def, nil
}

// LoadGameModeAndCreate loads and creates a JSONGameMode
func LoadGameModeAndCreate(filepath string) (*JSONGameMode, error) {
	def, err := LoadGameMode(filepath)
	if err != nil {
		return nil, err
	}

	return NewJSONGameMode(def)
}
