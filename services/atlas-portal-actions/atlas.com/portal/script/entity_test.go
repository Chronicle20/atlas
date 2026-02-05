package script

import (
	"encoding/json"
	"testing"
	"time"

	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
)

func TestMake(t *testing.T) {
	tests := []struct {
		name   string
		entity Entity
		want   struct {
			portalId    string
			mapId       _map.Id
			description string
			ruleCount   int
		}
		wantErr bool
	}{
		{
			name: "valid entity with rules",
			entity: Entity{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				PortalID: "test_portal",
				MapID:    100000000,
				Data: `{
					"portalId": "test_portal",
					"mapId": 100000000,
					"description": "Test portal script",
					"rules": [
						{
							"id": "rule1",
							"conditions": [
								{"type": "level", "operator": ">=", "value": "10"}
							],
							"onMatch": {
								"allow": true,
								"operations": []
							}
						}
					]
				}`,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			want: struct {
				portalId    string
				mapId       _map.Id
				description string
				ruleCount   int
			}{
				portalId:    "test_portal",
				mapId:       100000000,
				description: "Test portal script",
				ruleCount:   1,
			},
			wantErr: false,
		},
		{
			name: "entity with empty rules",
			entity: Entity{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				PortalID: "empty_rules",
				MapID:    200000000,
				Data: `{
					"portalId": "empty_rules",
					"mapId": 200000000,
					"description": "No rules",
					"rules": []
				}`,
			},
			want: struct {
				portalId    string
				mapId       _map.Id
				description string
				ruleCount   int
			}{
				portalId:    "empty_rules",
				mapId:       200000000,
				description: "No rules",
				ruleCount:   0,
			},
			wantErr: false,
		},
		{
			name: "invalid JSON",
			entity: Entity{
				ID:       uuid.New(),
				TenantID: uuid.New(),
				PortalID: "invalid",
				MapID:    100000000,
				Data:     `{invalid json}`,
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
				if script.PortalId() != tt.want.portalId {
					t.Errorf("Make() PortalId = %v, want %v", script.PortalId(), tt.want.portalId)
				}
				if script.MapId() != tt.want.mapId {
					t.Errorf("Make() MapId = %v, want %v", script.MapId(), tt.want.mapId)
				}
				if script.Description() != tt.want.description {
					t.Errorf("Make() Description = %v, want %v", script.Description(), tt.want.description)
				}
				if len(script.Rules()) != tt.want.ruleCount {
					t.Errorf("Make() len(Rules) = %v, want %v", len(script.Rules()), tt.want.ruleCount)
				}
			}
		})
	}
}

func TestToEntity(t *testing.T) {
	tenantId := uuid.New()

	cond, _ := condition.NewBuilder().
		SetType("level").
		SetOperator(">=").
		SetValue("20").
		SetReferenceId("quest_456").
		Build()

	op, _ := operation.NewBuilder().
		SetType("warp").
		SetParams(map[string]string{"mapId": "300000000"}).
		Build()

	outcome := NewRuleOutcomeBuilder().
		SetAllow(true).
		AddOperation(op).
		Build()

	rule := NewRuleBuilder().
		SetId("entity_test_rule").
		AddCondition(cond).
		SetOnMatch(outcome).
		Build()

	script := NewPortalScriptBuilder().
		SetPortalId("entity_test_portal").
		SetMapId(400000000).
		SetDescription("Entity test script").
		AddRule(rule).
		Build()

	entity, err := ToEntity(script, tenantId)
	if err != nil {
		t.Fatalf("ToEntity() error = %v", err)
	}

	if entity.TenantID != tenantId {
		t.Errorf("ToEntity() TenantID = %v, want %v", entity.TenantID, tenantId)
	}
	if entity.PortalID != "entity_test_portal" {
		t.Errorf("ToEntity() PortalID = %v, want %v", entity.PortalID, "entity_test_portal")
	}
	if entity.MapID != 400000000 {
		t.Errorf("ToEntity() MapID = %v, want %v", entity.MapID, 400000000)
	}

	// Verify Data is valid JSON
	var data jsonPortalScript
	if err := json.Unmarshal([]byte(entity.Data), &data); err != nil {
		t.Fatalf("ToEntity() Data is not valid JSON: %v", err)
	}

	if data.PortalId != "entity_test_portal" {
		t.Errorf("ToEntity() Data.PortalId = %v, want %v", data.PortalId, "entity_test_portal")
	}
	if len(data.Rules) != 1 {
		t.Fatalf("ToEntity() len(Data.Rules) = %v, want 1", len(data.Rules))
	}
	if data.Rules[0].Id != "entity_test_rule" {
		t.Errorf("ToEntity() Data.Rules[0].Id = %v, want %v", data.Rules[0].Id, "entity_test_rule")
	}
}

