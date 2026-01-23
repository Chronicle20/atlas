package script

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
)

func TestMake(t *testing.T) {
	tests := []struct {
		name    string
		entity  Entity
		want    struct {
			reactorId   string
			description string
			hitRules    int
			actRules    int
		}
		wantErr bool
	}{
		{
			name: "valid entity with hit rules",
			entity: Entity{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				ReactorID: "reactor_100",
				Data: `{
					"reactorId": "reactor_100",
					"description": "Test reactor script",
					"hitRules": [
						{
							"id": "rule1",
							"conditions": [
								{"type": "reactor_state", "operator": "=", "value": "0"}
							],
							"operations": [
								{"type": "spawn_monster", "params": {"monsterId": "100100"}}
							]
						}
					],
					"actRules": []
				}`,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			want: struct {
				reactorId   string
				description string
				hitRules    int
				actRules    int
			}{
				reactorId:   "reactor_100",
				description: "Test reactor script",
				hitRules:    1,
				actRules:    0,
			},
			wantErr: false,
		},
		{
			name: "valid entity with act rules",
			entity: Entity{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				ReactorID: "reactor_200",
				Data: `{
					"reactorId": "reactor_200",
					"description": "Reactor with act rules",
					"hitRules": [],
					"actRules": [
						{
							"id": "act_rule1",
							"conditions": [],
							"operations": [
								{"type": "drop_items", "params": {"meso": "true"}}
							]
						}
					]
				}`,
			},
			want: struct {
				reactorId   string
				description string
				hitRules    int
				actRules    int
			}{
				reactorId:   "reactor_200",
				description: "Reactor with act rules",
				hitRules:    0,
				actRules:    1,
			},
			wantErr: false,
		},
		{
			name: "entity with empty rules",
			entity: Entity{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				ReactorID: "reactor_empty",
				Data: `{
					"reactorId": "reactor_empty",
					"description": "No rules",
					"hitRules": [],
					"actRules": []
				}`,
			},
			want: struct {
				reactorId   string
				description string
				hitRules    int
				actRules    int
			}{
				reactorId:   "reactor_empty",
				description: "No rules",
				hitRules:    0,
				actRules:    0,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			entity: Entity{
				ID:        uuid.New(),
				TenantID:  uuid.New(),
				ReactorID: "invalid",
				Data:      `{invalid json}`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script, err := Make(tt.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("Make() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if script.ReactorId() != tt.want.reactorId {
					t.Errorf("Make() ReactorId = %v, want %v", script.ReactorId(), tt.want.reactorId)
				}
				if script.Description() != tt.want.description {
					t.Errorf("Make() Description = %v, want %v", script.Description(), tt.want.description)
				}
				if len(script.HitRules()) != tt.want.hitRules {
					t.Errorf("Make() len(HitRules) = %v, want %v", len(script.HitRules()), tt.want.hitRules)
				}
				if len(script.ActRules()) != tt.want.actRules {
					t.Errorf("Make() len(ActRules) = %v, want %v", len(script.ActRules()), tt.want.actRules)
				}
			}
		})
	}
}

func TestToEntity(t *testing.T) {
	tenantId := uuid.New()

	cond, _ := condition.NewBuilder().
		SetType("reactor_state").
		SetOperator("=").
		SetValue("0").
		Build()

	op, _ := operation.NewBuilder().
		SetType("spawn_monster").
		SetParams(map[string]string{"monsterId": "100100", "count": "2"}).
		Build()

	rule := NewRuleBuilder().
		SetId("entity_test_rule").
		AddCondition(cond).
		AddOperation(op).
		Build()

	script := NewReactorScriptBuilder().
		SetReactorId("reactor_test").
		SetDescription("Entity test script").
		AddHitRule(rule).
		Build()

	entity, err := ToEntity(script, tenantId)
	if err != nil {
		t.Fatalf("ToEntity() error = %v", err)
	}

	if entity.TenantID != tenantId {
		t.Errorf("ToEntity() TenantID = %v, want %v", entity.TenantID, tenantId)
	}
	if entity.ReactorID != "reactor_test" {
		t.Errorf("ToEntity() ReactorID = %v, want %v", entity.ReactorID, "reactor_test")
	}

	// Verify Data is valid JSON
	var data jsonReactorScript
	if err := json.Unmarshal([]byte(entity.Data), &data); err != nil {
		t.Fatalf("ToEntity() Data is not valid JSON: %v", err)
	}

	if data.ReactorId != "reactor_test" {
		t.Errorf("ToEntity() Data.ReactorId = %v, want %v", data.ReactorId, "reactor_test")
	}
	if len(data.HitRules) != 1 {
		t.Fatalf("ToEntity() len(Data.HitRules) = %v, want 1", len(data.HitRules))
	}
	if data.HitRules[0].Id != "entity_test_rule" {
		t.Errorf("ToEntity() Data.HitRules[0].Id = %v, want %v", data.HitRules[0].Id, "entity_test_rule")
	}
}

func TestMakeAndToEntity_RoundTrip(t *testing.T) {
	tenantId := uuid.New()

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
		SetDescription("Round trip entity test").
		AddActRule(rule).
		Build()

	// Convert to entity
	entity, err := ToEntity(original, tenantId)
	if err != nil {
		t.Fatalf("ToEntity() error = %v", err)
	}

	// Convert back to model
	restored, err := Make(entity)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
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
		t.Fatalf("Round trip len(ActRules): got %v, want %v", len(restored.ActRules()), len(original.ActRules()))
	}

	// Verify rule details
	restoredRule := restored.ActRules()[0]
	originalRule := original.ActRules()[0]
	if restoredRule.Id() != originalRule.Id() {
		t.Errorf("Round trip Rule.Id: got %v, want %v", restoredRule.Id(), originalRule.Id())
	}
	if len(restoredRule.Conditions()) != len(originalRule.Conditions()) {
		t.Errorf("Round trip len(Conditions): got %v, want %v", len(restoredRule.Conditions()), len(originalRule.Conditions()))
	}
	if len(restoredRule.Operations()) != len(originalRule.Operations()) {
		t.Errorf("Round trip len(Operations): got %v, want %v", len(restoredRule.Operations()), len(originalRule.Operations()))
	}
}

func TestEntity_TableName(t *testing.T) {
	e := Entity{}
	if e.TableName() != "reactor_scripts" {
		t.Errorf("TableName() = %v, want %v", e.TableName(), "reactor_scripts")
	}
}

func TestMake_WithOperationParams(t *testing.T) {
	entity := Entity{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		ReactorID: "params_test",
		Data: `{
			"reactorId": "params_test",
			"description": "Test with operation params",
			"hitRules": [
				{
					"id": "rule1",
					"conditions": [],
					"operations": [
						{
							"type": "spawn_monster",
							"params": {"monsterId": "100100", "count": "5"}
						}
					]
				}
			],
			"actRules": []
		}`,
	}

	script, err := Make(entity)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if len(script.HitRules()) != 1 {
		t.Fatalf("Expected 1 hit rule, got %d", len(script.HitRules()))
	}

	ops := script.HitRules()[0].Operations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(ops))
	}

	if ops[0].Type() != "spawn_monster" {
		t.Errorf("Operation type = %v, want spawn_monster", ops[0].Type())
	}

	params := ops[0].Params()
	if params["monsterId"] != "100100" {
		t.Errorf("Operation param monsterId = %v, want 100100", params["monsterId"])
	}
	if params["count"] != "5" {
		t.Errorf("Operation param count = %v, want 5", params["count"])
	}
}

