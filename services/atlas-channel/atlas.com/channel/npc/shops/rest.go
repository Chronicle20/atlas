package shops

import (
	"atlas-channel/npc/shops/commodities"
	"fmt"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is a JSON API representation of the Model
type RestModel struct {
	Id          string                  `json:"id"`
	NpcId       uint32                  `json:"npcId"`
	Commodities []commodities.RestModel `json:"-"` // Commodities are now a relationship, not a direct attribute
}

// GetID to satisfy jsonapi.MarshalIdentifier interface
func (r RestModel) GetID() string {
	return r.Id
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface
func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetName to satisfy jsonapi.EntityNamer interface
func (r RestModel) GetName() string {
	return "shops"
}

// GetReferences to satisfy jsonapi.MarshalReferences interface
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "commodities",
			Name: "commodities",
		},
	}
}

// GetReferencedIDs to satisfy jsonapi.MarshalLinkedRelations interface
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, c := range r.Commodities {
		result = append(result, jsonapi.ReferenceID{
			ID:   c.GetID(),
			Type: "commodities",
			Name: "commodities",
		})
	}
	return result
}

// GetReferencedStructs to satisfy jsonapi.MarshalIncludedRelations interface
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for _, c := range r.Commodities {
		result = append(result, c)
	}
	return result
}

// SetToOneReferenceID to satisfy jsonapi.UnmarshalToOneRelations interface
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs to satisfy jsonapi.UnmarshalToManyRelations interface
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "commodities" {
		r.Commodities = make([]commodities.RestModel, 0)
		for _, id := range IDs {
			commodity := commodities.RestModel{}
			commodity.SetID(id)
			r.Commodities = append(r.Commodities, commodity)
		}
	}
	return nil
}

// SetReferencedStructs to satisfy jsonapi.UnmarshalIncludedRelations interface
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["commodities"]; ok {
		commodities := make([]commodities.RestModel, 0)
		for _, ri := range r.Commodities {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				commodities = append(commodities, wip)
			}
		}
		r.Commodities = commodities
	}
	return nil
}

// Transform converts a Model to a RestModel
func Transform(m Model) (RestModel, error) {
	commodityRest := make([]commodities.RestModel, 0)
	for _, c := range m.Commodities() {
		cr, err := commodities.Transform(c)
		if err != nil {
			return RestModel{}, err
		}
		commodityRest = append(commodityRest, cr)
	}

	return RestModel{
		Id:          fmt.Sprintf("shop-%d", m.NpcId()),
		NpcId:       m.NpcId(),
		Commodities: commodityRest,
	}, nil
}

// Extract converts a RestModel to a Model
func Extract(rm RestModel) (Model, error) {
	commodityModels := make([]commodities.Model, 0)
	for _, cr := range rm.Commodities {
		cm, err := commodities.Extract(cr)
		if err != nil {
			return Model{}, err
		}
		commodityModels = append(commodityModels, cm)
	}

	return NewModelBuilder().
		SetNpcId(rm.NpcId).
		SetCommodities(commodityModels).
		Build()
}

// CharacterListRestModel is a JSON API representation of characters in a shop
type CharacterListRestModel struct {
	Id string `json:"-"`
}

// GetID to satisfy jsonapi.MarshalIdentifier interface
func (r CharacterListRestModel) GetID() string {
	return r.Id
}

// GetName to satisfy jsonapi.EntityNamer interface
func (r CharacterListRestModel) GetName() string {
	return "characters"
}

// SetID to satisfy jsonapi.UnmarshalIdentifier interface
func (r *CharacterListRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// TransformCharacterList converts a list of character IDs to a CharacterListRestModel
func TransformCharacterList(characterId uint32) (CharacterListRestModel, error) {
	idStr := strconv.Itoa(int(characterId))
	return CharacterListRestModel{Id: idStr}, nil
}