func TestMakeAndToEntity_RoundTrip(t *testing.T) {
	tenantId := uuid.New()

	cond, _ := condition.NewBuilder().
		SetType("job").
		SetOperator("==").
		SetValue("200").
		Build()

	op, _ := operation.NewBuilder().
		SetType("enable_actions").
		Build()

	outcome := NewRuleOutcomeBuilder().
		SetAllow(false).
		AddOperation(op).
		Build()

	rule := NewRuleBuilder().
		SetId("roundtrip_entity_rule").
		AddCondition(cond).
		SetOnMatch(outcome).
		Build()

	original := NewPortalScriptBuilder().
		SetPortalId("roundtrip_entity_portal").
		SetMapId(500000000).
		SetDescription("Round trip entity test").
		AddRule(rule).
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
		t.Fatalf("Round trip len(Rules): got %v, want %v", len(restored.Rules()), len(original.Rules()))
	}

	// Verify rule details
	restoredRule := restored.Rules()[0]
	originalRule := original.Rules()[0]
	if restoredRule.Id() != originalRule.Id() {
		t.Errorf("Round trip Rule.Id: got %v, want %v", restoredRule.Id(), originalRule.Id())
	}
	if restoredRule.OnMatch().Allow() != originalRule.OnMatch().Allow() {
		t.Errorf("Round trip Rule.OnMatch.Allow: got %v, want %v", restoredRule.OnMatch().Allow(), originalRule.OnMatch().Allow())
	}
}

func TestEntity_TableName(t *testing.T) {
	e := Entity{}
	if e.TableName() != "portal_scripts" {
		t.Errorf("TableName() = %v, want %v", e.TableName(), "portal_scripts")
	}
}

func TestMake_WithOperationParams(t *testing.T) {
	entity := Entity{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		PortalID: "params_test",
		MapID:    100000000,
		Data: `{
			"portalId": "params_test",
			"mapId": 100000000,
			"description": "Test with operation params",
			"rules": [
				{
					"id": "rule1",
					"conditions": [],
					"onMatch": {
						"allow": true,
						"operations": [
							{
								"type": "warp",
								"params": {"mapId": "200000000", "portalId": "5"}
							}
						]
					}
				}
			]
		}`,
	}

	script, err := Make(entity)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	if len(script.Rules()) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(script.Rules()))
	}

	ops := script.Rules()[0].OnMatch().Operations()
	if len(ops) != 1 {
		t.Fatalf("Expected 1 operation, got %d", len(ops))
	}

	if ops[0].Type() != "warp" {
		t.Errorf("Operation type = %v, want warp", ops[0].Type())
	}

	params := ops[0].Params()
	if params["mapId"] != "200000000" {
		t.Errorf("Operation param mapId = %v, want 200000000", params["mapId"])
	}
}

func TestMake_WithReferenceId(t *testing.T) {
	entity := Entity{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		PortalID: "ref_test",
		MapID:    100000000,
		Data: `{
			"portalId": "ref_test",
			"mapId": 100000000,
			"description": "Test with referenceId",
			"rules": [
				{
					"id": "rule1",
					"conditions": [
						{
							"type": "quest_state",
							"operator": "==",
							"value": "COMPLETED",
							"referenceId": "quest_2001"
						}
					],
					"onMatch": {
						"allow": true,
						"operations": []
					}
				}
			]
		}`,
	}

	script, err := Make(entity)
	if err != nil {
		t.Fatalf("Make() error = %v", err)
	}

	conditions := script.Rules()[0].Conditions()
	if len(conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(conditions))
	}

	if conditions[0].ReferenceIdRaw() != "quest_2001" {
		t.Errorf("Condition referenceId = %v, want quest_2001", conditions[0].ReferenceIdRaw())
	}
}
