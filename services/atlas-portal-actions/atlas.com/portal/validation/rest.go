package validation

import (
	"fmt"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

const (
	Resource = "validations"
)

// RestModel represents the REST model for validation requests and responses
type RestModel struct {
	Id         uint32           `json:"-"`
	Conditions []ConditionInput `json:"conditions,omitempty"`
	Passed     bool             `json:"passed"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

// SetID sets the resource ID
func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid character ID: %w", err)
	}
	r.Id = uint32(id)
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

// Transform converts a domain ValidationResult to a REST model
func Transform(result ValidationResult) (RestModel, error) {
	return RestModel{
		Id:     result.CharacterId(),
		Passed: result.Passed(),
	}, nil
}

// Extract converts a REST model to domain model
func Extract(rm RestModel) (ValidationResult, error) {
	return NewValidationResult(rm.Id, rm.Passed), nil
}
