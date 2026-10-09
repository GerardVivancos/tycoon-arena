package main

import (
	"testing"
)

func TestComparisonRuleBasic(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 500.0)

	ctx := &RuleContext{
		Stats: stats,
		Tick:  100,
	}

	// Test >= operator
	rule := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
		Operator: ">=",
		Right:    &ConstantValue{Value: 500.0},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected rule to evaluate to true (500 >= 500)")
	}

	// Test < operator
	rule2 := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
		Operator: "<",
		Right:    &ConstantValue{Value: 600.0},
	}

	if !rule2.Evaluate(ctx) {
		t.Error("Expected rule to evaluate to true (500 < 600)")
	}

	// Test == operator
	rule3 := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
		Operator: "==",
		Right:    &ConstantValue{Value: 500.0},
	}

	if !rule3.Evaluate(ctx) {
		t.Error("Expected rule to evaluate to true (500 == 500)")
	}
}

func TestComparisonRuleAllOperators(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 100.0)

	ctx := &RuleContext{
		Stats: stats,
	}

	tests := []struct {
		operator string
		right    float64
		expected bool
	}{
		{"==", 100.0, true},
		{"==", 200.0, false},
		{"!=", 200.0, true},
		{"!=", 100.0, false},
		{"<", 200.0, true},
		{"<", 100.0, false},
		{">", 50.0, true},
		{">", 100.0, false},
		{"<=", 100.0, true},
		{"<=", 99.0, false},
		{">=", 100.0, true},
		{">=", 101.0, false},
	}

	for _, tt := range tests {
		rule := &ComparisonRule{
			Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
			Operator: tt.operator,
			Right:    &ConstantValue{Value: tt.right},
		}

		result := rule.Evaluate(ctx)
		if result != tt.expected {
			t.Errorf("Operator %s with right=%f: expected %v, got %v",
				tt.operator, tt.right, tt.expected, result)
		}
	}
}

func TestLogicalRuleAnd(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 500.0)
	stats.Set(1, StatBuildingsPlaced, 5.0)

	ctx := &RuleContext{
		Stats: stats,
	}

	// Both conditions true
	rule := &LogicalRule{
		Operator: "and",
		Rules: []Rule{
			&ComparisonRule{
				Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 500.0},
			},
			&ComparisonRule{
				Left:     &StatValue{Stat: StatBuildingsPlaced, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 5.0},
			},
		},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected AND rule to evaluate to true when both conditions true")
	}

	// One condition false
	rule2 := &LogicalRule{
		Operator: "and",
		Rules: []Rule{
			&ComparisonRule{
				Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 600.0}, // False
			},
			&ComparisonRule{
				Left:     &StatValue{Stat: StatBuildingsPlaced, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 5.0}, // True
			},
		},
	}

	if rule2.Evaluate(ctx) {
		t.Error("Expected AND rule to evaluate to false when one condition false")
	}
}

func TestLogicalRuleOr(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 500.0)
	stats.Set(1, StatBuildingsPlaced, 3.0)

	ctx := &RuleContext{
		Stats: stats,
	}

	// One condition true
	rule := &LogicalRule{
		Operator: "or",
		Rules: []Rule{
			&ComparisonRule{
				Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 500.0}, // True
			},
			&ComparisonRule{
				Left:     &StatValue{Stat: StatBuildingsPlaced, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 5.0}, // False
			},
		},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected OR rule to evaluate to true when one condition true")
	}

	// Both conditions false
	rule2 := &LogicalRule{
		Operator: "or",
		Rules: []Rule{
			&ComparisonRule{
				Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 600.0}, // False
			},
			&ComparisonRule{
				Left:     &StatValue{Stat: StatBuildingsPlaced, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 5.0}, // False
			},
		},
	}

	if rule2.Evaluate(ctx) {
		t.Error("Expected OR rule to evaluate to false when both conditions false")
	}
}

func TestLogicalRuleNot(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 500.0)

	ctx := &RuleContext{
		Stats: stats,
	}

	// NOT true = false
	rule := &LogicalRule{
		Operator: "not",
		Rules: []Rule{
			&ComparisonRule{
				Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 500.0}, // True
			},
		},
	}

	if rule.Evaluate(ctx) {
		t.Error("Expected NOT rule to evaluate to false when condition is true")
	}

	// NOT false = true
	rule2 := &LogicalRule{
		Operator: "not",
		Rules: []Rule{
			&ComparisonRule{
				Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 600.0}, // False
			},
		},
	}

	if !rule2.Evaluate(ctx) {
		t.Error("Expected NOT rule to evaluate to true when condition is false")
	}
}

