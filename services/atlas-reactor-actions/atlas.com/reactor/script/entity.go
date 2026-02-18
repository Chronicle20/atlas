package script

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a reactor script stored in the database
type Entity struct {
	ID        uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID  uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;index:idx_reactor_scripts_tenant_reactor,priority:1"`
	ReactorID string         `gorm:"column:reactor_id;not null;index:idx_reactor_scripts_tenant_reactor,priority:2"`
	Data      string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the entity
func (Entity) TableName() string {
	return "reactor_scripts"
}

// jsonReactorScript represents the JSON format of a reactor script
type jsonReactorScript struct {
	ReactorId   string     `json:"reactorId"`
	Description string     `json:"description,omitempty"`
	HitRules    []jsonRule `json:"hitRules"`
	ActRules    []jsonRule `json:"actRules"`
}

// jsonRule represents a rule in JSON format
type jsonRule struct {
	Id         string          `json:"id"`
	Conditions []jsonCondition `json:"conditions"`
	Operations []jsonOperation `json:"operations"`
}

// jsonCondition represents a condition in JSON format
type jsonCondition struct {
	Type        string `json:"type"`
	Operator    string `json:"operator"`
	Value       string `json:"value"`
	ReferenceId string `json:"referenceId,omitempty"`
	Step        string `json:"step,omitempty"`
}

// jsonOperation represents an operation in JSON format
type jsonOperation struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// Make converts an Entity to a ReactorScript model
func Make(e Entity) (ReactorScript, error) {
	var data jsonReactorScript
	if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
		return ReactorScript{}, err
	}

	builder := NewReactorScriptBuilder().
		SetReactorId(data.ReactorId).
		SetDescription(data.Description)

	for _, jr := range data.HitRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return ReactorScript{}, err
		}
		builder.AddHitRule(rule)
	}

	for _, jr := range data.ActRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return ReactorScript{}, err
		}
		builder.AddActRule(rule)
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

	for _, jo := range jr.Operations {
		op, err := convertJsonOperation(jo)
		if err != nil {
			return Rule{}, err
		}
		rb.AddOperation(op)
	}

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

	if jc.Step != "" {
		builder.SetStep(jc.Step)
	}

	return builder.Build()
}

// convertJsonOperation converts a JSON operation to domain model
func convertJsonOperation(jo jsonOperation) (operation.Model, error) {
	builder := operation.NewBuilder().SetType(jo.Type)

	if jo.Params != nil {
		builder.SetParams(jo.Params)
	}

	return builder.Build()
}

// ToEntity converts a ReactorScript model to an Entity
func ToEntity(m ReactorScript, tenantId uuid.UUID) (Entity, error) {
	// Convert hit rules to JSON format
	jsonHitRules := make([]jsonRule, 0, len(m.HitRules()))
	for _, rule := range m.HitRules() {
		jr := convertRuleToJson(rule)
		jsonHitRules = append(jsonHitRules, jr)
	}

	// Convert act rules to JSON format
	jsonActRules := make([]jsonRule, 0, len(m.ActRules()))
	for _, rule := range m.ActRules() {
		jr := convertRuleToJson(rule)
		jsonActRules = append(jsonActRules, jr)
	}

	data := jsonReactorScript{
		ReactorId:   m.ReactorId(),
		Description: m.Description(),
		HitRules:    jsonHitRules,
		ActRules:    jsonActRules,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return Entity{}, err
	}

	return Entity{
		TenantID:  tenantId,
		ReactorID: m.ReactorId(),
		Data:      string(jsonData),
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
			Step:        cond.Step(),
		})
	}

	operations := make([]jsonOperation, 0, len(rule.Operations()))
	for _, op := range rule.Operations() {
		operations = append(operations, jsonOperation{
			Type:   op.Type(),
			Params: op.Params(),
		})
	}

	return jsonRule{
		Id:         rule.Id(),
		Conditions: conditions,
		Operations: operations,
	}
}

// MigrateTable creates or updates the reactor_scripts table
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
