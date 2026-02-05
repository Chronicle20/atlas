package world

import (
	"atlas-channel/channel"
	worldConstants "github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/jtumidanski/api2go/jsonapi"
	"strconv"
)

type RestModel struct {
	Id                 string              `json:"-"`
	Name               string              `json:"name"`
	State              byte                `json:"state"`
	Message            string              `json:"message"`
	EventMessage       string              `json:"eventMessage"`
	Recommended        bool                `json:"recommended"`
	RecommendedMessage string              `json:"recommendedMessage"`
	CapacityStatus     uint16              `json:"capacityStatus"`
	Channels           []channel.RestModel `json:"-"`
	ExpRate            float64             `json:"expRate"`
	MesoRate           float64             `json:"mesoRate"`
	ItemDropRate       float64             `json:"itemDropRate"`
	QuestExpRate       float64             `json:"questExpRate"`
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

// SetToManyReferenceIDs implements the jsonapi.UnmarshalToManyRelations interface
func (r *RestModel) SetToManyReferenceIDs(name string, IDs []string) error {
	if name == "channels" {
		r.Channels = make([]channel.RestModel, len(IDs))
		for i, id := range IDs {
			r.Channels[i] = channel.RestModel{}
			err := r.Channels[i].SetID(id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func Transform(m Model) (RestModel, error) {
	cms, err := model.SliceMap(channel.Transform)(model.FixedProvider(m.Channels()))(model.ParallelMap())()
	if err != nil {
		return RestModel{}, err
	}

	return RestModel{
		Id:                 strconv.Itoa(int(m.Id())),
		Name:               m.Name(),
		State:              byte(m.State()),
		Message:            m.Message(),
		EventMessage:       m.EventMessage(),
		Recommended:        m.RecommendedMessage() != "",
		RecommendedMessage: m.RecommendedMessage(),
		CapacityStatus:     uint16(m.CapacityStatus()),
		Channels:           cms,
	}, nil
}

// Extract converts a RestModel to a Model using the Builder pattern
func Extract(r RestModel) (Model, error) {
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		return Model{}, err
	}

	cms, err := model.SliceMap(channel.Extract)(model.FixedProvider(r.Channels))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return NewBuilder().
		SetId(worldConstants.Id(id)).
		SetName(r.Name).
		SetState(State(r.State)).
		SetMessage(r.Message).
		SetEventMessage(r.EventMessage).
		SetRecommendedMessage(r.RecommendedMessage).
		SetCapacityStatus(Status(r.CapacityStatus)).
		SetChannels(cms).
		Build()
}
