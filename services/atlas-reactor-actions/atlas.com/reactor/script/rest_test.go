package script

import (
	"testing"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
)

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}
	if rm.GetName() != Resource {
		t.Errorf("GetName() = %v, want %v", rm.GetName(), Resource)
	}
}

func TestRestModel_GetID(t *testing.T) {
	id := uuid.New()
	rm := RestModel{Id: id}
	if rm.GetID() != id.String() {
		t.Errorf("GetID() = %v, want %v", rm.GetID(), id.String())
	}
}

func TestRestModel_SetID(t *testing.T) {
	tests := []struct {
		name    string
		idStr   string
		wantErr bool
	}{
		{
			name:    "valid UUID",
			idStr:   "550e8400-e29b-41d4-a716-446655440000",
			wantErr: false,
		},
		{
			name:    "invalid UUID",
			idStr:   "invalid-uuid",
			wantErr: true,
		},
		{
			name:    "empty string",
			idStr:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := &RestModel{}
			err := rm.SetID(tt.idStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && rm.Id.String() != tt.idStr {
				t.Errorf("SetID() did not set ID correctly, got %v, want %v", rm.Id.String(), tt.idStr)
			}
		})
	}
}

func TestTransform(t *testing.T) {
	cond, _ := condition.NewBuilder().
		SetType("reactor_state").
		SetOperator("=").
		SetValue("0").
		SetReferenceId("ref_123").
		Build()

	op, _ := operation.NewBuilder().
		SetType("spawn_monster").
		SetParams(map[string]string{"monsterId": "100100", "count": "3"}).
		Build()

	hitRule := NewRuleBuilder().
		SetId("hit_rule").
		AddCondition(cond).
		AddOperation(op).
		Build()

	actOp, _ := operation.NewBuilder().
		SetType("drop_items").
		SetParams(map[string]string{"meso": "true"}).
		Build()

	actRule := NewRuleBuilder().
		SetId("act_rule").
		AddOperation(actOp).
		Build()

	script := NewReactorScriptBuilder().
		SetReactorId("reactor_100").
		SetDescription("Test script for transform").
		AddHitRule(hitRule).
		AddActRule(actRule).
		Build()

	rm, err := Transform(script)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	if rm.ReactorId != "reactor_100" {
		t.Errorf("Transform() ReactorId = %v, want %v", rm.ReactorId, "reactor_100")
	}
	if rm.Description != "Test script for transform" {
		t.Errorf("Transform() Description = %v, want %v", rm.Description, "Test script for transform")
	}
	if len(rm.HitRules) != 1 {
		t.Fatalf("Transform() len(HitRules) = %v, want 1", len(rm.HitRules))
	}
	if len(rm.ActRules) != 1 {
		t.Fatalf("Transform() len(ActRules) = %v, want 1", len(rm.ActRules))
	}

	restHitRule := rm.HitRules[0]
	if restHitRule.Id != "hit_rule" {
		t.Errorf("Transform() hit rule Id = %v, want %v", restHitRule.Id, "hit_rule")
	}
	if len(restHitRule.Conditions) != 1 {
		t.Fatalf("Transform() len(Conditions) = %v, want 1", len(restHitRule.Conditions))
	}
	if restHitRule.Conditions[0].Type != "reactor_state" {
		t.Errorf("Transform() condition Type = %v, want %v", restHitRule.Conditions[0].Type, "reactor_state")
	}
	if restHitRule.Conditions[0].ReferenceId != "ref_123" {
		t.Errorf("Transform() condition ReferenceId = %v, want %v", restHitRule.Conditions[0].ReferenceId, "ref_123")
	}
	if len(restHitRule.Operations) != 1 {
		t.Fatalf("Transform() len(Operations) = %v, want 1", len(restHitRule.Operations))
	}
	if restHitRule.Operations[0].Type != "spawn_monster" {
		t.Errorf("Transform() operation Type = %v, want %v", restHitRule.Operations[0].Type, "spawn_monster")
	}

	restActRule := rm.ActRules[0]
	if restActRule.Id != "act_rule" {
		t.Errorf("Transform() act rule Id = %v, want %v", restActRule.Id, "act_rule")
	}
}

