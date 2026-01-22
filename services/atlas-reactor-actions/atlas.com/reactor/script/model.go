package script

import (
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

// ReactorScript represents a reactor script loaded from JSON
type ReactorScript struct {
	reactorId   string
	description string
	hitRules    []Rule
	actRules    []Rule
}

// ReactorId returns the reactor script identifier (classification ID)
func (s ReactorScript) ReactorId() string {
	return s.reactorId
}

// Description returns the human-readable description
func (s ReactorScript) Description() string {
	return s.description
}

// HitRules returns the rules evaluated when reactor is hit
func (s ReactorScript) HitRules() []Rule {
	return s.hitRules
}

// ActRules returns the rules evaluated when reactor triggers (reaches final state)
func (s ReactorScript) ActRules() []Rule {
	return s.actRules
}

// Rule represents a single rule with conditions and operations
type Rule struct {
	id         string
	conditions []condition.Model
	operations []operation.Model
}

// Id returns the rule identifier
func (r Rule) Id() string {
	return r.id
}

// Conditions returns the conditions that must all be true for this rule to match
func (r Rule) Conditions() []condition.Model {
	return r.conditions
}

// Operations returns the operations to execute when this rule matches
func (r Rule) Operations() []operation.Model {
	return r.operations
}

// ProcessResult represents the result of processing a reactor script
type ProcessResult struct {
	MatchedRule string
	Operations  []operation.Model
	Error       error
}
