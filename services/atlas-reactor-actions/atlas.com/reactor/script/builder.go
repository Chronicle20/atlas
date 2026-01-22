package script

import (
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

// ReactorScriptBuilder builds a ReactorScript
type ReactorScriptBuilder struct {
	reactorId   string
	description string
	hitRules    []Rule
	actRules    []Rule
}

// NewReactorScriptBuilder creates a new builder
func NewReactorScriptBuilder() *ReactorScriptBuilder {
	return &ReactorScriptBuilder{
		hitRules: make([]Rule, 0),
		actRules: make([]Rule, 0),
	}
}

// SetReactorId sets the reactor ID
func (b *ReactorScriptBuilder) SetReactorId(reactorId string) *ReactorScriptBuilder {
	b.reactorId = reactorId
	return b
}

// SetDescription sets the description
func (b *ReactorScriptBuilder) SetDescription(description string) *ReactorScriptBuilder {
	b.description = description
	return b
}

// AddHitRule adds a hit rule to the script
func (b *ReactorScriptBuilder) AddHitRule(rule Rule) *ReactorScriptBuilder {
	b.hitRules = append(b.hitRules, rule)
	return b
}

// AddActRule adds an act rule to the script
func (b *ReactorScriptBuilder) AddActRule(rule Rule) *ReactorScriptBuilder {
	b.actRules = append(b.actRules, rule)
	return b
}

// Build builds the ReactorScript
func (b *ReactorScriptBuilder) Build() ReactorScript {
	return ReactorScript{
		reactorId:   b.reactorId,
		description: b.description,
		hitRules:    b.hitRules,
		actRules:    b.actRules,
	}
}

// RuleBuilder builds a Rule
type RuleBuilder struct {
	id         string
	conditions []condition.Model
	operations []operation.Model
}

// NewRuleBuilder creates a new rule builder
func NewRuleBuilder() *RuleBuilder {
	return &RuleBuilder{
		conditions: make([]condition.Model, 0),
		operations: make([]operation.Model, 0),
	}
}

// SetId sets the rule ID
func (b *RuleBuilder) SetId(id string) *RuleBuilder {
	b.id = id
	return b
}

// AddCondition adds a condition to the rule
func (b *RuleBuilder) AddCondition(cond condition.Model) *RuleBuilder {
	b.conditions = append(b.conditions, cond)
	return b
}

// AddOperation adds an operation to the rule
func (b *RuleBuilder) AddOperation(op operation.Model) *RuleBuilder {
	b.operations = append(b.operations, op)
	return b
}

// Build builds the Rule
func (b *RuleBuilder) Build() Rule {
	return Rule{
		id:         b.id,
		conditions: b.conditions,
		operations: b.operations,
	}
}