func TestMake_WithReferenceId(t *testing.T) {
	entity := Entity{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		ReactorID: "ref_test",
		Data: `{
			"reactorId": "ref_test",
			"description": "Test with referenceId",
			"hitRules": [
				{
					"id": "rule1",
					"conditions": [
						{
							"type": "reactor_state",
							"operator": "=",
							"value": "0",
							"referenceId": "reactor_ref_123"
						}
					],
					"operations": []
				}
			],
			"actRules": []
		}`,
	}

	script, err := Make(entity)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	conditions := script.HitRules()[0].Conditions()
	if len(conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(conditions))
	}

	if conditions[0].ReferenceIdRaw() != "reactor_ref_123" {
		t.Errorf("Condition referenceId = %v, want reactor_ref_123", conditions[0].ReferenceIdRaw())
	}
}

func TestMake_WithBothHitAndActRules(t *testing.T) {
	entity := Entity{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		ReactorID: "both_rules",
		Data: `{
			"reactorId": "both_rules",
			"description": "Test with both hit and act rules",
			"hitRules": [
				{
					"id": "hit_rule1",
					"conditions": [{"type": "reactor_state", "operator": "=", "value": "0"}],
					"operations": []
				},
				{
					"id": "hit_rule2",
					"conditions": [],
					"operations": []
				}
			],
			"actRules": [
				{
					"id": "act_rule1",
					"conditions": [],
					"operations": [{"type": "drop_items", "params": {"meso": "true"}}]
				}
			]
		}`,
	}

	script, err := Make(entity)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if len(script.HitRules()) != 2 {
		t.Errorf("Expected 2 hit rules, got %d", len(script.HitRules()))
	}
	if len(script.ActRules()) != 1 {
		t.Errorf("Expected 1 act rule, got %d", len(script.ActRules()))
	}

	if script.HitRules()[0].Id() != "hit_rule1" {
		t.Errorf("First hit rule id = %v, want hit_rule1", script.HitRules()[0].Id())
	}
	if script.HitRules()[1].Id() != "hit_rule2" {
		t.Errorf("Second hit rule id = %v, want hit_rule2", script.HitRules()[1].Id())
	}
	if script.ActRules()[0].Id() != "act_rule1" {
		t.Errorf("Act rule id = %v, want act_rule1", script.ActRules()[0].Id())
	}
}
