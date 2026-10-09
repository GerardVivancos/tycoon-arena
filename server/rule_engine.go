package main

import (
	"encoding/json"
	"fmt"
)

// Rule represents a declarative condition that can be evaluated
type Rule interface {
	Evaluate(ctx *RuleContext) bool
}

// RuleContext provides data access for rule evaluation
type RuleContext struct {
	Server    *GameServer
	Stats     *StatisticsTracker
	EventBus  *EventBus
	Tick      uint64
	Variables map[string]any // For parameterized rules
}

// ValueExpression represents a value that can be resolved at runtime
type ValueExpression interface {
	Resolve(ctx *RuleContext) float64
}

// ===== Value Expressions =====

// ConstantValue represents a constant numeric value
type ConstantValue struct {
	Value float64
}

func (v *ConstantValue) Resolve(ctx *RuleContext) float64 {
	return v.Value
}

// StatValue represents a player statistic
type StatValue struct {
	Stat     string
	PlayerID uint32 // 0 means "any", specific ID, or special values
	Global   bool   // If true, this is a global stat
}

func (v *StatValue) Resolve(ctx *RuleContext) float64 {
	if v.Global {
		return ctx.Stats.GetGlobal(v.Stat)
	}
	return ctx.Stats.Get(v.PlayerID, v.Stat)
}

// ParameterValue represents a parameter from game mode definition
type ParameterValue struct {
	ParamName string
}

func (v *ParameterValue) Resolve(ctx *RuleContext) float64 {
	if val, ok := ctx.Variables[v.ParamName]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int32:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0.0
}

// ===== Rule Types =====

// ComparisonRule compares two values
type ComparisonRule struct {
	Left     ValueExpression
	Operator string // "==", "!=", "<", ">", "<=", ">="
	Right    ValueExpression
}

func (r *ComparisonRule) Evaluate(ctx *RuleContext) bool {
	leftVal := r.Left.Resolve(ctx)
	rightVal := r.Right.Resolve(ctx)

	switch r.Operator {
	case "==":
		return leftVal == rightVal
	case "!=":
		return leftVal != rightVal
	case "<":
		return leftVal < rightVal
	case ">":
		return leftVal > rightVal
	case "<=":
		return leftVal <= rightVal
	case ">=":
		return leftVal >= rightVal
	default:
		return false
	}
}

// LogicalRule combines multiple rules with AND/OR/NOT
type LogicalRule struct {
	Operator string // "and", "or", "not"
	Rules    []Rule
}

func (r *LogicalRule) Evaluate(ctx *RuleContext) bool {
	switch r.Operator {
	case "and":
		for _, rule := range r.Rules {
			if !rule.Evaluate(ctx) {
				return false
			}
		}
		return true

	case "or":
		for _, rule := range r.Rules {
			if rule.Evaluate(ctx) {
				return true
			}
		}
		return false

	case "not":
		if len(r.Rules) > 0 {
			return !r.Rules[0].Evaluate(ctx)
		}
		return false

	default:
		return false
	}
}

// AlwaysTrueRule always evaluates to true (for testing)
type AlwaysTrueRule struct{}

func (r *AlwaysTrueRule) Evaluate(ctx *RuleContext) bool {
	return true
}

// AlwaysFalseRule always evaluates to false (for testing)
type AlwaysFalseRule struct{}

func (r *AlwaysFalseRule) Evaluate(ctx *RuleContext) bool {
	return false
}

// ===== JSON Rule Parsing =====

// RuleJSON represents a rule in JSON format
type RuleJSON struct {
	Type     string         `json:"type"`     // "comparison", "and", "or", "not", "always_true", "always_false"
	Left     *ValueJSON     `json:"left"`     // For comparison rules
	Operator string         `json:"operator"` // For comparison rules
	Right    *ValueJSON     `json:"right"`    // For comparison rules
	Rules    []RuleJSON     `json:"rules"`    // For logical rules
	Ref      string         `json:"ref"`      // Reference to another rule by ID
	Data     map[string]any `json:"data"`     // Additional data
}

// ValueJSON represents a value expression in JSON
type ValueJSON struct {
	// Exactly one of these should be set
	Value  *float64 `json:"value"`  // Constant value
	Stat   string   `json:"stat"`   // Stat name
	Player string   `json:"player"` // "any", "self", "opponent", or player ID
	Global bool     `json:"global"` // Is this a global stat?
	Param  string   `json:"param"`  // Parameter name
}

// ParseRule parses a JSON rule into a Rule interface
func ParseRule(ruleJSON RuleJSON) (Rule, error) {
	switch ruleJSON.Type {
	case "comparison":
		if ruleJSON.Left == nil || ruleJSON.Right == nil {
			return nil, fmt.Errorf("comparison rule missing left or right value")
		}

		left, err := ParseValue(*ruleJSON.Left)
		if err != nil {
			return nil, fmt.Errorf("failed to parse left value: %w", err)
		}

		right, err := ParseValue(*ruleJSON.Right)
		if err != nil {
			return nil, fmt.Errorf("failed to parse right value: %w", err)
		}

		return &ComparisonRule{
			Left:     left,
			Operator: ruleJSON.Operator,
			Right:    right,
		}, nil

	case "and", "or", "not":
		rules := make([]Rule, 0, len(ruleJSON.Rules))
		for _, subRuleJSON := range ruleJSON.Rules {
			subRule, err := ParseRule(subRuleJSON)
			if err != nil {
				return nil, fmt.Errorf("failed to parse sub-rule: %w", err)
			}
			rules = append(rules, subRule)
		}

		return &LogicalRule{
			Operator: ruleJSON.Type,
			Rules:    rules,
		}, nil

	case "always_true":
		return &AlwaysTrueRule{}, nil

	case "always_false":
		return &AlwaysFalseRule{}, nil

	default:
		return nil, fmt.Errorf("unknown rule type: %s", ruleJSON.Type)
	}
}

// ParseValue parses a JSON value expression into a ValueExpression
func ParseValue(valueJSON ValueJSON) (ValueExpression, error) {
	// Constant value
	if valueJSON.Value != nil {
		return &ConstantValue{Value: *valueJSON.Value}, nil
	}

	// Parameter reference
	if valueJSON.Param != "" {
		return &ParameterValue{ParamName: valueJSON.Param}, nil
	}

	// Stat value
	if valueJSON.Stat != "" {
		playerID := uint32(0) // Default to "any"
		if valueJSON.Player != "" && valueJSON.Player != "any" {
			// For now, we'll use 0 to represent "any"
			// In a real implementation, this would resolve based on context
			playerID = 0
		}

		return &StatValue{
			Stat:     valueJSON.Stat,
			PlayerID: playerID,
			Global:   valueJSON.Global,
		}, nil
	}

	return nil, fmt.Errorf("value expression missing all fields")
}

// ParseRuleFromJSON parses a rule from a JSON string
func ParseRuleFromJSON(jsonStr string) (Rule, error) {
	var ruleJSON RuleJSON
	if err := json.Unmarshal([]byte(jsonStr), &ruleJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return ParseRule(ruleJSON)
}
