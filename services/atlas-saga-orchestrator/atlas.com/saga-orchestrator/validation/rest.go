package validation

import (
	"fmt"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

const (
	Resource = "validations"
)

// RestModel represents the REST model for validation requests and responses
type RestModel struct {
	Id         uint32            `json:"-"`
	Conditions []ConditionInput  `json:"conditions,omitempty"`
	Passed     bool              `json:"passed"`
	Results    []ConditionResult `json:"results,omitempty"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
// For validation results, the character ID is used as the resource ID
func (r RestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

// SetID sets the resource ID
// For validation requests, the ID is parsed and set as the character ID
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
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many reference IDs
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a domain model to a REST model
func Transform(result ValidationResult) (RestModel, error) {
	return RestModel{
		Id:      result.CharacterId(),
		Passed:  result.Passed(),
		Results: result.Results(),
	}, nil
}

// Extract converts a REST model to domain model parameters for structured validation
func Extract(rm RestModel) (uint32, []ConditionInput, error) {
	// Validate that CharacterId is provided
	if rm.Id == 0 {
		return 0, nil, fmt.Errorf("Id is required")
	}

	// Validate that at least one condition is provided
	if len(rm.Conditions) == 0 {
		return 0, nil, fmt.Errorf("at least one condition is required")
	}

	return rm.Id, rm.Conditions, nil
}
