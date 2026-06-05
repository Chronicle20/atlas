package consumable

import (
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is a partial view of atlas-data's "consumables" resource. Only the
// fields the cover resolver reads are declared; any other attributes in the
// response are ignored by the JSON:API unmarshaller.
type RestModel struct {
	Id          uint32 `json:"-"`
	MonsterBook bool   `json:"monsterBook"`
	MonsterId   uint32 `json:"monsterId"`
}

func (r RestModel) GetName() string { return "consumables" }

func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

// JSON:API reference interface methods. Required even though this resource has
// no relationships we consume: api2go.Unmarshal errors out walking any
// `relationships` block unless these exist (libs/atlas-rest/CLAUDE.md).
func (r RestModel) GetReferences() []jsonapi.Reference                { return []jsonapi.Reference{} }
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID           { return []jsonapi.ReferenceID{} }
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// Extract converts the wire model into the immutable domain Model.
func Extract(rm RestModel) (Model, error) {
	return Model{monsterBook: rm.MonsterBook, monsterId: rm.MonsterId}, nil
}
