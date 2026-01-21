package script

import (
	"testing"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

func TestPortalScriptBuilder_Build(t *testing.T) {
	tests := []struct {
		name        string
		portalId    string
		mapId       uint32
		description string
		rules       []Rule
	}{
		{
			name:        "empty script",
			portalId:    "test_portal",
			mapId:       100000000,
			description: "Test portal script",
			rules:       []Rule{},
		},
		{
			name:        "script with one rule",
			portalId:    "script_with_rule",
			mapId:       200000000,
			description: "Script with a rule",
			rules: []Rule{
				NewRuleBuilder().SetId("rule1").SetOnMatch(NewRuleOutcomeBuilder().SetAllow(true).Build()).Build(),
			},
		},
		{
			name:        "script with multiple rules",
			portalId:    "multi_rule_script",
			mapId:       300000000,
			description: "Script with multiple rules",
			rules: []Rule{
				NewRuleBuilder().SetId("rule1").SetOnMatch(NewRuleOutcomeBuilder().SetAllow(false).Build()).Build(),
				NewRuleBuilder().SetId("rule2").SetOnMatch(NewRuleOutcomeBuilder().SetAllow(true).Build()).Build(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewPortalScriptBuilder().
				SetPortalId(tt.portalId).
				SetMapId(tt.mapId).
				SetDescription(tt.description)

			for _, rule := range tt.rules {
				builder.AddRule(rule)
			}

			script := builder.Build()

			if script.PortalId() != tt.portalId {
				t.Errorf("PortalId() = %v, want %v", script.PortalId(), tt.portalId)
			}
			if script.MapId() != tt.mapId {
				t.Errorf("MapId() = %v, want %v", script.MapId(), tt.mapId)
			}
			if script.Description() != tt.description {
				t.Errorf("Description() = %v, want %v", script.Description(), tt.description)
			}
			if len(script.Rules()) != len(tt.rules) {
				t.Errorf("len(Rules()) = %v, want %v", len(script.Rules()), len(tt.rules))
			}
		})
	}
}

func TestPortalScriptBuilder_FluentAPI(t *testing.T) {
	script := NewPortalScriptBuilder().
		SetPortalId("fluent_test").
		SetMapId(100).
		SetDescription("Fluent API test").
		AddRule(NewRuleBuilder().SetId("rule1").Build()).
		Build()

	if script.PortalId() != "fluent_test" {
		t.Error("Fluent API did not properly set portal ID")
	}
	if len(script.Rules()) != 1 {
		t.Error("Fluent API did not properly add rule")
	}
}

func TestRuleBuilder_Build(t *testing.T) {
	tests := []struct {
		name       string
		ruleId     string
		conditions int
		allow      bool
	}{
		{
			name:       "empty rule",
			ruleId:     "empty_rule",
			conditions: 0,
			allow:      true,
		},
		{
			name:       "rule with conditions",
			ruleId:     "rule_with_conditions",
			conditions: 2,
			allow:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewRuleBuilder().SetId(tt.ruleId)

			for i := 0; i < tt.conditions; i++ {
				cond, _ := condition.NewBuilder().
					SetType("level").
					SetOperator(">=").
					SetValue("10").
					Build()
				builder.AddCondition(cond)
			}

			outcome := NewRuleOutcomeBuilder().SetAllow(tt.allow).Build()
			builder.SetOnMatch(outcome)

			rule := builder.Build()

			if rule.Id() != tt.ruleId {
				t.Errorf("Id() = %v, want %v", rule.Id(), tt.ruleId)
			}
			if len(rule.Conditions()) != tt.conditions {
				t.Errorf("len(Conditions()) = %v, want %v", len(rule.Conditions()), tt.conditions)
			}
			if rule.OnMatch().Allow() != tt.allow {
				t.Errorf("OnMatch().Allow() = %v, want %v", rule.OnMatch().Allow(), tt.allow)
			}
		})
	}
}

func TestRuleOutcomeBuilder_Build(t *testing.T) {
	tests := []struct {
		name       string
		allow      bool
		operations int
	}{
		{
			name:       "allow with no operations",
			allow:      true,
			operations: 0,
		},
		{
			name:       "deny with no operations",
			allow:      false,
			operations: 0,
		},
		{
			name:       "allow with operations",
			allow:      true,
			operations: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewRuleOutcomeBuilder().SetAllow(tt.allow)

			for i := 0; i < tt.operations; i++ {
				op, _ := operation.NewBuilder().
					SetType("warp").
					SetParams(map[string]string{"mapId": "100000000"}).
					Build()
				builder.AddOperation(op)
			}

			outcome := builder.Build()

			if outcome.Allow() != tt.allow {
				t.Errorf("Allow() = %v, want %v", outcome.Allow(), tt.allow)
			}
			if len(outcome.Operations()) != tt.operations {
				t.Errorf("len(Operations()) = %v, want %v", len(outcome.Operations()), tt.operations)
			}
		})
	}
}

func TestNewPortalScriptBuilder_InitializesEmptyRules(t *testing.T) {
	builder := NewPortalScriptBuilder()
	script := builder.Build()

	if script.Rules() == nil {
		t.Error("Expected Rules() to be initialized, got nil")
	}
	if len(script.Rules()) != 0 {
		t.Errorf("Expected empty rules slice, got %d rules", len(script.Rules()))
	}
}

func TestNewRuleBuilder_InitializesEmptyConditions(t *testing.T) {
	builder := NewRuleBuilder()
	rule := builder.Build()

	if rule.Conditions() == nil {
		t.Error("Expected Conditions() to be initialized, got nil")
	}
	if len(rule.Conditions()) != 0 {
		t.Errorf("Expected empty conditions slice, got %d conditions", len(rule.Conditions()))
	}
}

func TestNewRuleOutcomeBuilder_InitializesEmptyOperations(t *testing.T) {
	builder := NewRuleOutcomeBuilder()
	outcome := builder.Build()

	if outcome.Operations() == nil {
		t.Error("Expected Operations() to be initialized, got nil")
	}
	if len(outcome.Operations()) != 0 {
		t.Errorf("Expected empty operations slice, got %d operations", len(outcome.Operations()))
	}
}