func TestExtract(t *testing.T) {
	tests := []struct {
		name    string
		rm      RestModel
		wantErr bool
	}{
		{
			name: "valid model",
			rm: RestModel{
				ReactorId:   "reactor_100",
				Description: "Test description",
				HitRules: []RestRuleModel{
					{
						Id: "rule1",
						Conditions: []RestConditionModel{
							{
								Type:     "reactor_state",
								Operator: "=",
								Value:    "0",
							},
						},
						Operations: []RestOperationModel{
							{
								Type:   "spawn_monster",
								Params: map[string]string{"monsterId": "100100"},
							},
						},
					},
				},
				ActRules: []RestRuleModel{},
			},
			wantErr: false,
		},
		{
			name: "empty reactorId",
			rm: RestModel{
				ReactorId: "",
			},
			wantErr: true,
		},
		{
			name: "model with no rules",
			rm: RestModel{
				ReactorId:   "no_rules_reactor",
				Description: "No rules",
				HitRules:    []RestRuleModel{},
				ActRules:    []RestRuleModel{},
			},
			wantErr: false,
		},
		{
			name: "model with act rules only",
			rm: RestModel{
				ReactorId:   "act_only_reactor",
				Description: "Act rules only",
				HitRules:    []RestRuleModel{},
				ActRules: []RestRuleModel{
					{
						Id:         "act_rule1",
						Conditions: []RestConditionModel{},
						Operations: []RestOperationModel{
							{
								Type:   "drop_items",
								Params: map[string]string{"meso": "true"},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := Extract(tt.rm)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if script.ReactorId() != tt.rm.ReactorId {
					t.Errorf("Extract() ReactorId = %v, want %v", script.ReactorId(), tt.rm.ReactorId)
				}
				if script.Description() != tt.rm.Description {
					t.Errorf("Extract() Description = %v, want %v", script.Description(), tt.rm.Description)
				}
				if len(script.HitRules()) != len(tt.rm.HitRules) {
					t.Errorf("Extract() len(HitRules) = %v, want %v", len(script.HitRules()), len(tt.rm.HitRules))
				}
				if len(script.ActRules()) != len(tt.rm.ActRules) {
					t.Errorf("Extract() len(ActRules) = %v, want %v", len(script.ActRules()), len(tt.rm.ActRules))
				}
			}
		})
	}
}

func TestTransformAndExtract_RoundTrip(t *testing.T) {
	cond, _ := condition.NewBuilder().
		SetType("reactor_state").
		SetOperator(">=").
		SetValue("2").
		Build()

	op, _ := operation.NewBuilder().
		SetType("drop_message").
		SetParams(map[string]string{"message": "Hello!", "type": "PINK_TEXT"}).
		Build()

	rule := NewRuleBuilder().
		SetId("roundtrip_rule").
		AddCondition(cond).
		AddOperation(op).
		Build()

	original := NewReactorScriptBuilder().
		SetReactorId("roundtrip_reactor").
		SetDescription("Round trip test").
		AddActRule(rule).
		Build()

	// Transform to REST
	rm, err := Transform(original)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Extract back to domain
	restored, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	// Verify fields match
	if restored.ReactorId() != original.ReactorId() {
		t.Errorf("Round trip ReactorId: got %v, want %v", restored.ReactorId(), original.ReactorId())
	}
	if restored.Description() != original.Description() {
		t.Errorf("Round trip Description: got %v, want %v", restored.Description(), original.Description())
	}
	if len(restored.HitRules()) != len(original.HitRules()) {
		t.Errorf("Round trip len(HitRules): got %v, want %v", len(restored.HitRules()), len(original.HitRules()))
	}
	if len(restored.ActRules()) != len(original.ActRules()) {
		t.Errorf("Round trip len(ActRules): got %v, want %v", len(restored.ActRules()), len(original.ActRules()))
	}
}

func TestRestModel_References(t *testing.T) {
	rm := RestModel{}

	if len(rm.GetReferences()) != 0 {
		t.Error("GetReferences() should return empty slice")
	}
	if len(rm.GetReferencedIDs()) != 0 {
		t.Error("GetReferencedIDs() should return empty slice")
	}
	if len(rm.GetReferencedStructs()) != 0 {
		t.Error("GetReferencedStructs() should return empty slice")
	}

	// These methods are no-ops but shouldn't error
	if err := rm.SetToOneReferenceID("test", "id"); err != nil {
		t.Errorf("SetToOneReferenceID() unexpected error: %v", err)
	}
	if err := rm.SetToManyReferenceIDs("test", []string{"id1", "id2"}); err != nil {
		t.Errorf("SetToManyReferenceIDs() unexpected error: %v", err)
	}
	if err := rm.SetReferencedStructs(nil); err != nil {
		t.Errorf("SetReferencedStructs() unexpected error: %v", err)
	}
}

func TestTransform_EmptyScript(t *testing.T) {
	script := NewReactorScriptBuilder().
		SetReactorId("empty_reactor").
		SetDescription("Empty script").
		Build()

	rm, err := Transform(script)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	if rm.ReactorId != "empty_reactor" {
		t.Errorf("ReactorId = %v, want empty_reactor", rm.ReactorId)
	}
	if len(rm.HitRules) != 0 {
		t.Errorf("Expected 0 hit rules, got %d", len(rm.HitRules))
	}
	if len(rm.ActRules) != 0 {
		t.Errorf("Expected 0 act rules, got %d", len(rm.ActRules))
	}
}

func TestExtract_WithReferenceId(t *testing.T) {
	rm := RestModel{
		ReactorId:   "ref_reactor",
		Description: "Test with referenceId",
		HitRules: []RestRuleModel{
			{
				Id: "rule1",
				Conditions: []RestConditionModel{
					{
						Type:        "reactor_state",
						Operator:    "=",
						Value:       "0",
						ReferenceId: "ref_456",
					},
				},
				Operations: []RestOperationModel{},
			},
		},
		ActRules: []RestRuleModel{},
	}

	script, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	conditions := script.HitRules()[0].Conditions()
	if len(conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(conditions))
	}

	if conditions[0].ReferenceIdRaw() != "ref_456" {
		t.Errorf("Condition referenceId = %v, want ref_456", conditions[0].ReferenceIdRaw())
	}
}

func TestExtract_WithOperationParams(t *testing.T) {
	rm := RestModel{
		ReactorId:   "params_reactor",
		Description: "Test with operation params",
		HitRules:    []RestRuleModel{},
		ActRules: []RestRuleModel{
			{
				Id:         "act_rule1",
				Conditions: []RestConditionModel{},
				Operations: []RestOperationModel{
					{
						Type: "spawn_monster",
						Params: map[string]string{
							"monsterId": "100100",
							"count":     "5",
						},
					},
				},
			},
		},
	}

	script, err := Extract(rm)
	if err != nil {
		t.Fatalf("Extract() error = %v", err)
	}

	ops := script.ActRules()[0].Operations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(ops))
	}

	params := ops[0].Params()
	if params["monsterId"] != "100100" {
		t.Errorf("Operation param monsterId = %v, want 100100", params["monsterId"])
	}
	if params["count"] != "5" {
		t.Errorf("Operation param count = %v, want 5", params["count"])
	}
}
