package validation

import (
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

const (
	Resource = "validations"
)

// RestModel represents the REST model for validation requests and responses
type RestModel struct {
	Id          string   `json:"-"`
	CharacterId uint32   `json:"characterId"`
	Conditions  []string `json:"conditions"`
	Passed      bool     `json:"passed"`
	Details     []string `json:"details"`
}

// GetName returns the resource name
func (r RestModel) GetName() string {
	return Resource
}

// GetID returns the resource ID
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID
func (r *RestModel) SetID(id string) error {
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
		Id:          strconv.FormatUint(uint64(result.CharacterId()), 10),
		CharacterId: result.CharacterId(),
		Passed:      result.Passed(),
		Details:     result.Details(),
	}, nil
}
