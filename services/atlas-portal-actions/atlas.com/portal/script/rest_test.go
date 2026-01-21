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
		SetType("level").
		SetOperator(">=").
		SetValue("30").
		SetReferenceId("quest_123").
		Build()

	op, _ := operation.NewBuilder().
		SetType("warp").
		SetParams(map[string]string{"mapId": "100000000", "portalId": "0"}).
		Build()

	outcome := NewRuleOutcomeBuilder().
		SetAllow(true).
		AddOperation(op).
		Build()

	rule := NewRuleBuilder().
		SetId("test_rule").
		AddCondition(cond).
		SetOnMatch(outcome).
		Build()

	script := NewPortalScriptBuilder().
		SetPortalId("test_portal").
		SetMapId(200000000).
		SetDescription("Test script for transform").
		AddRule(rule).
		Build()

	rm, err := Transform(script)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	if rm.PortalId != "test_portal" {
		t.Errorf("Transform() PortalId = %v, want %v", rm.PortalId, "test_portal")
	}
	if rm.MapId != 200000000 {
		t.Errorf("Transform() MapId = %v, want %v", rm.MapId, 200000000)
	}
	if rm.Description != "Test script for transform" {
		t.Errorf("Transform() Description = %v, want %v", rm.Description, "Test script for transform")
	}
	if len(rm.Rules) != 1 {
		t.Fatalf("Transform() len(Rules) = %v, want 1", len(rm.Rules))
	}

	restRule := rm.Rules[0]
	if restRule.Id != "test_rule" {
		t.Errorf("Transform() rule Id = %v, want %v", restRule.Id, "test_rule")
	}
	if len(restRule.Conditions) != 1 {
		t.Fatalf("Transform() len(Conditions) = %v, want 1", len(restRule.Conditions))
	}
	if restRule.Conditions[0].Type != "level" {
		t.Errorf("Transform() condition Type = %v, want %v", restRule.Conditions[0].Type, "level")
	}
	if restRule.Conditions[0].ReferenceId != "quest_123" {
		t.Errorf("Transform() condition ReferenceId = %v, want %v", restRule.Conditions[0].ReferenceId, "quest_123")
	}
	if !restRule.OnMatch.Allow {
		t.Error("Transform() OnMatch.Allow = false, want true")
	}
	if len(restRule.OnMatch.Operations) != 1 {
		t.Fatalf("Transform() len(Operations) = %v, want 1", len(restRule.OnMatch.Operations))
	}
	if restRule.OnMatch.Operations[0].Type != "warp" {
		t.Errorf("Transform() operation Type = %v, want %v", restRule.OnMatch.Operations[0].Type, "warp")
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
				PortalId:    "test_portal",
				MapId:       100000000,
				Description: "Test description",
				Rules: []RestRuleModel{
					{
						Id: "rule1",
						Conditions: []RestConditionModel{
							{
								Type:     "level",
								Operator: ">=",
								Value:    "10",
							},
						},
						OnMatch: RestOutcomeModel{
							Allow: true,
							Operations: []RestOperationModel{
								{
									Type:   "warp",
									Params: map[string]string{"mapId": "200000000"},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty portalId",
			rm: RestModel{
				PortalId: "",
				MapId:    100000000,
			},
			wantErr: true,
		},
		{
			name: "model with no rules",
			rm: RestModel{
				PortalId:    "no_rules_portal",
				MapId:       100000000,
				Description: "No rules",
				Rules:       []RestRuleModel{},
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
				if script.PortalId() != tt.rm.PortalId {
					t.Errorf("Extract() PortalId = %v, want %v", script.PortalId(), tt.rm.PortalId)
				}
				if script.MapId() != tt.rm.MapId {
					t.Errorf("Extract() MapId = %v, want %v", script.MapId(), tt.rm.MapId)
				}
				if len(script.Rules()) != len(tt.rm.Rules) {
					t.Errorf("Extract() len(Rules) = %v, want %v", len(script.Rules()), len(tt.rm.Rules))
				}
			}
		})
	}
}

func TestTransformAndExtract_RoundTrip(t *testing.T) {
	cond, _ := condition.NewBuilder().
		SetType("job").
		SetOperator("==").
		SetValue("100").
		Build()

	op, _ := operation.NewBuilder().
		SetType("message").
		SetParams(map[string]string{"text": "Welcome!"}).
		Build()

	outcome := NewRuleOutcomeBuilder().
		SetAllow(true).
		AddOperation(op).
		Build()

	rule := NewRuleBuilder().
		SetId("roundtrip_rule").
		AddCondition(cond).
		SetOnMatch(outcome).
		Build()

	original := NewPortalScriptBuilder().
		SetPortalId("roundtrip_portal").
		SetMapId(300000000).
		SetDescription("Round trip test").
		AddRule(rule).
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
	if restored.PortalId() != original.PortalId() {
		t.Errorf("Round trip PortalId: got %v, want %v", restored.PortalId(), original.PortalId())
	}
	if restored.MapId() != original.MapId() {
		t.Errorf("Round trip MapId: got %v, want %v", restored.MapId(), original.MapId())
	}
	if restored.Description() != original.Description() {
		t.Errorf("Round trip Description: got %v, want %v", restored.Description(), original.Description())
	}
	if len(restored.Rules()) != len(original.Rules()) {
		t.Errorf("Round trip len(Rules): got %v, want %v", len(restored.Rules()), len(original.Rules()))
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
