package drop

import (
	"strconv"
	"strings"

	"github.com/jtumidanski/api2go/jsonapi"
)

// ReactorRestModel is the JSON API representation of a reactor with its drops
type ReactorRestModel struct {
	Id    string          `json:"-"`
	Drops []DropRestModel `json:"-"`
}

func (r ReactorRestModel) GetID() string {
	return r.Id
}

func (r *ReactorRestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r ReactorRestModel) GetName() string {
	return "reactors"
}

func (r ReactorRestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Type: "drops",
			Name: "drops",
		},
	}
}

func (r ReactorRestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, v := range r.Drops {
		result = append(result, jsonapi.ReferenceID{
			ID:   v.GetID(),
			Type: v.GetName(),
			Name: v.GetName(),
		})
	}
	return result
}

func (r ReactorRestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for key := range r.Drops {
		result = append(result, r.Drops[key])
	}
	return result
}

func (r *ReactorRestModel) SetToOneReferenceID(_, _ string) error {
	return nil
}

func (r *ReactorRestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "drops" {
		for _, idStr := range IDs {
			r.Drops = append(r.Drops, DropRestModel{Id: idStr})
		}
	}
	return nil
}

func (r *ReactorRestModel) SetReferencedStructs(references map[string]map[string]jsonapi.Data) error {
	if refMap, ok := references["drops"]; ok {
		drops := make([]DropRestModel, 0)
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

// DropRestModel is the JSON API representation of a reactor drop
type DropRestModel struct {
	Id        string `json:"-"`
	ReactorId uint32 `json:"-"`
	ItemId    uint32 `json:"itemId"`
	QuestId   uint32 `json:"questId,omitempty"`
	Chance    uint32 `json:"chance"`
}

func (r DropRestModel) GetID() string {
	return r.Id
}

func (r *DropRestModel) SetID(id string) error {
	r.Id = id
	// Parse the ID to extract reactorId, itemId, and optionally questId
	// Format: "reactorId:itemId" or "reactorId:itemId:questId"
	parts := strings.Split(id, ":")
	if len(parts) >= 2 {
		reactorId, err := strconv.ParseUint(parts[0], 10, 32)
		if err == nil {
			r.ReactorId = uint32(reactorId)
		}
		itemId, err := strconv.ParseUint(parts[1], 10, 32)
		if err == nil {
			r.ItemId = uint32(itemId)
		}
		if len(parts) >= 3 {
			questId, err := strconv.ParseUint(parts[2], 10, 32)
			if err == nil {
				r.QuestId = uint32(questId)
			}
		}
	}
	return nil
}

func (r DropRestModel) GetName() string {
	return "drops"
}

// Extract converts a DropRestModel to a Model
func Extract(rm DropRestModel) Model {
	return Model{
		reactorId: rm.ReactorId,
		itemId:    rm.ItemId,
		questId:   rm.QuestId,
		chance:    rm.Chance,
	}
}

// DropPositionInputModel is the input for the drop position calculation request
type DropPositionInputModel struct {
	Id        uint32 `json:"-"`
	InitialX  int16  `json:"initialX"`
	InitialY  int16  `json:"initialY"`
	FallbackX int16  `json:"fallbackX"`
	FallbackY int16  `json:"fallbackY"`
}

func (r DropPositionInputModel) GetName() string {
	return "positions"
}

func (r DropPositionInputModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *DropPositionInputModel) SetID(id string) error {
	parsed, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(parsed)
	return nil
}

// PositionRestModel is the JSON API representation of a calculated position
type PositionRestModel struct {
	Id uint32 `json:"-"`
	X  int16  `json:"x"`
	Y  int16  `json:"y"`
}

func (r PositionRestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *PositionRestModel) SetID(id string) error {
	parsed, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	r.Id = uint32(parsed)
	return nil
}

func (r PositionRestModel) GetName() string {
	return "points"
}
