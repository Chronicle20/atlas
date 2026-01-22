package script

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a portal script stored in the database
type Entity struct {
	ID        uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID  uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;index:idx_portal_scripts_tenant_portal,priority:1"`
	PortalID  string         `gorm:"column:portal_id;not null;index:idx_portal_scripts_tenant_portal,priority:2"`
	MapID     uint32         `gorm:"column:map_id;index"`
	Data      string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the entity
func (Entity) TableName() string {
	return "portal_scripts"
}

// Make converts an Entity to a PortalScript model
func Make(e Entity) (PortalScript, error) {
	var data jsonPortalScript
	if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
		return PortalScript{}, err
	}

	builder := NewPortalScriptBuilder().
		SetPortalId(data.PortalId).
		SetMapId(data.MapId).
		SetDescription(data.Description)

	for _, jr := range data.Rules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return PortalScript{}, err
		}
		builder.AddRule(rule)
	}

	return builder.Build(), nil
}

// convertJsonRule converts a JSON rule to domain model
func convertJsonRule(jr jsonRule) (Rule, error) {
	rb := NewRuleBuilder().SetId(jr.Id)

	for _, jc := range jr.Conditions {
		cond, err := convertJsonCondition(jc)
		if err != nil {
			return Rule{}, err
		}
		rb.AddCondition(cond)
	}

	outcome, err := convertJsonOutcome(jr.OnMatch)
	if err != nil {
		return Rule{}, err
	}
	rb.SetOnMatch(outcome)

	return rb.Build(), nil
}

// convertJsonCondition converts a JSON condition to domain model
func convertJsonCondition(jc jsonCondition) (condition.Model, error) {
	builder := condition.NewBuilder().
		SetType(jc.Type).
		SetOperator(jc.Operator).
		SetValue(jc.Value)

	if jc.ReferenceId != "" {
		builder.SetReferenceId(jc.ReferenceId)
	}

	return builder.Build()
}

// convertJsonOutcome converts a JSON outcome to domain model
func convertJsonOutcome(jo jsonOutcome) (RuleOutcome, error) {
	ob := NewRuleOutcomeBuilder().SetAllow(jo.Allow)

	for _, jop := range jo.Operations {
		op, err := convertJsonOperation(jop)
		if err != nil {
			return RuleOutcome{}, err
		}
		ob.AddOperation(op)
	}

	return ob.Build(), nil
}

// convertJsonOperation converts a JSON operation to domain model
func convertJsonOperation(jo jsonOperation) (operation.Model, error) {
	builder := operation.NewBuilder().SetType(jo.Type)

	if jo.Params != nil {
		builder.SetParams(jo.Params)
	}

	return builder.Build()
}

// ToEntity converts a PortalScript model to an Entity
func ToEntity(m PortalScript, tenantId uuid.UUID) (Entity, error) {
	// Convert rules to JSON format
	jsonRules := make([]jsonRule, 0, len(m.Rules()))
	for _, rule := range m.Rules() {
		jr := convertRuleToJson(rule)
		jsonRules = append(jsonRules, jr)
	}

	data := jsonPortalScript{
		PortalId:    m.PortalId(),
		MapId:       m.MapId(),
		Description: m.Description(),
		Rules:       jsonRules,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return Entity{}, err
	}

	return Entity{
		TenantID: tenantId,
		PortalID: m.PortalId(),
		MapID:    m.MapId(),
		Data:     string(jsonData),
	}, nil
}

// convertRuleToJson converts a domain Rule to JSON format
func convertRuleToJson(rule Rule) jsonRule {
	conditions := make([]jsonCondition, 0, len(rule.Conditions()))
	for _, cond := range rule.Conditions() {
		conditions = append(conditions, jsonCondition{
			Type:        cond.Type(),
			Operator:    cond.Operator(),
			Value:       cond.Value(),
			ReferenceId: cond.ReferenceIdRaw(),
		})
	}

	operations := make([]jsonOperation, 0, len(rule.OnMatch().Operations()))
	for _, op := range rule.OnMatch().Operations() {
		operations = append(operations, jsonOperation{
			Type:   op.Type(),
			Params: op.Params(),
		})
	}

	return jsonRule{
		Id:         rule.Id(),
		Conditions: conditions,
		OnMatch: jsonOutcome{
			Allow:      rule.OnMatch().Allow(),
			Operations: operations,
		},
	}
}

// MigrateTable creates or updates the portal_scripts table
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
