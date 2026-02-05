package transport

import (
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is the JSON:API resource for transport routes
type RestModel struct {
	Id         string  `json:"-"`
	Name       string  `json:"name"`
	State      string  `json:"state"`
	StartMapId _map.Id `json:"startMapId"`
}

// GetName returns the resource name for JSON:API
func (r RestModel) GetName() string {
	return "routes"
}

// GetID returns the resource ID for JSON:API
func (r RestModel) GetID() string {
	return r.Id
}

// SetID sets the resource ID from JSON:API
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetReferences returns JSON:API references
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{}
}

// GetReferencedIDs returns JSON:API referenced IDs
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	return []jsonapi.ReferenceID{}
}

// GetReferencedStructs returns JSON:API referenced structs
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	return []jsonapi.MarshalIdentifier{}
}

// SetToOneReferenceID sets a to-one reference
func (r *RestModel) SetToOneReferenceID(name, ID string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many references
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a domain Model to a RestModel
func Transform(m Model) RestModel {
	return RestModel{
		Id:         m.Id().String(),
		Name:       m.Name(),
		State:      m.State(),
		StartMapId: m.StartMapId(),
	}
}

// Extract converts a RestModel to a domain Model
func Extract(r RestModel) (Model, error) {
	id, err := uuid.Parse(r.Id)
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:         id,
		name:       r.Name,
		state:      r.State,
		startMapId: r.StartMapId,
	}, nil
}
