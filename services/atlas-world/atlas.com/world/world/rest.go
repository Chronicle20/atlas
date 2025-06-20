package world

import (
	"atlas-world/channel"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id                 string              `json:"-"`
	Name               string              `json:"name"`
	Flag               int                 `json:"flag"`
	Message            string              `json:"message"`
	EventMessage       string              `json:"eventMessage"`
	Recommended        bool                `json:"recommended"`
	RecommendedMessage string              `json:"recommendedMessage"`
	CapacityStatus     uint32              `json:"capacityStatus"`
	Channels           []channel.RestModel `json:"-"`
}

func (r RestModel) GetName() string {
	return "worlds"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// GetReferences implements the jsonapi.MarshalReferences interface
func (r RestModel) GetReferences() []jsonapi.Reference {
	return []jsonapi.Reference{
		{
			Name:         "channels",
			Type:         "channels",
			Relationship: jsonapi.ToManyRelationship,
		},
	}
}

// GetReferencedIDs implements the jsonapi.MarshalLinkedRelations interface
func (r RestModel) GetReferencedIDs() []jsonapi.ReferenceID {
	var result []jsonapi.ReferenceID
	for _, c := range r.Channels {
		result = append(result, jsonapi.ReferenceID{
			ID:           c.GetID(),
			Type:         c.GetName(),
			Name:         "channels",
			Relationship: jsonapi.ToManyRelationship,
		})
	}
	return result
}

// GetReferencedStructs implements the jsonapi.MarshalIncludedRelations interface
func (r RestModel) GetReferencedStructs() []jsonapi.MarshalIdentifier {
	var result []jsonapi.MarshalIdentifier
	for i := range r.Channels {
		result = append(result, &r.Channels[i])
	}
	return result
}

func Transform(m Model) (RestModel, error) {
	cms, err := model.SliceMap(channel.Transform)(model.FixedProvider(m.Channels()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:                 strconv.Itoa(int(m.Id())),
		Name:               m.Name(),
		Flag:               getFlag(m.Flag()),
		Message:            m.Message(),
		EventMessage:       m.EventMessage(),
		Recommended:        m.RecommendedMessage() != "",
		RecommendedMessage: m.RecommendedMessage(),
		CapacityStatus:     m.CapacityStatus(),
		Channels:           cms,
	}, nil
}

// Extract converts a RestModel to a Model using the Builder pattern
func Extract(r RestModel) (Model, error) {
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		return Model{}, err
	}

	// Convert flag int back to string
	flagStr := getFlagString(r.Flag)

	cms, err := model.SliceMap(channel.Extract)(model.FixedProvider(r.Channels))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return NewBuilder().
		SetId(byte(id)).
		SetName(r.Name).
		SetFlag(flagStr).
		SetMessage(r.Message).
		SetEventMessage(r.EventMessage).
		SetRecommendedMessage(r.RecommendedMessage).
		SetCapacityStatus(r.CapacityStatus).
		SetChannels(cms).
		Build(), nil
}

// getFlagString converts a flag int back to its string representation
func getFlagString(flag int) string {
	switch flag {
	case 0:
		return "NOTHING"
	case 1:
		return "EVENT"
	case 2:
		return "NEW"
	case 3:
		return "HOT"
	default:
		return "NOTHING"
	}
}
