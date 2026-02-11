package script

import (
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

// MapScriptBuilder builds a MapScript
type MapScriptBuilder struct {
	scriptName  string
	scriptType  string
	description string
	rules       []Rule
}

// NewMapScriptBuilder creates a new builder
func NewMapScriptBuilder() *MapScriptBuilder {
	return &MapScriptBuilder{
		rules: make([]Rule, 0),
	}
}

// SetScriptName sets the script name
func (b *MapScriptBuilder) SetScriptName(scriptName string) *MapScriptBuilder {
	b.scriptName = scriptName
	return b
}

// SetScriptType sets the script type
func (b *MapScriptBuilder) SetScriptType(scriptType string) *MapScriptBuilder {
	b.scriptType = scriptType
	return b
}

// SetDescription sets the description
func (b *MapScriptBuilder) SetDescription(description string) *MapScriptBuilder {
	b.description = description
	return b
}

// AddRule adds a rule to the script
func (b *MapScriptBuilder) AddRule(rule Rule) *MapScriptBuilder {
	b.rules = append(b.rules, rule)
	return b
}

// Build builds the MapScript
func (b *MapScriptBuilder) Build() MapScript {
	return MapScript{
		scriptName:  b.scriptName,
		scriptType:  b.scriptType,
		description: b.description,
		rules:       b.rules,
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
