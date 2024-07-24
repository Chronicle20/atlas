package continent

import (
	"atlas-dis/continent/drop"
	"github.com/manyminds/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	ID    string           `json:"-"`
	Drops []drop.RestModel `json:"drops"`
}

func (p RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Name: "drops",
			Type: "drops",
		},
	}
}

func (p RestModel) GetName() string {
	return "continents"
}

func (p RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, m := range p.Drops {
		result = append(result, jsonapi.ReferenceID{ID: m.GetID(), Name: "drops", Type: "drop"})
	}
	return result
}

func (p RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range p.Drops {
		result = append(result, p.Drops[key])
	}

	return result
}

func (p RestModel) GetID() string {
	return p.ID
}

func Transform(model Model) (RestModel, error) {
	rm := RestModel{
		ID:    strconv.Itoa(int(model.id)),
		Drops: make([]drop.RestModel, 0),
	}
	for _, m := range model.drops {
		rm.Drops = append(rm.Drops, drop.Transform(m))
	}
	return rm, nil
}
