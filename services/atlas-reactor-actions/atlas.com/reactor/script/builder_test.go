package script

import (
	"testing"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
)

func TestReactorScriptBuilder_Build(t *testing.T) {
	tests := []struct {
		name        string
		reactorId   string
		description string
		hitRules    []Rule
		actRules    []Rule
	}{
		{
			name:        "empty script",
			reactorId:   "reactor_100",
			description: "Test reactor script",
			hitRules:    []Rule{},
			actRules:    []Rule{},
		},
		{
			name:        "script with hit rule",
			reactorId:   "reactor_200",
			description: "Script with a hit rule",
			hitRules: []Rule{
				NewRuleBuilder().SetId("rule1").Build(),
			},
			actRules: []Rule{},
		},
		{
			name:        "script with act rule",
			reactorId:   "reactor_300",
			description: "Script with an act rule",
			hitRules:    []Rule{},
			actRules: []Rule{
				NewRuleBuilder().SetId("rule1").Build(),
			},
		},
		{
			name:        "script with multiple rules",
			reactorId:   "reactor_400",
			description: "Script with multiple rules",
			hitRules: []Rule{
				NewRuleBuilder().SetId("hit_rule1").Build(),
				NewRuleBuilder().SetId("hit_rule2").Build(),
			},
			actRules: []Rule{
				NewRuleBuilder().SetId("act_rule1").Build(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewReactorScriptBuilder().
				SetReactorId(tt.reactorId).
				SetDescription(tt.description)

			for _, rule := range tt.hitRules {
				builder.AddHitRule(rule)
			}
			for _, rule := range tt.actRules {
				builder.AddActRule(rule)
			}

			script := builder.Build()

			if script.ReactorId() != tt.reactorId {
				t.Errorf("ReactorId() = %v, want %v", script.ReactorId(), tt.reactorId)
			}
			if script.Description() != tt.description {
				t.Errorf("Description() = %v, want %v", script.Description(), tt.description)
			}
			if len(script.HitRules()) != len(tt.hitRules) {
				t.Errorf("len(HitRules()) = %v, want %v", len(script.HitRules()), len(tt.hitRules))
			}
			if len(script.ActRules()) != len(tt.actRules) {
				t.Errorf("len(ActRules()) = %v, want %v", len(script.ActRules()), len(tt.actRules))
			}
		})
	}
}

func TestReactorScriptBuilder_FluentAPI(t *testing.T) {
	script := NewReactorScriptBuilder().
		SetReactorId("fluent_test").
		SetDescription("Fluent API test").
		AddHitRule(NewRuleBuilder().SetId("hit1").Build()).
		AddActRule(NewRuleBuilder().SetId("act1").Build()).
		Build()

	if script.ReactorId() != "fluent_test" {
		t.Error("Fluent API did not properly set reactor ID")
	}
	if len(script.HitRules()) != 1 {
		t.Error("Fluent API did not properly add hit rule")
	}
	if len(script.ActRules()) != 1 {
		t.Error("Fluent API did not properly add act rule")
	}
}

func TestRuleBuilder_Build(t *testing.T) {
	tests := []struct {
		name       string
		ruleId     string
		conditions int
		operations int
	}{
		{
			name:       "empty rule",
			ruleId:     "empty_rule",
			conditions: 0,
			operations: 0,
		},
		{
			name:       "rule with conditions",
			ruleId:     "rule_with_conditions",
			conditions: 2,
			operations: 0,
		},
		{
			name:       "rule with operations",
			ruleId:     "rule_with_operations",
			conditions: 0,
			operations: 3,
		},
		{
			name:       "rule with conditions and operations",
			ruleId:     "full_rule",
			conditions: 2,
			operations: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewRuleBuilder().SetId(tt.ruleId)

			for i := 0; i < tt.conditions; i++ {
				cond, _ := condition.NewBuilder().
					SetType("reactor_state").
					SetOperator("=").
					SetValue("0").
					Build()
				builder.AddCondition(cond)
			}

			for i := 0; i < tt.operations; i++ {
				op, _ := operation.NewBuilder().
					SetType("spawn_monster").
					SetParams(map[string]string{"monsterId": "100100"}).
					Build()
				builder.AddOperation(op)
			}

			rule := builder.Build()

			if rule.Id() != tt.ruleId {
				t.Errorf("Id() = %v, want %v", rule.Id(), tt.ruleId)
			}
			if len(rule.Conditions()) != tt.conditions {
				t.Errorf("len(Conditions()) = %v, want %v", len(rule.Conditions()), tt.conditions)
			}
			if len(rule.Operations()) != tt.operations {
				t.Errorf("len(Operations()) = %v, want %v", len(rule.Operations()), tt.operations)
			}
		})
	}
}

func TestNewReactorScriptBuilder_InitializesEmptyRules(t *testing.T) {
	builder := NewReactorScriptBuilder()
	script := builder.Build()

	if script.HitRules() == nil {
		t.Error("Expected HitRules() to be initialized, got nil")
	}
	if len(script.HitRules()) != 0 {
		t.Errorf("Expected empty hit rules slice, got %d rules", len(script.HitRules()))
	}
	if script.ActRules() == nil {
		t.Error("Expected ActRules() to be initialized, got nil")
	}
	if len(script.ActRules()) != 0 {
		t.Errorf("Expected empty act rules slice, got %d rules", len(script.ActRules()))
	}
}

func TestNewRuleBuilder_InitializesEmptySlices(t *testing.T) {
	builder := NewRuleBuilder()
	rule := builder.Build()

	if rule.Conditions() == nil {
		t.Error("Expected Conditions() to be initialized, got nil")
	}
	if len(rule.Conditions()) != 0 {
		t.Errorf("Expected empty conditions slice, got %d conditions", len(rule.Conditions()))
	}
	if rule.Operations() == nil {
		t.Error("Expected Operations() to be initialized, got nil")
	}
	if len(rule.Operations()) != 0 {
		t.Errorf("Expected empty operations slice, got %d operations", len(rule.Operations()))
	}
}
