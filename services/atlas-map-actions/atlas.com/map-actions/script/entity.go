package script

import (
	"encoding/json"
	"time"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Entity represents a map script stored in the database
type Entity struct {
	ID         uuid.UUID      `gorm:"primaryKey;column:id;type:uuid"`
	TenantID   uuid.UUID      `gorm:"column:tenant_id;type:uuid;not null;index:idx_map_scripts_tenant_script,priority:1"`
	ScriptName string         `gorm:"column:script_name;not null;index:idx_map_scripts_tenant_script,priority:2"`
	ScriptType string         `gorm:"column:script_type;not null;index:idx_map_scripts_tenant_script,priority:3"`
	Data       string         `gorm:"column:data;type:jsonb;not null"`
	CreatedAt  time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt  time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the table name for the entity
func (Entity) TableName() string {
	return "map_scripts"
}

// jsonMapScript represents the JSON format of a map script
type jsonMapScript struct {
	ScriptName  string     `json:"scriptName"`
	ScriptType  string     `json:"scriptType,omitempty"`
	Description string     `json:"description,omitempty"`
	Rules       []jsonRule `json:"rules"`
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
}

// jsonOperation represents an operation in JSON format
type jsonOperation struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// Make converts an Entity to a MapScript model
func Make(e Entity) (MapScript, error) {
	var data jsonMapScript
	if err := json.Unmarshal([]byte(e.Data), &data); err != nil {
		return MapScript{}, err
	}

	builder := NewMapScriptBuilder().
		SetScriptName(data.ScriptName).
		SetScriptType(e.ScriptType).
		SetDescription(data.Description)

	for _, jr := range data.Rules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return MapScript{}, err
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

// ToEntity converts a MapScript model to an Entity
func ToEntity(m MapScript, tenantId uuid.UUID) (Entity, error) {
	jsonRules := make([]jsonRule, 0, len(m.Rules()))
	for _, rule := range m.Rules() {
		jr := convertRuleToJson(rule)
		jsonRules = append(jsonRules, jr)
	}

	data := jsonMapScript{
		ScriptName:  m.ScriptName(),
		ScriptType:  m.ScriptType(),
		Description: m.Description(),
		Rules:       jsonRules,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return Entity{}, err
	}

	return Entity{
		TenantID:   tenantId,
		ScriptName: m.ScriptName(),
		ScriptType: m.ScriptType(),
		Data:       string(jsonData),
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

// MigrateTable creates or updates the map_scripts table
func MigrateTable(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}
