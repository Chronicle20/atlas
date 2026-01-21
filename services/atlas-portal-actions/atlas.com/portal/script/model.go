package script

import (
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

// PortalScript represents a portal script loaded from JSON
type PortalScript struct {
	portalId    string
	mapId       uint32
	description string
	rules       []Rule
}

// PortalId returns the portal script identifier
func (s PortalScript) PortalId() string {
	return s.portalId
}

// MapId returns the map ID where this portal exists
func (s PortalScript) MapId() uint32 {
	return s.mapId
}

// Description returns the human-readable description
func (s PortalScript) Description() string {
	return s.description
}

// Rules returns the ordered list of rules
func (s PortalScript) Rules() []Rule {
	return s.rules
}

// Rule represents a single rule with conditions and outcome
type Rule struct {
	id         string
	conditions []condition.Model
	onMatch    RuleOutcome
}

// Id returns the rule identifier
func (r Rule) Id() string {
	return r.id
}

// Conditions returns the conditions that must all be true for this rule to match
func (r Rule) Conditions() []condition.Model {
	return r.conditions
}

// OnMatch returns the outcome when this rule matches
func (r Rule) OnMatch() RuleOutcome {
	return r.onMatch
}

// RuleOutcome represents what happens when a rule matches
type RuleOutcome struct {
	allow      bool
	operations []operation.Model
}

// Allow returns whether portal entry is allowed
func (o RuleOutcome) Allow() bool {
	return o.allow
}

// Operations returns the operations to execute
func (o RuleOutcome) Operations() []operation.Model {
	return o.operations
}

// ProcessResult represents the result of processing a portal script
type ProcessResult struct {
	Allow       bool
	MatchedRule string
	Operations  []operation.Model
	Error       error
}
