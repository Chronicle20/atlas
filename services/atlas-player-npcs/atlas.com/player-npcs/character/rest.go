package character

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
)

// RestModel mirrors the subset of atlas-character's "characters" resource
// that a deploy snapshot needs (design.md §6.1): appearance, identity and
// the equipped compartment behind the `include=inventory` relationship.
type RestModel struct {
	Id        uint32 `json:"-"`
	Name      string `json:"name"`
	Gender    byte   `json:"gender"`
	SkinColor byte   `json:"skinColor"`
	Face      uint32 `json:"face"`
	Hair      uint32 `json:"hair"`
	JobId     job.Id `json:"jobId"`
	Level     byte   `json:"level"`
	Gm        int    `json:"gm"`

	// Equipment is populated from the "equipment" included block by
	// SetReferencedStructs below; it carries no JSON tag of its own since
	// it never appears as a flat attribute.
	Equipment []EquippedItemRestModel `json:"-"`
}

// EquippedItemRestModel is the decoded shape of an item in the "equipment"
// included resource: slot/templateId, matching the "assets" shape used
// elsewhere in the codebase for equipped items (e.g.
// services/atlas-channel/atlas.com/channel/asset/rest.go).
type EquippedItemRestModel struct {
	Slot       int16  `json:"slot"`
	TemplateId uint32 `json:"templateId"`
}

func (r RestModel) GetName() string {
	return "characters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}

	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "equipment",
			Name: "equipment",
		},
	}
}

func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	return result
}

func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	return result
}

func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error {
	return nil
}

// SetReferencedStructs decodes the "equipment" included block, if present,
// into Equipment. A response with no included block (or no equipment type
// within it) leaves Equipment nil rather than erroring — an unequipped
// character is a normal deploy input.
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	items, ok := references["equipment"]
	if !ok {
		return nil
	}

	equipped := make([]EquippedItemRestModel, 0, len(items))
	for _, data := range items {
		var attrs EquippedItemRestModel
		if err := json.Unmarshal(data.Attributes, &attrs); err != nil {
			return err
		}
		equipped = append(equipped, attrs)
	}
	sort.Slice(equipped, func(i, j int) bool { return equipped[i].Slot < equipped[j].Slot })

	r.Equipment = equipped
	return nil
}

func Extract(rm RestModel) (Model, error) {
	equipment := make([]EquippedItem, 0, len(rm.Equipment))
	for _, e := range rm.Equipment {
		equipment = append(equipment, EquippedItem{slot: e.Slot, templateId: e.TemplateId})
	}
	return Model{
		id:        rm.Id,
		name:      rm.Name,
		gender:    rm.Gender,
		skinColor: rm.SkinColor,
		face:      rm.Face,
		hair:      rm.Hair,
		jobId:     rm.JobId,
		level:     rm.Level,
		gm:        rm.Gm == 1,
		equipment: equipment,
	}, nil
}