func TestGlobalStats(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.SetGlobal(StatGlobalMatchDurationTicks, 12000.0)

	ctx := &RuleContext{
		Stats: stats,
	}

	rule := &ComparisonRule{
		Left:     &StatValue{Stat: StatGlobalMatchDurationTicks, Global: true},
		Operator: ">=",
		Right:    &ConstantValue{Value: 12000.0},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected global stat rule to evaluate to true")
	}
}

func TestParameterValues(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 500.0)

	ctx := &RuleContext{
		Stats: stats,
		Variables: map[string]any{
			"resource_goal": 500.0,
		},
	}

	rule := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
		Operator: ">=",
		Right:    &ParameterValue{ParamName: "resource_goal"},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected parameter value rule to evaluate to true")
	}

	// Change parameter
	ctx.Variables["resource_goal"] = 600.0

	if rule.Evaluate(ctx) {
		t.Error("Expected parameter value rule to evaluate to false after parameter change")
	}
}

func TestParseRuleFromJSON(t *testing.T) {
	// Simple comparison rule
	jsonStr := `{
		"type": "comparison",
		"left": {"stat": "money_current", "player": "any"},
		"operator": ">=",
		"right": {"value": 500}
	}`

	rule, err := ParseRuleFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse JSON rule: %v", err)
	}

	stats := NewStatisticsTracker()
	// "any" aggregates (max) across ctx.Players, so the player must be listed.
	stats.Set(0, StatMoneyCurrent, 600.0)

	ctx := &RuleContext{
		Stats:   stats,
		Players: []uint32{0},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected parsed rule to evaluate to true")
	}
}

func TestParseLogicalRuleFromJSON(t *testing.T) {
	jsonStr := `{
		"type": "and",
		"rules": [
			{
				"type": "comparison",
				"left": {"stat": "money_current", "player": "any"},
				"operator": ">=",
				"right": {"value": 500}
			},
			{
				"type": "comparison",
				"left": {"stat": "buildings_placed", "player": "any"},
				"operator": ">=",
				"right": {"value": 3}
			}
		]
	}`

	rule, err := ParseRuleFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse JSON rule: %v", err)
	}

	stats := NewStatisticsTracker()
	// "any" aggregates (max) across ctx.Players, so the player must be listed.
	stats.Set(0, StatMoneyCurrent, 600.0)
	stats.Set(0, StatBuildingsPlaced, 5.0)

	ctx := &RuleContext{
		Stats:   stats,
		Players: []uint32{0},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected parsed logical rule to evaluate to true")
	}
}

func TestParseRuleWithParameter(t *testing.T) {
	jsonStr := `{
		"type": "comparison",
		"left": {"stat": "money_current", "player": "any"},
		"operator": ">=",
		"right": {"param": "resource_goal"}
	}`

	rule, err := ParseRuleFromJSON(jsonStr)
	if err != nil {
		t.Fatalf("Failed to parse JSON rule: %v", err)
	}

	stats := NewStatisticsTracker()
	// "any" aggregates (max) across ctx.Players, so the player must be listed.
	stats.Set(0, StatMoneyCurrent, 500.0)

	ctx := &RuleContext{
		Stats:   stats,
		Players: []uint32{0},
		Variables: map[string]any{
			"resource_goal": 500.0,
		},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected parsed parameter rule to evaluate to true")
	}
}

