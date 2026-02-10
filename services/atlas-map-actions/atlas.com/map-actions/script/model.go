package script

import (
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

// MapScript represents a map script loaded from JSON
type MapScript struct {
	scriptName  string
	scriptType  string
	description string
	rules       []Rule
}

// ScriptName returns the script name identifier
func (s MapScript) ScriptName() string {
	return s.scriptName
}

// ScriptType returns the script type (onFirstUserEnter or onUserEnter)
func (s MapScript) ScriptType() string {
	return s.scriptType
}

// Description returns the human-readable description
func (s MapScript) Description() string {
	return s.description
}

// Rules returns the rules for this script
func (s MapScript) Rules() []Rule {
	return s.rules
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

// ProcessResult represents the result of processing a map script
type ProcessResult struct {
	MatchedRule string
	Operations  []operation.Model
	Error       error
}
