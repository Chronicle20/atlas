package script

import (
	"fmt"

	"github.com/Chronicle20/atlas-script-core/condition"
	"github.com/Chronicle20/atlas-script-core/operation"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	Resource = "map-scripts"
)

// RestModel represents the REST model for map scripts
type RestModel struct {
	Id          uuid.UUID       `json:"-"`
	ScriptName  string          `json:"scriptName"`
	ScriptType  string          `json:"scriptType"`
	Description string          `json:"description,omitempty"`
	Rules       []RestRuleModel `json:"rules"`
}

// RestRuleModel represents a rule in REST format
type RestRuleModel struct {
	Id         string               `json:"id"`
	Conditions []RestConditionModel `json:"conditions"`
	Operations []RestOperationModel `json:"operations"`
}

// RestConditionModel represents a condition in REST format
type RestConditionModel struct {
	Type        string `json:"type"`
	Operator    string `json:"operator"`
	Value       string `json:"value"`
	ReferenceId string `json:"referenceId,omitempty"`
}

// RestOperationModel represents an operation in REST format
type RestOperationModel struct {
	Type   string            `json:"type"`
	Params map[string]string `json:"params,omitempty"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id.String()
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid script ID: %w", err)
	}
	r.Id = id
	return nil
}

// GetReferences returns the resource references
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns the referenced IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns the referenced structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference ID
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a domain model to a REST model
func Transform(m MapScript) (RestModel, error) {
	restRules := make([]RestRuleModel, 0, len(m.Rules()))
	for _, rule := range m.Rules() {
		restRule := transformRule(rule)
		restRules = append(restRules, restRule)
	}

	return RestModel{
		ScriptName:  m.ScriptName(),
		ScriptType:  m.ScriptType(),
		Description: m.Description(),
		Rules:       restRules,
	}, nil
}

// transformRule converts a domain Rule to REST format
func transformRule(rule Rule) RestRuleModel {
	conditions := make([]RestConditionModel, 0, len(rule.Conditions()))
	for _, cond := range rule.Conditions() {
		conditions = append(conditions, RestConditionModel{
			Type:        cond.Type(),
			Operator:    cond.Operator(),
			Value:       cond.Value(),
			ReferenceId: cond.ReferenceIdRaw(),
		})
	}

	operations := make([]RestOperationModel, 0, len(rule.Operations()))
	for _, op := range rule.Operations() {
		operations = append(operations, RestOperationModel{
			Type:   op.Type(),
			Params: op.Params(),
		})
	}

	return RestRuleModel{
		Id:         rule.Id(),
		Conditions: conditions,
		Operations: operations,
	}
}

// Extract converts a REST model to a domain model
func Extract(r RestModel) (MapScript, error) {
	if r.ScriptName == "" {
		return MapScript{}, fmt.Errorf("scriptName is required")
	}
	if r.ScriptType == "" {
		return MapScript{}, fmt.Errorf("scriptType is required")
	}

	builder := NewMapScriptBuilder().
		SetScriptName(r.ScriptName).
		SetScriptType(r.ScriptType).
		SetDescription(r.Description)

	for _, restRule := range r.Rules {
		rule, err := extractRule(restRule)
		if err != nil {
			return MapScript{}, err
		}
		builder.AddRule(rule)
	}

	return builder.Build(), nil
}

// extractRule converts a REST rule to domain format
func extractRule(r RestRuleModel) (Rule, error) {
	rb := NewRuleBuilder().SetId(r.Id)

	for _, restCond := range r.Conditions {
		cond, err := extractCondition(restCond)
		if err != nil {
			return Rule{}, err
		}
		rb.AddCondition(cond)
	}

	for _, restOp := range r.Operations {
		op, err := extractOperation(restOp)
		if err != nil {
			return Rule{}, err
		}
		rb.AddOperation(op)
	}

	return rb.Build(), nil
}

// extractCondition converts a REST condition to domain format
func extractCondition(r RestConditionModel) (condition.Model, error) {
	builder := condition.NewBuilder().
		SetType(r.Type).
		SetOperator(r.Operator).
		SetValue(r.Value)

	if r.ReferenceId != "" {
		builder.SetReferenceId(r.ReferenceId)
	}

	return builder.Build()
}

// extractOperation converts a REST operation to domain format
func extractOperation(r RestOperationModel) (operation.Model, error) {
	builder := operation.NewBuilder().SetType(r.Type)

	if r.Params != nil {
		builder.SetParams(r.Params)
	}

	return builder.Build()
}
