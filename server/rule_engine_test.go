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
	// When player is "any", it uses PlayerID 0
	stats.Set(0, StatMoneyCurrent, 600.0)

	ctx := &RuleContext{
		Stats: stats,
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
	// When player is "any", it uses PlayerID 0
	stats.Set(0, StatMoneyCurrent, 600.0)
	stats.Set(0, StatBuildingsPlaced, 5.0)

	ctx := &RuleContext{
		Stats: stats,
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
	// When player is "any", it uses PlayerID 0
	stats.Set(0, StatMoneyCurrent, 500.0)

	ctx := &RuleContext{
		Stats: stats,
		Variables: map[string]any{
			"resource_goal": 500.0,
		},
	}

	if !rule.Evaluate(ctx) {
		t.Error("Expected parsed parameter rule to evaluate to true")
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
