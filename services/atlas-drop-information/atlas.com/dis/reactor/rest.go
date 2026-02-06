package reactor

import (
	"atlas-drops-information/reactor/drop"
	"fmt"
	"strconv"

	"github.com/jtumidanski/api2go/jsonapi"
)

// RestModel is a JSON API representation of a reactor with its drops
type RestModel struct {
	Id    string           `json:"-"`
	Drops []drop.RestModel `json:"-"` // Drops are a relationship, not a direct attribute
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
	return "reactors"
}

// GetReferences to satisfy jsonapi.MarshalReferences interface
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "drops",
			Name: "drops",
		},
	}
}

// GetReferencedIDs to satisfy jsonapi.MarshalLinkedRelations interface
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, d := range r.Drops {
		result = append(result, jsonapi.ReferenceID{
			ID:   d.GetID(),
			Type: "drops",
			Name: "drops",
		})
	}
	return result
}

// GetReferencedStructs to satisfy jsonapi.MarshalIncludedRelations interface
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for _, d := range r.Drops {
		result = append(result, d)
	}
	return result
}

// SetToOneReferenceID to satisfy jsonapi.UnmarshalToOneRelations interface
func (r *RestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

// SetToManyReferenceIDs to satisfy jsonapi.UnmarshalToManyRelations interface
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "drops" {
		r.Drops = make([]drop.RestModel, 0)
		for _, id := range IDs {
			d := drop.RestModel{}
			d.SetID(id)
			r.Drops = append(r.Drops, d)
		}
	}
	return nil
}

// SetReferencedStructs to satisfy jsonapi.UnmarshalIncludedRelations interface
func (r *RestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["drops"]; ok {
		drops := make([]drop.RestModel, 0)
		for _, ri := range r.Drops {
			if ref, ok := refMap[ri.GetID()]; ok {
				wip := ri
				err := jsonapi.ProcessIncludeData(&wip, ref, references)
				if err != nil {
					return err
				}
				drops = append(drops, wip)
			}
		}
		r.Drops = drops
	}
	return nil
}

// Transform converts reactor drop models to a ReactorRestModel
func Transform(reactorId uint32, drops []drop.Model) (RestModel, error) {
	dropRest := make([]drop.RestModel, 0)
	for _, d := range drops {
		dr, err := drop.Transform(d)
		if err != nil {
			return RestModel{}, err
		}
		dropRest = append(dropRest, dr)
	}

	return RestModel{
		Id:    fmt.Sprintf("%d", reactorId),
		Drops: dropRest,
	}, nil
}

// Extract converts a RestModel to reactor ID and drop data
func Extract(rm RestModel) (uint32, []drop.RestModel, error) {
	reactorId, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid reactor id: %w", err)
	}
	return uint32(reactorId), rm.Drops, nil
}
