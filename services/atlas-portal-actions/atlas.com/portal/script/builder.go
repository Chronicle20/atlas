package script

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

// PortalScriptBuilder builds a PortalScript
type PortalScriptBuilder struct {
	portalId    string
	mapId       _map.Id
	description string
	rules       []Rule
}

// NewPortalScriptBuilder creates a new builder
func NewPortalScriptBuilder() *PortalScriptBuilder {
	return &PortalScriptBuilder{
		rules: make([]Rule, 0),
	}
}

// SetPortalId sets the portal ID
func (b *PortalScriptBuilder) SetPortalId(portalId string) *PortalScriptBuilder {
	b.portalId = portalId
	return b
}

// SetMapId sets the map ID
func (b *PortalScriptBuilder) SetMapId(mapId _map.Id) *PortalScriptBuilder {
	b.mapId = mapId
	return b
}

// SetDescription sets the description
func (b *PortalScriptBuilder) SetDescription(description string) *PortalScriptBuilder {
	b.description = description
	return b
}

// AddRule adds a rule to the script
func (b *PortalScriptBuilder) AddRule(rule Rule) *PortalScriptBuilder {
	b.rules = append(b.rules, rule)
	return b
}

// Build builds the PortalScript
func (b *PortalScriptBuilder) Build() PortalScript {
	return PortalScript{
		portalId:    b.portalId,
		mapId:       b.mapId,
		description: b.description,
		rules:       b.rules,
	}
}

// RuleBuilder builds a Rule
type RuleBuilder struct {
	id         string
	conditions []condition.Model
	onMatch    RuleOutcome
}

// NewRuleBuilder creates a new rule builder
func NewRuleBuilder() *RuleBuilder {
	return &RuleBuilder{
		conditions: make([]condition.Model, 0),
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

// SetOnMatch sets the outcome
func (b *RuleBuilder) SetOnMatch(outcome RuleOutcome) *RuleBuilder {
	b.onMatch = outcome
	return b
}

// Build builds the Rule
func (b *RuleBuilder) Build() Rule {
	return Rule{
		id:         b.id,
		conditions: b.conditions,
		onMatch:    b.onMatch,
	}
}

// RuleOutcomeBuilder builds a RuleOutcome
type RuleOutcomeBuilder struct {
	allow      bool
	operations []operation.Model
}

// NewRuleOutcomeBuilder creates a new outcome builder
func NewRuleOutcomeBuilder() *RuleOutcomeBuilder {
	return &RuleOutcomeBuilder{
		operations: make([]operation.Model, 0),
	}
}

// SetAllow sets whether portal entry is allowed
func (b *RuleOutcomeBuilder) SetAllow(allow bool) *RuleOutcomeBuilder {
	b.allow = allow
	return b
}

// AddOperation adds an operation to execute
func (b *RuleOutcomeBuilder) AddOperation(op operation.Model) *RuleOutcomeBuilder {
	b.operations = append(b.operations, op)
	return b
}

// Build builds the RuleOutcome
func (b *RuleOutcomeBuilder) Build() RuleOutcome {
	return RuleOutcome{
		allow:      b.allow,
		operations: b.operations,
	}
}
