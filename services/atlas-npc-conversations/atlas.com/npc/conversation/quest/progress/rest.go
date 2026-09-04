package progress

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel represents a single quest progress entry from atlas-quest's
// GET /characters/{characterId}/quests/{questId}/progress collection.
type RestModel struct {
	Id         uint32 `json:"-"`
	InfoNumber uint32 `json:"infoNumber"`
	Progress   string `json:"progress"`
}

// GetName returns the JSON:API type name
func (r RestModel) GetName() string {
	return "progress"
}

// GetID returns the JSON:API resource ID
func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

// SetID sets the JSON:API resource ID
func (r *RestModel) SetID(strId string) error {
	if strId == "" {
		r.Id = 0
		return nil
	}

	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
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

// Extract converts a RestModel into a Model
func Extract(rm RestModel) (Model, error) {
	return Model{
		infoNumber: rm.InfoNumber,
		progress:   rm.Progress,
	}, nil
}