func TestStatValuePerPlayerResolution(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 100.0)
	stats.Set(2, StatMoneyCurrent, 300.0)
	stats.Set(3, StatMoneyCurrent, 200.0)

	players := []uint32{1, 2, 3}

	// self -> current player
	selfRule := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerRef: "self"},
		Operator: ">=",
		Right:    &ConstantValue{Value: 300.0},
	}
	ctxSelf := &RuleContext{Stats: stats, CurrentPlayer: 2, Players: players}
	if !selfRule.Evaluate(ctxSelf) {
		t.Error("self: expected player 2 (300) to satisfy >= 300")
	}
	ctxSelf1 := &RuleContext{Stats: stats, CurrentPlayer: 1, Players: players}
	if selfRule.Evaluate(ctxSelf1) {
		t.Error("self: expected player 1 (100) to fail >= 300")
	}

	// any -> max across all players (300)
	anyRule := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerRef: "any"},
		Operator: ">=",
		Right:    &ConstantValue{Value: 300.0},
	}
	ctxAny := &RuleContext{Stats: stats, Players: players}
	if !anyRule.Evaluate(ctxAny) {
		t.Error("any: expected aggregate max (300) to satisfy >= 300")
	}

	// opponent -> max excluding current (current=2 -> max of {100,200}=200)
	oppRule := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerRef: "opponent"},
		Operator: ">=",
		Right:    &ConstantValue{Value: 200.0},
	}
	ctxOpp := &RuleContext{Stats: stats, CurrentPlayer: 2, Players: players}
	if !oppRule.Evaluate(ctxOpp) {
		t.Error("opponent: expected max excluding player 2 (200) to satisfy >= 200")
	}
	// With current=3, opponent max is {100,300}=300; >= 300 true, but ensure
	// it does NOT include the current player's own value:
	oppRule2 := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerRef: "opponent"},
		Operator: ">=",
		Right:    &ConstantValue{Value: 300.0},
	}
	ctxOpp3 := &RuleContext{Stats: stats, CurrentPlayer: 3, Players: players}
	if !oppRule2.Evaluate(ctxOpp3) {
		t.Error("opponent: expected max excluding player 3 (300) to satisfy >= 300")
	}
	oppRule3 := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerRef: "opponent"},
		Operator: ">=",
		Right:    &ConstantValue{Value: 301.0},
	}
	ctxOpp2 := &RuleContext{Stats: stats, CurrentPlayer: 2, Players: players}
	if oppRule3.Evaluate(ctxOpp2) {
		t.Error("opponent: max excluding player 2 is 200, should fail >= 301")
	}

	// numeric id selector
	idRule := &ComparisonRule{
		Left:     &StatValue{Stat: StatMoneyCurrent, PlayerRef: "3", PlayerID: 3},
		Operator: "==",
		Right:    &ConstantValue{Value: 200.0},
	}
	ctxID := &RuleContext{Stats: stats, Players: players}
	if !idRule.Evaluate(ctxID) {
		t.Error("numeric id: expected player 3 (200) to equal 200")
	}
}

func TestAlwaysRules(t *testing.T) {
	ctx := &RuleContext{}

	trueRule := &AlwaysTrueRule{}
	if !trueRule.Evaluate(ctx) {
		t.Error("AlwaysTrueRule should always return true")
	}

	falseRule := &AlwaysFalseRule{}
	if falseRule.Evaluate(ctx) {
		t.Error("AlwaysFalseRule should always return false")
	}
}

func TestNestedLogicalRules(t *testing.T) {
	stats := NewStatisticsTracker()
	stats.Set(1, StatMoneyCurrent, 500.0)
	stats.Set(1, StatBuildingsPlaced, 5.0)
	stats.Set(1, StatDamageDealt, 100.0)

	ctx := &RuleContext{
		Stats: stats,
	}

	// (money >= 500 AND buildings >= 5) OR damage >= 200
	rule := &LogicalRule{
		Operator: "or",
		Rules: []Rule{
			&LogicalRule{
				Operator: "and",
				Rules: []Rule{
					&ComparisonRule{
						Left:     &StatValue{Stat: StatMoneyCurrent, PlayerID: 1},
						Operator: ">=",
						Right:    &ConstantValue{Value: 500.0},
					},
					&ComparisonRule{
						Left:     &StatValue{Stat: StatBuildingsPlaced, PlayerID: 1},
						Operator: ">=",
						Right:    &ConstantValue{Value: 5.0},
					},
				},
			},
			&ComparisonRule{
				Left:     &StatValue{Stat: StatDamageDealt, PlayerID: 1},
				Operator: ">=",
				Right:    &ConstantValue{Value: 200.0},
			},
		},
	}

	// First part (AND) is true, second part (damage) is false
	// Overall OR should be true
	if !rule.Evaluate(ctx) {
		t.Error("Expected nested rule to evaluate to true")
	}
}
