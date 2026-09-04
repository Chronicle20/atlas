package area_info

import (
	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is the JSON:API resource for a character's area-info entry.
type RestModel struct {
	Id          string `json:"-"`
	CharacterId uint32 `json:"characterId"`
	Area        uint16 `json:"area"`
	Info        string `json:"info"`
}

func (r RestModel) GetName() string {
	return "area-infos"
}

func (r RestModel) GetID() string {
	return r.Id
}

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
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs sets to-many references
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs sets referenced structs
func (r *RestModel) SetReferencedStructs(_ map[string]map[string]jsonapi.Data) error {
	return nil
}

// Transform converts a domain Model to a RestModel
func Transform(m Model) (RestModel, error) {
	return RestModel{
		CharacterId: m.CharacterId(),
		Area:        m.Area(),
		Info:        m.Info(),
	}, nil
}

// Extract converts a RestModel to a domain Model.
func Extract(r RestModel) (Model, error) {
	return Model{
		characterId: r.CharacterId,
		area:        r.Area,
		info:        r.Info,
	}, nil
}
